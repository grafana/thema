package thema

import (
	"cuelang.org/go/cue"
)

// A Lineage is the top-level container in thema, holding the complete
// evolutionary history of a particular kind of object: every schema that has
// ever existed for that object, and the lenses that allow translating between
// those schema versions.
type Lineage interface {
	// First returns the first schema in the lineage.
	//
	// All valid Lineage implementations must return a non-nil Schema from this
	// method with a Version() of [0, 0].
	First() Schema

	// RawValue returns the cue.Value of the entire lineage.
	RawValue() cue.Value

	// Name returns the name of the object schematized in the lineage, as defined
	// in the lineage's Name field.
	Name() string

	// Lineage must be a private interface in order to force creation of them
	// through BindLineage().
	_p()
}

// A LineageFactory returns a Lineage, which is immutably bound to a single
// instance of #Lineage declared in CUE.
//
// LineageFactory funcs are intended to be the main Go entrypoint to all of the
// operations, guarantees, and capabilities of Thema lineages. Lineage authors
// should define and export one instance of LineageFactory per #Lineage
// instance.
//
// It is idiomatic to name LineageFactory funcs after the "name" field on the
// lineage they return:
//
//   func <Name>Lineage ...
//
// If the Go package and lineage name are the same, the name should be omitted from
// the builder func to reduce stutter:
//
//   func Lineage ...
//
// type LineageFactory func(lib Library, opts ...BuildOption) (Lineage, error)
type LineageFactory func(lib Library) (Lineage, error)

// A BuildOption defines build-time only options for constructing a Lineage.
//
// No options currently exist, but some are planned. This option is preemptively
// defined to avoid breaking changes to the signature of BindLineage and
// LineageFactory.
// type BuildOption buildOption

// Internal representation of BuildOption.
type buildOption func(c *buildConfig)

// Internal build-time configuration options.
type buildConfig struct{}

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
// created; the production of a lacuna object as the output of the translation
// of a particular instance indicates the lacuna applies to that specific
// translation.
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

// Schema represents a single, complete schema from a thema lineage. A Schema can
// perform operations on resources.
type Schema interface {
	// Validate checks that the provided data is valid with respect to the
	// schema. If valid, the data is wrapped in an Instance and returned.
	// Otherwise, a nil Instance is returned along with an error detailing the
	// validation failure.
	Validate(data cue.Value) (*Instance, error)

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
	Translate(Instance) (Instance, Schema, error)

	// Successor returns the next schema in the lineage, or nil if it is the last schema.
	Successor() Schema

	// Predecessor returns the previous schema in the lineage, or nil if it is the first schema.
	Predecessor() Schema

	// RawValue returns the cue.Value that represents the underlying CUE schema.
	RawValue() cue.Value

	// Version reports the canonical Thema version number for the schema: a
	// two-tuple of sequence number and schema number.
	Version() (major, minor int)

	// Lineage returns the lineage of which this schema is a part.
	Lineage() Lineage
}

type SSchema struct {
	val        cue.Value
	pred, succ *SSchema
}

type TranslationLacunae interface {
	AsList() []Lacuna
}

// An Instance represents some data that has been validated against a
// lineage's schema. It includes a reference to the schema.
type Instance struct {
	// The CUE representation of the input data
	val cue.Value
	// A name for the input data, primarily for use in error messages
	name string
	// The schema the data validated against/of which the input data is a valid instance
	sch Schema
}

// func (i *Instance) Forward() (*Instance, []Lacuna) {

// }

// func (i *Instance) Reverse() (*Instance, []Lacuna) {

// }

// RawValue returns the cue.Value that represents the instance's underlying data.
func (i *Instance) RawValue() cue.Value {
	return i.val
}

// Schema returns the schema which validated the instance.
func (i *Instance) Schema() Schema {
	return i.sch
}
