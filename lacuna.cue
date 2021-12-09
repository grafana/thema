package scuemata

// A Lacuna represents a gap in a translation: a case in which there exists a
// flaw in the lens that affected the translation of a particular resource
// from one schema to another.
//
// A lacuna may be unconditional (the gap exists for all possible instances
// being translated between the schema pair) or conditional (the gap only exists
// when certain values appear in the instance being translated between schema).
// However, the conditionality of lacunae is expected to be expressed at the
// level of the lens, and determines whether a particular lacuna object is
// created; the production of a lacuna object as the output of a specific
// translation indicates the lacuna applies to that specific translation.
#Lacuna: {
    // A reference to a field and its value in a schema/resource.
    #FieldRef: {
        // TODO would be great to be able to constrain that value should always be a reference,
        // and path is (a modified version of) the string representation of the reference
        value: _
        path: string
    }

    // The field path(s) and their value(s) in the pre-translation resource
    // that are relevant to the lacuna.
    sourceFields: [...#FieldRef]

    // The field path(s) and their value(s) in the post-translation resource
    // that are relevant to the lacuna.
    targetFields: [...#FieldRef]

    // At least one of sourceFields or targetFields must be non-empty.
    // TODO(must) https://github.com/cue-lang/cue/issues/943
    // must(len(sourceFields) > 0 || len(targetFields) > 0, "at least one of sourceFields or targetFields must be non-empty")
    _mustlen: >0 & (len(sourceFields) + len(targetFields))

    // A human-readable message describing the gap in translation.
    message: string

    type: or([for t in #LacunaTypes {t}])
}

#LacunaTypes: [N=string]: #LacunaType & { Name: N }
#LacunaTypes: {
    // Placeholder lacunae indicate that a field in the target resource has
    // been filled with a placeholder value.
    //
    // Use Placeholder when introducing a new required field that lacks a default,
    // and it is necessary to fill the field with some value to meet lens
    // validity requirements.
    //
    // A placeholder is NOT a schema-defined default. It is expressly the
    // opposite: a lens-defined value that exists solely to be replaced by the
    // calling program.
    Placeholder: {
        id: 1
    }

    // DroppedField lacunae indicate that field(s) in the source resource were
    // dropped in a manner that potentially lost some of their contained semantics.
    //
    // When a lens drops multiple fields, prefer to create one DroppedField
    // lacuna per distinct cause. For example, if multiple resource fields are
    // dropped from a single open struct because they were absent from the
    // schema, all of those fields should be included in a single DroppedField.
    DroppedField: {
        id: 2
    }
}

#LacunaType: {
    Name: string
    id: int // FIXME this is a dumb way of trying to express identity
}