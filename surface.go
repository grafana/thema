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
	// Last returns the last schema in the lineage.
	//
	// All valid Lineage implementations must return a non-nil Schema from this
	// method.
	Last() Schema
	// Value returns the cue.Value representing the entire lineage.
	Value() cue.Value
}

// A Lacuna is a gap in translation.
//
// Each Lacuna instance represents represents a flaw
//
// Lacuna are NOT intended to represent lossiness in translation.
type Lacuna interface {
	SourcePath() string
	TargetPath() string
	Type() LacunaType
	Message() string
}

type LacunaType uint16

// Schema represents a single, complete CUE-based schema from a scuemata lineage
// that can perform operations on Resources.
//
// All Schema MUST EITHER:
//  - Be a Schema, and be the latest version in the latest sequence in a lineage
//  - Return non-nil from Successor(), and a procedure to Translate() a Resource to that successor schema
//
// By definition, Schema are within a sequence. As long as sequence
// backwards compatibility invariants hold, translation to a Schema to
// a successor schema in their sequence is trivial: simply unify the Resource
// with the successor schema.
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

	// Successor returns the Schema to which this Schema can migrate a
	// Resource.
	Successor() Schema

	// CUE returns the cue.Value representing the actual schema.
	CUE() cue.Value

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
