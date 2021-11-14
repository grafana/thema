package scuemata

import (
	"cuelang.org/go/cue"
)

func BuildLineage(raw cue.Value) (Lineage, error) {
	// TODO verify subsumption by #Lineage; renders many
	// error checks below unnecessary
	majiter, err := raw.LookupPath(cue.MakePath(cue.Str("seqs"))).List()
	if err != nil {
		return nil, err
	}

	var major int
	var first, lastgvs *genericVersionedSchema
	for majiter.Next() {
		var minor int
		miniter, _ := majiter.Value().List()
		for miniter.Next() {
			gvs := &genericVersionedSchema{
				actual: miniter.Value(),
				major:  major,
				minor:  minor,
				// This gets overwritten on all but the very final schema
				translation: terminalTranslationFunc,
			}

			if minor != 0 {
				// TODO Verify that this schema is backwards compat with prior.
				// Create an implicit translation operation on the prior schema.
				lastgvs.translation = implicitTranslation(gvs.actual, gvs)
				lastgvs.next = gvs
			} else if major != 0 {
				lastgvs.next = gvs
				// x.0. There must exist a lens; load it up and ready it for
				// use, and place it on the final schema in the prior sequence.
				//
				// Also...should at least try to make sure it's pointing at the
				// expected schema, to maintain our invariants?

				// TODO impl
			} else {
				first = gvs
			}
			lastgvs = gvs
			minor++
		}
		major++
	}

	return first, nil
}

type genericVersionedSchema struct {
	actual      cue.Value
	major       int
	minor       int
	next        *genericVersionedSchema
	translation translationFunc
}

// Validate checks that the resource is correct with respect to the schema.
func (gvs *genericVersionedSchema) Validate(r schema.Resource) error {
	name := r.Name
	if name == "" {
		name = "resource"
	}
	rv := ctx.CompileString(r.Value.(string), cue.Filename(name))
	if rv.Err() != nil {
		return rv.Err()
	}
	return gvs.actual.Unify(rv).Validate(cue.Concrete(true))
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
	if gvs.next == nil {
		// Untyped nil, allows `<sch> == nil` checks to work as people expect
		return nil
	}
	return gvs.next
}

// Translate transforms a resource into a new Resource that is correct with
// respect to its Successor schema.
func (gvs *genericVersionedSchema) Translate(x schema.Resource) (schema.Resource, Schema, error) { // TODO restrict input/return type to concrete
	r, sch, err := gvs.translation(x.Value)
	if err != nil || sch == nil {
		r = x.Value.(cue.Value)
	}

	return schema.Resource{Value: r}, sch, nil
}

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
