package scuemata

import (
	"cuelang.org/go/cue"
)

// A Lineage is the top-level container in scuemata, holding the complete
// evolutionary history of a particular kind of object: every schema that has
// ever existed for that object, and the lenses that allow translating between
// those schema versions.
type Lineage interface {
	// First returns the first schema in the lineage.
	//
	// All valid Lineage implementations must return a non-nil Schema from this
	// method with a Version() of [0, 0].
	First() Schema

	// Raw returns the cue.Value of the entire lineage.
	RawValue() cue.Value

	// Name returns the name of the object schematized in the lineage, as defined
	// in the lineage's Name field.
	Name() string
}

// A Lacuna represents a semantic gap in a Lens's mapping between schemas.
//
// For any given mapping between schema, there may exist some valid values and
// intended semantics on either side that are impossible to precisely translate.
// When such gaps occur, and an actual schema instance falls into such a gap,
// the Lens is expected to emit Lacuna that describe the general nature of the
// translation gap.
//
// A lacuna may be unconditional (the gap exists for all possible instances
// being translated between the schema pair) or conditional (the gap only exists
// when certain values appear in the instance being translated between schema).
// However, the conditionality of lacunae is expected to be expressed at the
// level of the lens, and determines whether a particular lacuna object is
// created; the production of a lacuna object as the output of a specific
// translation indicates the lacuna applies to that specific translation.
type Lacuna struct {
	// The field path(s) and their value(s) in the pre-translation resource
	// that are relevant to the lacuna.
	SourceFields []FieldRef

	// The field path(s) and their value(s) in the post-translation resource
	// that are relevant to the lacuna.
	TargetFields []FieldRef
	Type         LacunaType

	// A human-readable message describing the gap in translation.
	Message string
}

type LacunaType uint16

type FieldRef struct {
	Path  string
	Value interface{}
}

// Schema represents a single, complete schema from a scuemata lineage. A Schema can
// perform operations on resources.
type Schema interface {
	// Validate checks that the resource is correct with respect to the schema.
	Validate(Resource) error

	// Translate transforms a Resource into a new Resource that is correct with
	// respect to its Successor schema. It returns the transformed resource,
	// the schema to which the resource now conforms, and any errors that
	// may have occurred during the translation.
	//
	// No translation occurs and the input Resource is returned in two cases:
	//
	//   - The translation encountered an error; the third return is non-nil.
	//   - There exists no schema to migrate to; the second and third return are nil.
	//
	// Note that the returned schema is always a Schema. This
	// reflects a key design invariant of the system: all translations, whether
	// they begin from a schema inside or outside of the lineage, must land
	// somewhere on a lineage's sequence of schemas.
	Translate(Resource) (Resource, Schema, error)

	// Successor returns the next schema in the lineage.
	Successor() Schema

	// Raw returns the cue.Value containing the actual underlying CUE schema.
	RawValue() cue.Value

	// Version reports the major and minor versions of the schema.
	Version() (major, minor int)
}

type SSchema struct {
	val        *cue.Value
	pred, succ *SSchema
}

type ValidatedResource interface {
	Forward() (ValidatedResource, []Lacuna, bool)
	Backward() (ValidatedResource, []Lacuna, bool)
}

// A Resource represents a concrete data object - e.g., JSON
// representing a dashboard.
//
// This type mostly exists to improve readability for users. Having a type that
// differentiates cue.Value that represent a schema from cue.Value that
// represent a concrete object is quite helpful. It also gives us a working type
// for a resource that can be reused across multiple calls, so that re-parsing
// isn't necessary.
//
// TODO this is a terrible way to do this, refactor
type Resource struct {
	Value interface{}
	Name  string
}
