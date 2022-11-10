package thema

import (
	"fmt"
	"sort"
	"strings"

	"cuelang.org/go/cue"
	terrors "github.com/grafana/thema/errors"
)

var (
	_ Lineage                     = &UnaryLineage{}
	_ ConvergentLineage[Assignee] = &unaryConvLineage[Assignee]{}
)

// A UnaryLineage is a Go facade over a valid CUE lineage that does not compose
// other lineage.
type UnaryLineage struct {
	validated bool
	name      string
	raw       cue.Value
	rt        *Runtime
	allv      []SyntacticVersion
	allsch    []*UnarySchema
}

func defPathFor(name string, v SyntacticVersion) cue.Path {
	return cue.MakePath(cue.Def(fmt.Sprintf("%s%v%v", name, v[0], v[1])))
}

// BindLineage takes a raw cue.Value, checks that it is a valid lineage (that it
// upholds the invariants which undergird Thema's translatability guarantees),
// and returns the cue.Value wrapped in a Lineage, iff validity checks succeed.
// The Lineage type provides access to all the types and functions for working
// with Thema in Go.
//
// This function is the sole intended mechanism for creating Lineage objects,
// thereby providing a practical promise that all instances of Lineage uphold
// Thema's invariants. It is primarily intended for use by authors of lineages
// in the creation of a LineageFactory.
func BindLineage(raw cue.Value, rt *Runtime, opts ...BindOption) (Lineage, error) {
	// We could be more selective than this, but this isn't supposed to be forever, soooooo
	rt.l()
	defer rt.u()

	p := raw.Path().String()
	// The candidate lineage must exist.
	if !raw.Exists() {
		if p != "" {
			return nil, fmt.Errorf("%w: path was %q", terrors.ErrValueNotExist, p)
		}

		return nil, terrors.ErrValueNotExist
	}
	if p == "" {
		p = "instance root"
	}

	// The candidate lineage must be error-free.
	if err := raw.Validate(cue.Concrete(false)); err != nil {
		return nil, err
	}

	// The candidate lineage must be an instance of #Lineage.
	dlin := rt.linDef()
	err := dlin.Subsume(raw, cue.Raw(), cue.Schema(), cue.Final())
	if err != nil {
		// FIXME figure out how to wrap both the sentinel and CUE error sanely
		return nil, fmt.Errorf("%w (%s): %s", terrors.ErrValueNotALineage, p, err)
	}

	nam, err := raw.LookupPath(cue.MakePath(cue.Str("name"))).String()
	if err != nil {
		return nil, fmt.Errorf("%w (%s): name field is not concrete", terrors.ErrInvalidLineage, p)
	}

	cfg := &bindConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	lin := &UnaryLineage{
		validated: true,
		raw:       raw,
		rt:        rt,
		name:      nam,
	}

	// Populate the version list and enforce compat/subsumption invariants
	seqiter, _ := raw.LookupPath(cue.MakePath(cue.Str("seqs"))).List()
	var seqv uint
	var predecessor cue.Value
	var predsv SyntacticVersion
	for seqiter.Next() {
		var schv uint
		schemas := seqiter.Value().LookupPath(cue.MakePath(cue.Str("schemas")))

		schiter, _ := schemas.List()
		for schiter.Next() {
			v := synv(seqv, schv)
			lin.allv = append(lin.allv, v)

			sch := schiter.Value()
			defpath := cue.MakePath(cue.Def(fmt.Sprintf("%s%v%v", sanitizeLabelString(nam), v[0], v[1])))
			defsch := rt.Underlying().FillPath(defpath, sch).LookupPath(defpath)
			if defsch.Validate() != nil {
				panic(defsch.Validate())
			}
			lin.allsch = append(lin.allsch, &UnarySchema{
				raw:    sch,
				defraw: defsch,
				lin:    lin,
				v:      v,
			})

			// No predecessor to compare against with the very first schema
			if !(schv == 0 && seqv == 0) {
				// TODO Marked as buggy until we figure out how to both _not_ require
				// schema to be closed in the .cue file, _and_ how to detect default changes
				if !cfg.skipbuggychecks {
					// The sequences and schema in the candidate lineage must follow
					// backwards [in]compatibility rules.
					// TODO Subsumption may not be what we actually want to check here,
					// as it does not allow the addition of required fields with defaults
					bcompat := sch.Subsume(predecessor, cue.Raw(), cue.Schema(), cue.Definitions(true), cue.All(), cue.Final())
					if (schv == 0 && bcompat == nil) || (schv != 0 && bcompat != nil) {
						return nil, &compatInvariantError{
							rawlin:    raw,
							violation: [2]SyntacticVersion{predsv, v},
							detail:    bcompat,
						}
					}
				}
			}

			predecessor = sch
			predsv = v
			schv++
		}
		seqv++
	}

	return lin, nil
}

