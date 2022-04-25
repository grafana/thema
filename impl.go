package thema

import (
	"fmt"
	"sort"

	"cuelang.org/go/cue"
	terrors "github.com/grafana/thema/errors"
)

// ErrValueNotExist indicates that an operation failed because a provided
// cue.Value does not exist.
type ErrValueNotExist struct {
	path string
}

func (e *ErrValueNotExist) Error() string {
	return fmt.Sprintf("value from path %q does not exist, absent values cannot be lineages", e.path)
}

// ErrNoSchemaWithVersion indicates that an operation was requested against a
// schema version that does not exist within a particular lineage.
type ErrNoSchemaWithVersion struct {
	lin Lineage
	v   SyntacticVersion
}

func (e *ErrNoSchemaWithVersion) Error() string {
	return fmt.Sprintf("lineage %q does not contain a schema with version %v", e.lin.Name(), e.v)
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
func BindLineage(raw cue.Value, lib Library, opts ...BindOption) (Lineage, error) {
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
	dlin := lib.linDef()
	if err := dlin.Subsume(raw, cue.Raw(), cue.Schema(), cue.Final()); err != nil {
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
		lib:       lib,
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
			defpath := cue.MakePath(cue.Def(fmt.Sprintf("%s%v%v", nam, v[0], v[1])))
			defsch := lib.UnwrapCUE().FillPath(defpath, sch).LookupPath(defpath)
			if defsch.Err() != nil {
				panic(defsch.Err())
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

func getLinLib(lin Lineage) Library {
	switch tlin := lin.(type) {
	case *UnaryLineage:
		return tlin.lib
	default:
		panic("unreachable")
	}
}

type compatInvariantError struct {
	rawlin    cue.Value
	violation [2]SyntacticVersion
	detail    error
}

func (e *compatInvariantError) Error() string {
	if e.violation[0][0] == e.violation[1][0] {
		// TODO better
		return e.detail.Error()
	}
	return fmt.Sprintf("schema %s must be backwards incompatible with schema %s", e.violation[0], e.violation[1])
}

// A UnaryLineage is a Go facade over a valid CUE lineage that does not compose
// other lineage.
type UnaryLineage struct {
	validated bool
	name      string
	// schmap    sync.Map
	raw    cue.Value
	lib    Library
	allv   []SyntacticVersion
	allsch []*UnarySchema
}

var _ Lineage = &UnaryLineage{}

// UnwrapCUE returns the cue.Value of the entire lineage.
func (lin *UnaryLineage) UnwrapCUE() cue.Value {
	isValidLineage(lin)

	if !lin.validated {
		panic("lineage not validated")
	}
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
// or if you must, rely on UnwrapCUE().
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

// lazy approach, uses sync.Map
// func (lin *UnaryLineage) schemam(v SyntacticVersion) *UnarySchema {
// 	isch, ok := lin.schmap.Load(v)
// 	if !ok {
// 		schval, err := cueArgs{
// 			"v":   v,
// 			"lin": lin.UnwrapCUE(),
// 		}.call("#Pick", lin.lib)
// 		if err != nil {
// 			panic(err)
// 		}
// 		sch := &UnarySchema{
// 			raw: schval,
// 			lin: lin,
// 			v:   v,
// 		}
// 		isch, _ = lin.schmap.LoadOrStore(v, sch)
// 	}
// 	return isch.(*UnarySchema)
// }

func (lin *UnaryLineage) _lineage() {}

func searchSynv(a []SyntacticVersion, x SyntacticVersion) int {
	return sort.Search(len(a), func(i int) bool { return !a[i].Less(x) })
}

func synvExists(a []SyntacticVersion, x SyntacticVersion) bool {
	i := searchSynv(a, x)
	return i < len(a) && a[i] == x
}

// A UnarySchema is a Go facade over a Thema schema that does not compose any
// schemas from any other lineages.
type UnarySchema struct {
	raw    cue.Value
	defraw cue.Value
	lin    *UnaryLineage
	v      SyntacticVersion
}

var _ Schema = &UnarySchema{}

// Validate checks that the provided data is valid with respect to the
// schema. If valid, the data is wrapped in an Instance and returned.
// Otherwise, a nil Instance is returned along with an error detailing the
// validation failure.
//
// While Validate takes a cue.Value, this is only to avoid having to trigger
// the translation internally; input values must be concrete. To use
// incomplete CUE values with Thema schemas, prefer working directly in CUE,
// or if you must, rely on UnwrapCUE().
//
// TODO should this instead be interface{} (ugh ugh wish Go had discriminated unions) like FillPath?
func (sch *UnarySchema) Validate(data cue.Value) (*Instance, error) {
	// TODO which approach is actually the right one, unify or subsume? ugh
	// err := sch.raw.Subsume(data, cue.Concrete(true), cue.Final(), cue.All())
	// if err != nil {
	// 	return nil, mungeValidateErr(err, sch)
	// }

	x := sch.defraw.Unify(data)
	if err := x.Validate(cue.Concrete(true), cue.Final(), cue.All()); err != nil {
		return nil, mungeValidateErr(err, sch)
	}

	return &Instance{
		raw:  data,
		sch:  sch,
		name: "", // FIXME how are we getting this out?
	}, nil
}

// Successor returns the next schema in the lineage, or nil if it is the last schema.
func (sch *UnarySchema) Successor() Schema {
	if s := sch.successor(); s != nil {
		return s
	}
	return nil
}

func (sch *UnarySchema) successor() *UnarySchema {
	if sch.lin.allv[len(sch.lin.allv)-1] == sch.v {
		return nil
	}

	succv := sch.lin.allv[searchSynv(sch.lin.allv, sch.v)+1]
	return sch.lin.schema(succv)
}

// Predecessor returns the previous schema in the lineage, or nil if it is the first schema.
func (sch *UnarySchema) Predecessor() Schema {
	if s := sch.predecessor(); s != nil {
		return s
	}
	return nil
}

func (sch *UnarySchema) predecessor() *UnarySchema {
	if sch.v == synv() {
		return nil
	}

	predv := sch.lin.allv[searchSynv(sch.lin.allv, sch.v)-1]
	return sch.lin.schema(predv)
}

// LatestVersionInSequence returns the version number of the newest (largest) schema
// version in the provided sequence number.
//
// An error indicates the number of the provided sequence does not exist.
func (sch *UnarySchema) LatestVersionInSequence() SyntacticVersion {
	// Lineage invariants preclude an error
	sv, _ := LatestVersionInSequence(sch.lin, sch.v[0])
	return sv
}

// UnwrapCUE returns the cue.Value that represents the underlying CUE schema.
func (sch *UnarySchema) UnwrapCUE() cue.Value {
	return sch.raw
}

// Version returns the schema's version number.
func (sch *UnarySchema) Version() SyntacticVersion {
	return sch.v
}

// Lineage returns the lineage that contains this schema.
func (sch *UnarySchema) Lineage() Lineage {
	return sch.lin
}

func (sch *UnarySchema) defPathFor() cue.Path {
	return defPathFor(sch.lin.Name(), sch.v)
}

func (sch *UnarySchema) _schema() {}

// Call with no args to get init v, {0, 0}
// Call with one to get first version in a seq, {x, 0}
// Call with two because smooth brackets are prettier than curly
// Call with three or more because len(synv) < len(panic)
func synv(v ...uint) SyntacticVersion {
	switch len(v) {
	case 0:
		return SyntacticVersion{0, 0}
	case 1:
		return SyntacticVersion{v[0], 0}
	case 2:
		return SyntacticVersion{v[0], v[1]}
	default:
		panic("cmon")
	}
}

func tosynv(v cue.Value) SyntacticVersion {
	var sv SyntacticVersion
	if err := v.Decode(&sv); err != nil {
		panic(err)
	}
	return sv
}
