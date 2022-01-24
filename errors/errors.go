package errors

import "errors"

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

	// ErrVersionNotExist indicates that no schema exists in a lineage with a
	// given version.
	ErrVersionNotExist = errors.New("lineage does not contain schema with version")

	// ErrMalformedSyntacticVersion indicates a string input of a syntactic
	// version was malformed.
	ErrMalformedSyntacticVersion = errors.New("not a valid syntactic version")
)
