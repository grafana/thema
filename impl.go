package thema

import (
	"fmt"

	"cuelang.org/go/cue"
)

type ErrValueNotExist struct {
	path string
}

func (e *ErrValueNotExist) Error() string {
	return fmt.Sprintf("value from path %q does not exist, absent values cannot be lineages", e.path)
}

// BindLineage takes a raw cue.Value, checks that it is a valid lineage (that it
// upholds the invariants which undergird Thema's translatability guarantees),
// and returns the cue.Value wrapped in a Lineage, iff validity checks succeed. The Lineage type
// provides access to all the types and functions for working with Thema in Go.
//
// This function is the sole intended mechanism for creating Lineage objects,
// thereby providing a practical promise that all instances of Lineage uphold
// Thema's invariants. It is primarily intended for use by authors of lineages
// in the creation of a LineageFactory.
func BindLineage(raw cue.Value, lib Library) (Lineage, error) {
	if !raw.Exists() {
		return nil, &ErrValueNotExist{
			path: raw.Path().String(),
		}
	}

	defLineage := lib.RawValue().LookupPath(cue.MakePath(cue.Def("#Lineage")))
	if err := defLineage.Subsume(raw, cue.Raw()); err != nil {
		return nil, err
	}

	// FIXME Guarded with unstable until the underlying CUE bugs related to iterated
	// object subsumption checks are fixed
	if unstable {
		if err := verifySeqCompatInvariants(raw, lib); err != nil {
			return nil, err
		}
	}

	return &UnaryLineage{
		validated: true,
		raw:       raw,
		lib:       lib,
	}, nil
}

type compatInvariantError struct {
	rawlin    cue.Value
	violation [2][2]int
	detail    error
}

func (e *compatInvariantError) Error() string {
	panic("TODO")
}

func verifySeqCompatInvariants(lin cue.Value, lib Library) error {
	dlin := lib.RawValue().LookupPath(cue.MakePath(cue.Def("#Lineage")))
	if dlin.Err() != nil {
		panic(dlin.Err())
	}

	if err := dlin.Subsume(lin, cue.Raw(), cue.Schema()); err != nil {
		return err
	}

	seqiter, _ := lin.LookupPath(cue.MakePath(cue.Str("Seqs"))).List()
	var seqv int
	var predecessor cue.Value
	var predv [2]int
	for seqiter.Next() {
		var schv int
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
					violation: [2][2]int{predv, {seqv, schv}},
					detail:    bcompat,
				}
			}

			predecessor = sch
			predv = [2]int{seqv, schv}
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
}

func (lin *UnaryLineage) First() Schema {
	return lin.first
}

func (lin *UnaryLineage) RawValue() cue.Value {
	return lin.raw
}

func (lin *UnaryLineage) Name() string {
	return lin.name
}

func (lin *UnaryLineage) _p() {}

type genericVersionedSchema struct {
	actual      cue.Value
	major       int
	minor       int
	next        *genericVersionedSchema
	translation translationFunc
}

// Validate checks that the resource is correct with respect to the schema.
func (gvs *genericVersionedSchema) Validate(r Instance) error {
	return gvs.actual.Unify(r.val).Validate(cue.Concrete(true))
}

// CUE returns the cue.Value representing the actual schema.
func (gvs *genericVersionedSchema) CUE() cue.Value {
	return gvs.actual
}

// Version reports the major and minor versions of the schema.
func (gvs *genericVersionedSchema) Version() (major int, minor int) {
	return gvs.major, gvs.minor
}

// Returns the next VersionedCueSchema
func (gvs *genericVersionedSchema) Successor() Schema {
	panic("TODO")
}

// Translate transforms a resource into a new Resource that is correct with
// respect to its Successor schema.
// func (gvs *genericVersionedSchema) Translate(x Instance) (Instance, Schema, error) { // TODO restrict input/return type to concrete
// 	r, sch, err := gvs.translation(x.Value)
// 	if err != nil || sch == nil {
// 		r = x.Value.(cue.Value)
// 	}

// 	return Instance{Value: r}, sch, nil
// }

type translationFunc func(x interface{}) (cue.Value, Schema, error)

var terminalTranslationFunc = func(x interface{}) (cue.Value, Schema, error) {
	// TODO send back the input
	return cue.Value{}, nil, nil
}

// panic if called
// var panicTranslationFunc = func(x interface{}) (cue.Value, Schema, error) {
// 	panic("translations are not yet implemented")
// }

// Creates a func to perform a "translation" that simply unifies the input
// artifact (which is expected to have already have been validated against an
// earlier schema) with a later schema.
func implicitTranslation(v cue.Value, next Schema) translationFunc {
	return func(x interface{}) (cue.Value, Schema, error) {
		w := v.FillPath(cue.Path{}, x)
		// TODO is it possible that translation would be successful, but there
		// still exists some error here? Need to better understand internal CUE
		// erroring rules? seems like incomplete cue.Value may always an Err()?
		//
		// TODO should check concreteness here? Or can we guarantee a priori it
		// can be made concrete simply by looking at the schema, before
		// implicitTranslation() is called to create this function?
		if w.Err() != nil {
			return w, nil, w.Err()
		}
		return w, next, w.Err()
	}
}
