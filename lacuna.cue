package scuemata

// A Lacuna represents a gap in a translation: a case in which there exists a
// flaw in the mapping from one schema to another.
//
// A lacuna may be unconditional (the gap exists for all possible instances
// being translated between the schema pair) or conditional (the gap only exists
// when certain values appear in the instance being translated between schema).
// However, the conditionality of lacunae is expected to be expressed at the
// level of the lens, and determines whether a particular lacuna object is
// created; the production of a lacuna object as the output of a specific
// translation indicates the lacuna applies to that specific translation.
#Lacuna: {
    // The actual values relevant to the gap.
    ref: {
        from: string
    } | {
        to: string
    } | {
        from: string
        to: string
    }
    // TODO need (?) ability to express constraint that string representation of
    // path exists in another structure
}
