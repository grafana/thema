package thema

import (
	"fmt"
	"sort"

	"cuelang.org/go/cue"
	cerrors "cuelang.org/go/cue/errors"
)

var (
	_ Lineage                     = &baseLineage{}
	_ ConvergentLineage[Assignee] = &unaryConvLineage[Assignee]{}
)

// A baseLineage is a Go facade over a valid CUE lineage that does not compose
// other lineage.
type baseLineage struct {
	rt *Runtime

	// internal flag to ensure BindLineage is only mechanism to create
	validated bool

	// name of the lineage, #Lineage.name
	name string

	// original raw input cue.Value containing lineage definition
	raw cue.Value

	// input cue.Value, unified with thema.#Lineage
	uni cue.Value

	// all schema versions in the lineage
	allv []SyntacticVersion

	// all the schemas
	allsch []*schemaDef
}

// BindLineage takes a raw [cue.Value], checks that it correctly follows Thema's
// invariants, such as translatability and backwards compatibility version
// numbering. If checks succeed, a [Lineage] is returned.
//
// This function is the only way to create non-nil Lineage objects. As a result,
// all non-nil instances of Lineage in any Go program are guaranteed to follow
// Thema invariants.
func BindLineage(raw cue.Value, rt *Runtime, opts ...BindOption) (Lineage, error) {
	// We could be more selective than this, but this isn't supposed to be forever, soooooo
	rt.l()
	defer rt.u()

	cfg := &bindConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	lindef := rt.linDef()
	ml := &maybeLineage{
		rt:  rt,
		raw: raw,
		uni: lindef.Unify(raw),
		cfg: cfg,
	}

	if err := ml.checkExists(cfg); err != nil {
		return nil, err
	}
	if err := ml.checkLineageShape(cfg); err != nil {
		return nil, err
	}
	if err := ml.checkNativeValidity(cfg); err != nil {
		return nil, err
	}
	if err := ml.checkGoValidity(cfg); err != nil {
		return nil, err
	}

	// previously verified that this value is concrete
	nam, _ := raw.LookupPath(cue.MakePath(cue.Str("name"))).String()

	lin := &baseLineage{
		validated: true,
		rt:        rt,
		name:      nam,
		raw:       ml.raw,
		uni:       ml.uni,
		allsch:    ml.schlist,
		allv:      ml.allv,
	}

	for _, sch := range lin.allsch {
		sch.lin = lin
	}
	return lin, nil
}

func isValidLineage(lin Lineage) {
	switch tlin := lin.(type) {
	case nil:
		panic("nil lineage")
	case *baseLineage:
		if !tlin.validated {
			panic("lineage not validated")
		}
	default:
		panic("unreachable")
	}
}

func getLinLib(lin Lineage) *Runtime {
	switch tlin := lin.(type) {
	case *baseLineage:
		return tlin.rt
	default:
		panic("unreachable")
	}
}

func mkerror(val cue.Value, format string, args ...any) error {
	s := val.Source()
	if s == nil {
		return fmt.Errorf(format, args...)
	}
	return cerrors.Newf(s.Pos(), format, args...)
}

// Runtime returns the thema.Runtime instance with which this lineage was built.
func (lin *baseLineage) Runtime() *Runtime {
	return lin.rt
}

// Latest returns the newest Schema in the lineage - largest minor version
// within the largest major version.
func (lin *baseLineage) Latest() Schema {
	return lin.allsch[len(lin.allsch)-1]
}

// First returns the first Schema in the lineage (v0.0). Thema requires that all
// valid lineages contain at least one schema, so this is guaranteed to exist.
func (lin *baseLineage) First() Schema {
	return lin.allsch[0]
}

// Underlying returns the cue.Value of the entire lineage.
func (lin *baseLineage) Underlying() cue.Value {
	isValidLineage(lin)

	return lin.raw
}

// Name returns the name of the object schematized by the lineage, as declared in
// the lineage's name field.
func (lin *baseLineage) Name() string {
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
func (lin *baseLineage) ValidateAny(data cue.Value) *Instance {
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
func (lin *baseLineage) Schema(v SyntacticVersion) (Schema, error) {
	isValidLineage(lin)

	if !synvExists(lin.allv, v) {
		return nil, &ErrNoSchemaWithVersion{
			lin: lin,
			v:   v,
		}
	}

	return lin.schema(v), nil
}

func (lin *baseLineage) allVersions() versionList {
	return lin.allv
}

func (lin *baseLineage) schema(v SyntacticVersion) *schemaDef {
	return lin.allsch[searchSynv(lin.allv, v)]
}

func (lin *baseLineage) _lineage() {}

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
