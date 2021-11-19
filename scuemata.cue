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

    Name: string

    // A Sequence is an ordered list of schema, with the invariant that
    // successive schemas are backwards compatible with their predecessors.
    #Sequence: [JoinSchema, ...JoinSchema]
    // TODO check subsumption (backwards compat) of each schema with its successor

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

    // seqs is the list of sequences of schema that comprise the overall
    // lineage, along with the lenses that allow translation back and forth
    // across sequences.
    seqs: [
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
    for lv, l in seqs {
        if lv < len(seqs)-1 {
            // TODO can we close these? would be great to close these
            seqs[lv+1] & { lens: ancestor: l.schemas[len(l.schemas)-1] }
            seqs[lv+1] & { lens: descendant: seqs[lv+1].schemas[0] }
        }
    }

    // Pick a single schema version from the lineage.
    pick: {
        // The schema version to pick. Either:
        //
        //   * An exact #SchemaVersion, e.g. [1, 0]
        //   * Just the sequence number, list, e.g. [1]
        //
        // The latter form will select the latest schema within the given
        // sequence.
        v: #SchemaVersion | [int & >= 0]
        v: [<len(seqs), <len(seqs[v[0]].schemas)] | [<len(seqs)]

        let _v = #SchemaVersion & [
            v[0],
            if len(v) == 2 { v[1] },
            if len(v) == 1 { len(seqs[v[0].schemas]) - 1 },
        ]

        out: seqs[_v[0]].schemas[_v[1]]
        // TODO ^ apply object headers, etc.
    }
    _latest: #SchemaVersion
    _latest: [len(seqs)-1, len(seqs[len(seqs)-1].schemas)-1]

    // Helper that flattens all schema into a single list, putting their
    // SchemaVersion in an adjacent property.
    _all: [..._#vSch] & list.FlattenN([for seqv, seq in seqs {
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
// 2-tuple of integers - a coordinate, corresponding to the schema's position
// within the list of sequences.
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