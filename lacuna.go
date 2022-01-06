package thema

// TranslationLacunae defines common patterns for unary and composite lineages
// in the lacunae their translations emit.
type TranslationLacunae interface {
	AsList() []Lacuna
}

type flatLacunae []Lacuna

func (fl flatLacunae) AsList() []Lacuna {
	return fl
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
