package thema

import (
	"cuelang.org/go/cue"
)

func BuildLineage(raw cue.Value, lib Library) (Lineage, error) {
	defLineage := lib.RawValue().LookupPath(cue.MakePath(cue.Def("#Lineage")))
	if err := defLineage.Subsume(raw, cue.Raw()); err != nil {
		return nil, err
	}

	// seqiter, err := raw.LookupPath(cue.MakePath(cue.Str("Seqs"))).List()
	// if err != nil {
	// 	return nil, err
	// }

	// var seqv int
	// var first, lastgvs *genericVersionedSchema
	// for seqiter.Next() {
	// 	var schv int
	// 	schiter, _ := seqiter.Value().LookupPath(cue.MakePath(cue.Str("schemas"))).List()
	// 	for schiter.Next() {
	// 		gvs := &genericVersionedSchema{
	// 			actual: schiter.Value(),
	// 			major:  seqv,
	// 			minor:  schv,
	// 			// This gets overwritten on all but the very final schema
	// 			translation: terminalTranslationFunc,
	// 		}

	// 		if schv != 0 {
	// 			// TODO Verify that this schema is backwards compat with prior.
	// 			// Create an implicit translation operation on the prior schema.
	// 			lastgvs.translation = implicitTranslation(gvs.actual, gvs)
	// 			lastgvs.next = gvs
	// 		} else if seqv != 0 {
	// 			lastgvs.next = gvs
	// 			// x.0. There must exist a lens; load it up and ready it for
	// 			// use, and place it on the final schema in the prior sequence.
	// 			//
	// 			// Also...should at least try to make sure it's pointing at the
	// 			// expected schema, to maintain our invariants?

	// 			// TODO impl
	// 		} else {
	// 			first = gvs
	// 		}
	// 		lastgvs = gvs
	// 		schv++
	// 	}
	// 	seqv++
	// }

	return &lineage{}, nil
}

type compatInvariantError struct {
	rawlin    cue.Value
	violation [2][2]int
	detail    error
}

func (e *compatInvariantError) Error() string {
	panic("TODO")
}

func ValidateCompatibilityInvariants(lin cue.Value, lib Library) error {
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

type lineage struct {
	name  string
	first Schema
	raw   cue.Value
}

func (lin *lineage) First() Schema {
	return lin.first
}

func (lin *lineage) RawValue() cue.Value {
	return lin.raw
}

func (lin *lineage) Name() string {
	return lin.name
}

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
