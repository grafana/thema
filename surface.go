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

	// Name returns the name of the object schematized by the lineage, as declared
	// in the lineage's `name` field.
	Name() string

	// Lineage must be a private interface in order to restrict their creation
	// through BindLineage().
	_lineage()
}

// A LineageFactory returns a Lineage, which is immutably bound to a single
// instance of #Lineage declared in CUE.
//
// LineageFactory funcs are intended to be the main Go entrypoint to all of the
// operations, guarantees, and capabilities of Thema lineages. Lineage authors
// should generally define and export one instance of LineageFactory per
// #Lineage instance.
//
// It is idiomatic to name LineageFactory funcs after the "name" field on the
// lineage they return:
//
//   func <name>Lineage ...
//
// If the Go package and lineage name are the same, the name should be omitted from
// the builder func to reduce stutter:
//
//   func Lineage ...
//
type LineageFactory func(lib Library, opts ...BindOption) (Lineage, error)

// A BindOption defines options that may be specified only at initial
// construction of a Lineage via BindLineage.
type BindOption bindOption

// Internal representation of BindOption.
type bindOption func(c *bindConfig)

// Internal bind-time configuration options.
type bindConfig struct {
	skipbuggychecks bool
}

// SkipBuggyChecks indicates that BindLineage should skip validation checks
// which have known bugs (e.g. panics) for certain should-be-valid CUE inputs.
//
// By default, BindLineage performs these checks anyway, as otherwise the
// default behavior of BindLineage is to not provide the guarantees it's
// supposed to provide.
//
// As Thema and CUE move towards maturity and the set of validations that are
// both a) necessary and b) buggy empties out, this will naturally become a
// no-op. At that point, this function will be marked deprecated.
//
// Ratcheting up verification checks in this way does mean that any code relying
// on this to bypass verification in BindLineage may begin failing in future
// versions of Thema if the underlying lineage being verified doesn't comply
// with a planned invariant.
func SkipBuggyChecks() BindOption {
	return func(c *bindConfig) {
		c.skipbuggychecks = true
	}
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
// created; the production of a lacuna object as the output of the translation
// of a particular instance indicates the lacuna applies to that specific
// translation.
type Lacuna struct {
	// The field path(s) and their value(s) in the pre-translation instance
	// that are relevant to the lacuna.
	SourceFields []FieldRef

	// The field path(s) and their value(s) in the post-translation instance
	// that are relevant to the lacuna.
	TargetFields []FieldRef
	Type         LacunaType

	// A human-readable message describing the gap in translation.
	Message string
}

// LacunaType assigns numeric identifiers to different classes of Lacunae.
//
// FIXME this is a terrible way of doing this and needs to change
type LacunaType uint16

// FieldRef identifies a path/field and the value in it within a Lacuna.
type FieldRef struct {
	Path  string
	Value interface{}
}

// Schema represents a single, complete schema from a thema lineage. A Schema's
// Validate() method determines whether some data constitutes an Instance.
type Schema interface {
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
	// TODO should this instead be interface{} (ugh ugh wish Go had tagged unions) like FillPath?
	Validate(data cue.Value) (*Instance, error)

	// Successor returns the next schema in the lineage, or nil if it is the last schema.
	Successor() Schema

	// Predecessor returns the previous schema in the lineage, or nil if it is the first schema.
	Predecessor() Schema

	// RawValue returns the cue.Value that represents the underlying CUE schema.
	RawValue() cue.Value

	// Version returns the schema's version number.
	Version() SyntacticVersion

	// Lineage returns the lineage that contains this schema.
	Lineage() Lineage

	// Schema must be a private interface in order to ensure all instances fully
	// conform to Thema invariants.
	_schema()
}

// SyntacticVersion is a two-tuple of uints describing the position of a schema
// within a lineage. Syntactic versions are Thema's canonical version numbering
// system.
//
// The first element is the index of the sequence containing the schema within
// the lineage, and the second element is the index of the schema within that
// sequence.
type SyntacticVersion [2]uint

func (sv SyntacticVersion) less(osv SyntacticVersion) bool {
	return sv[0] < osv[0] || sv[1] < osv[1]
}

// TranslationLacunae defines common patterns for unary and composite lineages
// in the lacunae their translations emit.
type TranslationLacunae interface {
	AsList() []Lacuna
}

// An Instance represents some data that has been validated against a
// lineage's schema. It includes a reference to the schema.
type Instance struct {
	// The CUE representation of the input data
	raw cue.Value
	// A name for the input data, primarily for use in error messages
	name string
	// The schema the data validated against/of which the input data is a valid instance
	sch Schema
}

// func (i *Instance) TranslateForward() (*Instance, []Lacuna) {
// }

// func (i *Instance) TranslateReverse() (*Instance, []Lacuna) {
// }

// RawValue returns the cue.Value that represents the instance's underlying data.
func (i *Instance) RawValue() cue.Value {
	return i.raw
}

// Schema returns the schema which subsumes/validated this instance.
func (i *Instance) Schema() Schema {
	return i.sch
}