func sanitizeLabelString(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			fallthrough
		case r >= 'A' && r <= 'Z':
			fallthrough
		case r >= '0' && r <= '9':
			fallthrough
		case r == '_':
			return r
		default:
			return -1
		}
	}, s)
}

// Runtime returns the thema.Runtime instance with which this lineage was built.
func (lin *UnaryLineage) Runtime() *Runtime {
	return lin.rt
}

// Latest returns the newest Schema in the lineage - largest minor version
// within the largest major version.
func (lin *UnaryLineage) Latest() Schema {
	return lin.allsch[len(lin.allsch)-1]
}

// First returns the first Schema in the lineage (v0.0). Thema requires that all
// valid lineages contain at least one schema, so this is guaranteed to exist.
func (lin *UnaryLineage) First() Schema {
	return lin.allsch[0]
}

func isValidLineage(lin Lineage) {
	switch tlin := lin.(type) {
	case nil:
		panic("nil lineage")
	case *UnaryLineage:
		if !tlin.validated {
			panic("lineage not validated")
		}
	default:
		panic("unreachable")
	}
}

func getLinLib(lin Lineage) *Runtime {
	switch tlin := lin.(type) {
	case *UnaryLineage:
		return tlin.rt
	default:
		panic("unreachable")
	}
}

// Underlying returns the cue.Value of the entire lineage.
func (lin *UnaryLineage) Underlying() cue.Value {
	isValidLineage(lin)

	return lin.raw
}

// Name returns the name of the object schematized by the lineage, as declared in
// the lineage's name field.
func (lin *UnaryLineage) Name() string {
	isValidLineage(lin)

	if !lin.validated {
		panic("lineage not validated")
	}
	return lin.name
}

// ValidateAny checks that the provided data is valid with respect to at
// least one of the schemas in the lineage. The oldest (smallest) schema against
// which the data validates is chosen. A nil return indicates no validating
// schema was found.
//
// While this method takes a cue.Value, this is only to avoid having to trigger
// the translation internally; input values must be concrete. To use
// incomplete CUE values with Thema schemas, prefer working directly in CUE,
// or if you must, rely on Underlying().
//
// TODO should this instead be interface{} (ugh ugh wish Go had tagged unions) like FillPath?
func (lin *UnaryLineage) ValidateAny(data cue.Value) *Instance {
	isValidLineage(lin)

	for sch := lin.schema(synv()); sch != nil; sch = sch.successor() {
		if inst, err := sch.Validate(data); err == nil {
			return inst
		}
	}
	return nil
}

// Schema returns the schema identified by the provided version, if one exists.
//
// Only the [0, 0] schema is guaranteed to exist in all valid lineages.
func (lin *UnaryLineage) Schema(v SyntacticVersion) (Schema, error) {
	isValidLineage(lin)

	if !synvExists(lin.allv, v) {
		return nil, &ErrNoSchemaWithVersion{
			lin: lin,
			v:   v,
		}
	}

	return lin.schema(v), nil
}

func (lin *UnaryLineage) schema(v SyntacticVersion) *UnarySchema {
	return lin.allsch[searchSynv(lin.allv, v)]
}

func (lin *UnaryLineage) _lineage() {}

func searchSynv(a []SyntacticVersion, x SyntacticVersion) int {
	return sort.Search(len(a), func(i int) bool { return !a[i].Less(x) })
}

func synvExists(a []SyntacticVersion, x SyntacticVersion) bool {
	i := searchSynv(a, x)
	return i < len(a) && a[i] == x
}

type unaryConvLineage[T Assignee] struct {
	Lineage
	tsch TypedSchema[T]
}

func (lin *unaryConvLineage[T]) TypedSchema() TypedSchema[T] {
	return lin.tsch
}
