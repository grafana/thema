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
	First() Schema

	// Schema(SyntacticVersion) Schema

	// RawValue returns the cue.Value of the entire lineage.
	RawValue() cue.Value

	// Name returns the name of the object schematized by the lineage, as declared
	// in the lineage's `name` field.
	Name() string

	// LatestVersion returns the version number of the newest (largest) schema
	// version in the lineage.
	LatestVersion() SyntacticVersion

	// LatestVersionInSequence returns the version number of the newest (largest) schema
	// version in the provided sequence number.
	//
	// An error indicates the number of the provided sequence does not exist.
	LatestVersionInSequence(seqv uint) (SyntacticVersion, error)

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
	ValidateAny(data cue.Value) *Instance

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

	// LatestVersionInSequence returns the version number of the newest (largest) schema
	// in this schema's sequence.
	LatestVersionInSequence() SyntacticVersion

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

// SV creates a SyntacticVersion.
//
// A trivial helper to avoid repetitive Go-stress disorder from typing
//
//   SyntacticVersion([2]uint{0, 0})
func SV(seqv, schv uint) SyntacticVersion {
	return SyntacticVersion([2]uint{seqv, schv})
}

func (sv SyntacticVersion) less(osv SyntacticVersion) bool {
	return sv[0] < osv[0] || sv[1] < osv[1]
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

// AsSuccessor translates the instance into the form specified by the successor
// schema.
//
// TODO figure out how to represent unary vs. composite lineages here
func (i *Instance) AsSuccessor() (*Instance, TranslationLacunae) {
	return Translate(i, i.sch.Successor().Version())
}

// AsPredecessor translates the instance into the form specified by the predecessor
// schema.
//
// TODO figure out how to represent unary vs. composite lineages here
func (i *Instance) AsPredecessor() (*Instance, TranslationLacunae) {
	panic("TODO translation from newer to older schema is not yet implemented")
}

// RawValue returns the cue.Value that represents the instance's underlying data.
func (i *Instance) RawValue() cue.Value {
	return i.raw
}

// Schema returns the schema which subsumes/validated this instance.
func (i *Instance) Schema() Schema {
	return i.sch
}

func (i *Instance) lib() Library {
	return getLinLib(i.Schema().Lineage())
}
