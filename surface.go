package thema

import (
	"fmt"
	"strconv"
	"strings"

	"cuelang.org/go/cue"
	terrors "github.com/grafana/thema/errors"
	"github.com/grafana/thema/internal/envvars"
)

// A CUEWrapper wraps a cue.Value, and can return that value for inspection.
type CUEWrapper interface {
	// UnwrapCUE returns the underlying cue.Value wrapped by the object.
	UnwrapCUE() cue.Value
}

// A Lineage is the top-level container in thema, holding the complete
// evolutionary history of a particular kind of object: every schema that has
// ever existed for that object, and the lenses that allow translating between
// those schema versions.
type Lineage interface {
	CUEWrapper

	// Name returns the name of the object schematized by the lineage, as declared
	// in the lineage's `name` field.
	Name() string

	// ValidateAny checks that the provided data is valid with respect to at
	// least one of the schemas in the lineage. The oldest (smallest) schema against
	// which the data validates is chosen. A nil return indicates no validating
	// schema was found.
	//
	// While this method takes a cue.Value, this is only to avoid having to trigger
	// the translation internally; input values must be concrete. To use
	// incomplete CUE values with Thema schemas, prefer working directly in CUE,
	// or if you must, rely on UnwrapCUE().
	//
	// TODO should this instead be interface{} (ugh ugh wish Go had tagged unions) like FillPath?
	ValidateAny(data cue.Value) *Instance

	// Schema returns the schema identified by the provided version, if one exists.
	//
	// Only the [0, 0] schema is guaranteed to exist in all valid lineages.
	Schema(v SyntacticVersion) (Schema, error)

	// Library returns the thema.Library instance with which this lineage was built.
	Library() Library

	// Lineage must be a private interface in order to restrict their creation
	// through BindLineage().
	_lineage()
}

// SchemaP returns the schema identified by the provided version. If no schema
// exists in the lineage with the provided version, it panics.
//
// This is a simple convenience wrapper on the Lineage.Schema() method.
func SchemaP(lin Lineage, v SyntacticVersion) Schema {
	sch, err := lin.Schema(v)
	if err != nil {
		panic(err)
	}
	return sch
}

// LatestVersion returns the version number of the newest (largest) schema
// version in the provided lineage.
func LatestVersion(lin Lineage) SyntacticVersion {
	isValidLineage(lin)

	switch tlin := lin.(type) {
	case *UnaryLineage:
		return tlin.allv[len(tlin.allv)-1]
	default:
		panic("unreachable")
	}
}

// LatestVersionInSequence returns the version number of the newest (largest) schema
// version in the provided sequence number.
//
// An error indicates the number of the provided sequence does not exist.
func LatestVersionInSequence(lin Lineage, seqv uint) (SyntacticVersion, error) {
	isValidLineage(lin)

	switch tlin := lin.(type) {
	case *UnaryLineage:
		latest := tlin.allv[len(tlin.allv)-1]
		switch {
		case latest[0] < seqv:
			return synv(), fmt.Errorf("lineage does not contain a sequence with number %v", seqv)
		case latest[0] == seqv:
			return latest, nil
		default:
			return tlin.allv[searchSynv(tlin.allv, SyntacticVersion{seqv + 1, 0})], nil
		}
	default:
		panic("unreachable")
	}
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
//	func <name>Lineage ...
//
// If the Go package and lineage name are the same, the name should be omitted from
// the builder func to reduce stutter:
//
//	func Lineage ...
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
		// We let the env var override this to make it easy to disable on tests.
		if !envvars.ForceVerify {
			c.skipbuggychecks = true
		}
	}
}

// Schema represents a single, complete schema from a thema lineage. A Schema's
// Validate() method determines whether some data constitutes an Instance.
type Schema interface {
	CUEWrapper

	// Validate checks that the provided data is valid with respect to the
	// schema. If valid, the data is wrapped in an Instance and returned.
	// Otherwise, a nil Instance is returned along with an error detailing the
	// validation failure.
	//
	// While Validate takes a cue.Value, this is only to avoid having to trigger
	// the translation internally; input values must be concrete. To use
	// incomplete CUE values with Thema schemas, prefer working directly in CUE,
	// or if you must, rely on the UnwrapCUE().
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

	// Version returns the schema's version number.
	Version() SyntacticVersion

	// Lineage returns the lineage that contains this schema.
	Lineage() Lineage

	// Schema must be a private interface in order to ensure all instances fully
	// conform to Thema invariants.
	_schema()
}

// Assignee is a type constraint used by Thema generics for type parameters
// where there exists a particular Schema that is AssignableTo() the type.
//
// This property is not representable in Go's static type system, as Thema types
// are dynamic, and AssignableTo() is a runtime check. Thus, the only actual
// type constraint Go's type system can be made aware of is any.
//
// Instead, Thema's implementation guarantees that it is only possible to
// instantiate a generic type with an Assignee type parameter if the relevant
// AssignableTo() relation has already been verified, and there is an
// unambiguous relationship between the generic type and the relevant Schema.
//
// For example: for TypedSchema[T Assignee], it is the related Schema. With
// TypedInstance[T Assignee], the related schema is returned from its
// TypedSchema() method.
//
// As this type constraint is simply any, it exists solely as a signal to the
// human reader that the relation to a Schema exists, and that the relation
// has been verified in any properly instantiated type carrying this generic
// type constraint. (Improperly instantiated generic Thema types panic upon
// calls to any of their methods)
type Assignee interface {
	any
}

// type Assignee any

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
// A trivial helper to avoid repetitive Go-stress disorder from countless
// instances of typing:
//
//	SyntacticVersion{0, 0}
func SV(seqv, schv uint) SyntacticVersion {
	return SyntacticVersion([2]uint{seqv, schv})
}

// Less reports whether the provided SyntacticVersion is less than the receiver,
// consistent with the expectations of Go's sort package.
func (sv SyntacticVersion) Less(osv SyntacticVersion) bool {
	return sv[0] < osv[0] || sv[1] < osv[1]
}

func (sv SyntacticVersion) String() string {
	return fmt.Sprintf("%v.%v", sv[0], sv[1])
}

// ParseSyntacticVersion parses a canonical representation of a syntactic
// version (e.g. "0.0") from a string.
func ParseSyntacticVersion(s string) (SyntacticVersion, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 2 {
		return synv(), fmt.Errorf("%w: %q", terrors.ErrMalformedSyntacticVersion, s)
	}

	// i mean 4 billion is probably enough version numbers
	seqv, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		return synv(), fmt.Errorf("%w: %q has invalid sequence number %q", terrors.ErrMalformedSyntacticVersion, s, parts[0])
	}

	// especially when squared
	schv, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return synv(), fmt.Errorf("%w: %q has invalid schema number %q", terrors.ErrMalformedSyntacticVersion, s, parts[1])
	}
	return synv(uint(seqv), uint(schv)), nil
}
