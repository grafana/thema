package scuemata

import (
    "list"
)

// A Lineage is the top-level container in scuemata, holding the complete
// evolutionary history of a particular kind of object: every schema that has
// ever existed for that object, and the lenses that allow translating between
// those schema versions.
#Lineage: {
    // JoinSchema governs the shape of schema that may be expressed in a
    // lineage. It is the least upper bound, or join, of the acceptable schema
    // value space; the schemas defined in this lineage must be instances of the
    // JoinSchema.
    // 
    // In the base case, the JoinSchema is unconstrained/top - any value may be
    // used as a schema.
    //
    // A lineage's JoinSchema may never change as the lineage evolves.
    //
    // TODO should it be an open struct rather than top?
    // TODO can this be a def? should it?
    JoinSchema: _

    // The name of the object schematized in this lineage.
    Name: string
    // TODO https://github.com/cue-lang/cue/issues/943
    // Name: must(isconcrete(Name), "all lineages must have a name")

    // A Sequence is an ordered list of schema, with the invariant that
    // successive schemas are backwards compatible with their predecessors.
    #Sequence: [...JoinSchema] & list.MinItems(1)

    // Uncomment this for certain local dev tasks because constraining with
    // list.MinItems(1) isn't yet smart enough to tell the typechecker that it
    // is always safe to reference #Sequence[0]. We DON'T want this committed,
    // though, because it would allow empty lineage declarations by relying on
    // the JoinSchema, which we do not want to be valid text for authors to write.
    // #Sequence: [JoinSchema, ...JoinSchema]

    #Lens: {
        // The last schema in the previous sequence; logical predecessor
        ancestor: JoinSchema
        // The first schema in this sequence; logical successor
        descendant: JoinSchema
        forward: {
            to: descendant
            from: ancestor
            rel: descendant
            lacunae: [...#Lacuna]
            translated: to & rel
        }
        reverse: {
            to: ancestor
            from: descendant
            rel: ancestor
            lacunae: [...#Lacuna]
            translated: to & rel
        }
    }

    // Seqs is the list of sequences of schema that comprise the overall
    // lineage, along with the lenses that allow translation back and forth
    // across sequences.
    Seqs: [
        {
            schemas: #Sequence
        },
        ...{
            lens: #Lens
            schemas: #Sequence
        }
    ]

    // Constrain that ancestor and descendant for each defined lens are the
    // final and initial schemas in the predecessor seq and the seq containing
    // the lens, respectively.
    if len(Seqs) > 1 {
        for lv, l in Seqs {
            if lv < len(Seqs)-1 {
                // TODO can we close these? would be great to close these
                Seqs[lv+1] & { lens: ancestor: l.schemas[len(l.schemas)-1] }
                Seqs[lv+1] & { lens: descendant: Seqs[lv+1].schemas[0] }
            }
        }
    }

    // TODO check subsumption (backwards compat) of each schema with its successor

    // Select a single schema version from the lineage.
    Pick: {
        // The schema version to pick. Either:
        //
        //   * An exact #SchemaVersion, e.g. [1, 0]
        //   * Just the sequence number, list, e.g. [1]
        //
        // The latter form will select the latest schema within the given
        // sequence.
        v: #SchemaVersion | [int & >= 0]
        v: [<len(Seqs), <len(Seqs[v[0]].schemas)] | [<len(Seqs)]

        let _v = #SchemaVersion & [
            v[0],
            if len(v) == 2 { v[1] },
            if len(v) == 1 { len(Seqs[v[0].schemas]) - 1 },
        ]

        out: Seqs[_v[0]].schemas[_v[1]]
        // TODO ^ apply object headers, etc.
    }

    // Helper that extracts SchemaVersion of latest schema.
    //
    // Internal only, because writing programs that include a textual references
    // which can float across backwards-incompatible changes (like this one) is
    // the exact thing scuemata is trying to avoid.
    //
    // TODO make into an alias?
    _latest: #SchemaVersion
    _latest: [len(Seqs)-1, len(Seqs[len(Seqs)-1].schemas)-1]

    // Helper that flattens all schema into a single list, putting their
    // SchemaVersion in an adjacent property.
    _all: [..._#vSch] & list.FlattenN([for seqv, seq in Seqs {
        [for schv, sch in seq.schemas {
            v: [seqv, schv]
            sch: sch
        }]
    }], 1)
}

_#vSch: {
    v: #SchemaVersion
    sch: _
}

// SchemaVersion represents the version of a schema within a lineage as a
// 2-tuple of integers: coordinates, corresponding to the schema's index
// in a sequence, and that sequence's index within the list of sequences.
//
// Unlike most version numbering systems, a schema's version is not an arbitrary
// number declared by the lineage's author. Rather, version numbers are derived
// from the position of the schema within the lineage's list of sequences.
// Sequence position, in turn, is governed by scuemata's constraints on
// backwards compatibility and lens existence. By tying version numbers to these
// checkable invariants, scuemata versions gain a precise semantics absent from
// other systems.
#SchemaVersion: [int & >= 0, int & >= 0]

// TODO functionize
_cmpSV: {
    l: #SchemaVersion
    r: #SchemaVersion
    out: -1 | 0 | 1
    out: {
        if l == r { 0 }
        if l[0] < r[0] { -1 }
        if l[1] < r[1] { -1 }
        if l[0] > r[0] { 1 }
        if l[1] > r[1] { 1 }
    }
}

_flatidx: {
    lin: #Lineage
    v: #SchemaVersion
    fidx: {for i, sch in lin._all if sch.v == v { i }}
}