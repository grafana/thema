package thema

import (
	"fmt"
	"sort"

	"cuelang.org/go/cue"
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
	// The candidate lineage must exist.
	if !raw.Exists() {
		return nil, &ErrValueNotExist{
			path: raw.Path().String(),
		}
	}

	// The candidate lineage must be error-free.
	if err := raw.Validate(cue.Concrete(false)); err != nil {
		return nil, err
	}

	// The candidate lineage must be an instance of #Lineage.
	dlin := lib.linDef()
	if err := dlin.Subsume(raw, cue.Raw(), cue.Schema()); err != nil {
		return nil, err
	}

	cfg := &bindConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	if !cfg.skipbuggychecks {
		// The sequences and schema in the candidate lineage must follow
		// backwards [in]compatibility rules.
		if err := verifySeqCompatInvariants(raw, lib); err != nil {
			return nil, err
		}
	}

	lin := &UnaryLineage{
		validated: true,
		raw:       raw,
		lib:       lib,
	}

	lin.first, _ = Pick(lin, synv())
	allv, err := cueArgs{
		"lin": raw,
	}.call("_allv", lib)
	if err != nil {
		// This can't happen without a name change or something
		panic(err)
	}
	_ = allv.Decode(&lin.allv)

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
	panic("TODO")
}

// Assumes that lin has already been verified to be subsumed by #Lineage
func verifySeqCompatInvariants(lin cue.Value, lib Library) error {
	seqiter, _ := lin.LookupPath(cue.MakePath(cue.Str("seqs"))).List()
	var seqv uint
	var predecessor cue.Value
	var predsv SyntacticVersion
	for seqiter.Next() {
		var schv uint
		schemas := seqiter.Value().LookupPath(cue.MakePath(cue.Str("schemas")))
		schiter, _ := schemas.List()
		for schiter.Next() {
			if schv == 0 && seqv == 0 {
				// Very first schema, no predecessor to compare against
				continue
			}

			sch := schiter.Value()
			bcompat := sch.Subsume(predecessor, cue.Raw(), cue.Schema())
			if (schv == 0 && bcompat == nil) || (schv != 0 && bcompat != nil) {
				return &compatInvariantError{
					rawlin:    lin,
					violation: [2]SyntacticVersion{predsv, {seqv, schv}},
					detail:    bcompat,
				}
			}

			predecessor = sch
			predsv = SyntacticVersion{seqv, schv}
			schv++
		}
		seqv++
	}

	return nil
}

// A UnaryLineage is a Go facade over a valid CUE lineage that does not compose
// other lineage.
type UnaryLineage struct {
	validated bool
	name      string
	first     Schema
	raw       cue.Value
	lib       Library
	allv      []SyntacticVersion
}

// RawValue returns the cue.Value of the entire lineage.
func (lin *UnaryLineage) RawValue() cue.Value {
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
// or if you must, rely on the RawValue().
//
// TODO should this instead be interface{} (ugh ugh wish Go had tagged unions) like FillPath?
func (lin *UnaryLineage) ValidateAny(data cue.Value) *Instance {
	isValidLineage(lin)

	for sch := lin.first; sch != nil; sch.Successor() {
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

	schval, err := cueArgs{
		"v":   v,
		"lin": lin.RawValue(),
	}.call("#Pick", lin.lib)
	if err != nil {
		return nil, err
	}

	return &UnarySchema{
		raw: schval,
		lin: lin,
		v:   v,
	}, nil
}

func (lin *UnaryLineage) _lineage() {}

func searchSynv(a []SyntacticVersion, x SyntacticVersion) int {
	return sort.Search(len(a), func(i int) bool { return !a[i].less(x) })
}

func synvExists(a []SyntacticVersion, x SyntacticVersion) bool {
	i := searchSynv(a, x)
	return i < len(a) && a[i] == x
}

// A UnarySchema is a Go facade over a Thema schema that does not compose any
// schemas from any other lineages.
type UnarySchema struct {
	raw cue.Value
	lin *UnaryLineage
	v   SyntacticVersion
}

// Validate checks that the provided data is valid with respect to the
// schema. If valid, the data is wrapped in an Instance and returned.
// Otherwise, a nil Instance is returned along with an error detailing the
// validation failure.
//
// While Validate takes a cue.Value, this is only to avoid having to trigger
// the translation internally; input values must be concrete. To use
// incomplete CUE values with Thema schemas, prefer working directly in CUE,
// or if you must, rely on the RawValue().
//
// TODO should this instead be interface{} (ugh ugh wish Go had discriminated unions) like FillPath?
func (sch *UnarySchema) Validate(data cue.Value) (*Instance, error) {
	err := sch.raw.Subsume(data, cue.Concrete(true))
	if err != nil {
		return nil, err
	}

	return &Instance{
		raw:  data,
		sch:  sch,
		name: "", // FIXME how are we getting this out?
	}, nil
}

// Successor returns the next schema in the lineage, or nil if it is the last schema.
func (sch *UnarySchema) Successor() Schema {
	if sch.lin.allv[len(sch.lin.allv)-1] == sch.v {
		return nil
	}

	succv := sch.lin.allv[searchSynv(sch.lin.allv, sch.v)+1]
	succ, _ := Pick(sch.lin, succv)
	return succ
}

// Predecessor returns the previous schema in the lineage, or nil if it is the first schema.
func (sch *UnarySchema) Predecessor() Schema {
	if sch.v == synv() {
		return nil
	}

	predv := sch.lin.allv[searchSynv(sch.lin.allv, sch.v)-1]
	pred, _ := Pick(sch.lin, predv)
	return pred
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

// RawValue returns the cue.Value that represents the underlying CUE schema.
func (sch *UnarySchema) RawValue() cue.Value {
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
