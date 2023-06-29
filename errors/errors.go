package errors

import "github.com/cockroachdb/errors"

// ValidationCode represents different classes of validation errors that may
// occur vs. concrete data inputs.
type ValidationCode uint16

const (
	// KindConflict indicates a validation failure in which the schema and data
	// values are of differing, conflicting kinds - the schema value does not
	// subsume the data value. Example: data: "foo"; schema: int
	KindConflict ValidationCode = 1 << iota

	// OutOfBounds indicates a validation failure in which the data and schema have
	// the same (or subsuming) kinds, but the data is out of schema-defined bounds.
	// Example: data: 4; schema: int & <4
	OutOfBounds

	// MissingField indicates a validation failure in which the data lacks
	// a field that is required in the schema.
	MissingField

	// ExcessField indicates a validation failure in which the schema is treated as
	// closed, and the data contains a field not specified in the schema.
	ExcessField
)

// ValidationError is a subtype of
type ValidationError struct {
	msg string
}

// Unwrap implements standard Go error unwrapping, relied on by errors.Is.
//
// All ValidationErrors wrap the general ErrNotAnInstance sentinel error.
func (ve *ValidationError) Unwrap() error {
	return ErrNotAnInstance
}

// Validation error codes/types
var (
	// ErrNotAnInstance is the general error that indicates some data failed validation
	// against a Thema schema. Use it with errors.Is() to differentiate validation errors
	// from other classes of failure.
	ErrNotAnInstance = errors.New("data not a valid instance of schema")

	// ErrInvalidExcessField indicates a validation failure in which the schema is
	// treated as closed, and the data contains a field not specified in the schema.
	ErrInvalidExcessField = errors.New("data contains field not present in schema")

	// ErrInvalidMissingField indicates a validation failure in which the data lacks
	// a field that is required in the schema.
	ErrInvalidMissingField = errors.New("required field is absent in data")

	// ErrInvalidKindConflict indicates a validation failure in which the schema and
	// data values are of differing, conflicting kinds - the schema value does not
	// subsume the data value. Example: data: "foo"; schema: int
	ErrInvalidKindConflict = errors.New("schema and data are conflicting kinds")

	// ErrInvalidOutOfBounds indicates a validation failure in which the data and
	// schema have the same (or subsuming) kinds, but the data is out of
	// schema-defined bounds. Example: data: 4; schema: int & <3
	ErrInvalidOutOfBounds = errors.New("data is out of schema bounds")
)

// Lower level general errors
var (
	// ErrValueNotExist indicates that a necessary CUE value did not exist.
	ErrValueNotExist = errors.New("cue value does not exist")

	// ErrValueNotALineage indicates that a provided CUE value is not a lineage.
	// This is almost always an end-user error - they oops'd and provided the
	// wrong path, file, etc.
	ErrValueNotALineage = errors.New("not a lineage")

	// ErrInvalidLineage indicates that a provided lineage does not fulfill one
	// or more of the Thema invariants.
	ErrInvalidLineage = errors.New("invalid lineage")

	// ErrInvalidSchemasOrder indicates that schemas in a lineage are not ordered
	// by version.
	ErrInvalidSchemasOrder = errors.New("schemas in lineage are not ordered by version")

	// ErrInvalidLensesOrder indicates that lenses are in the wrong order - they must be sorted by `to`, then `from`.
	ErrInvalidLensesOrder = errors.New("schemas in lineage are not ordered by version")

	// ErrVersionNotExist indicates that no schema exists in a lineage with a
	// given version.
	ErrVersionNotExist = errors.New("lineage does not contain schema with version") // ErrNoSchemaWithVersion

	// ErrMalformedSyntacticVersion indicates a string input of a syntactic
	// version was malformed.
	ErrMalformedSyntacticVersion = errors.New("not a valid syntactic version")
)
