package thema

import (
    "list"
)

// A Lineage is the top-level container in thema, holding the complete
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

    // The name of the thing being schematized in this lineage.
    Name: string
    // TODO(must) https://github.com/cue-lang/cue/issues/943
    // Name: must(isconcrete(Name), "all lineages must have a name")

    // A Sequence is a non-empty ordered list of schemas, with the property that
    // every schema in the sequence is backwards compatible with (subsumes) its
    // predecessors.
    #Sequence: [...JoinSchema] & list.MinItems(1)

    // This exists because constraining with list.MinItems(1) isn't able to
    // tell the evaluator that it is always safe to reference #Sequence[0],
    // resulting in lots of garbage errors.
    //
    // Unfortunately, this allows empty lineage declarations by making the first
    // schema an actual JoinSchema, which we do not want to be valid text for
    // authors to write.
    //
    // TODO figure out how to express the constraint without blowing up our Go logic
    #Sequence: [JoinSchema, ...JoinSchema]

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
            schemas: #Sequence
            lens: #Lens
        }
    ]

    // Constrain that ancestor and descendant for each defined lens are the
    // final and initial schemas in the predecessor seq and the seq containing
    // the lens, respectively.
    if len(seqs) > 1 {
        for lv, l in seqs {
            if lv < len(seqs)-1 {
                // TODO can we close these? would be great to close these
                seqs[lv+1] & { lens: ancestor: l.schemas[len(l.schemas)-1] }
                seqs[lv+1] & { lens: descendant: seqs[lv+1].schemas[0] }
            }
        }
    }

    // TODO check subsumption (backwards compat) of each schema with its successor natively in CUE
}

_#vSch: {
    v: #SchemaVersion
    sch: _
}

// Helper that extracts SchemaVersion of latest schema.
//
// Internal only, because writing programs that include a textual references
// which can float across backwards-incompatible changes (like this one) is
// the exact thing thema is trying to avoid.
//
// TODO functionize
_latest: {
    lin: #Lineage
    out: #SchemaVersion & [len(lin.seqs)-1, len(lin.seqs[len(lin.seqs)-1].schemas)-1]
}

// Helper that flattens all schema into a single list, putting their
// SchemaVersion in an adjacent property.
//
// TODO functionize
_all: {
    lin: #Lineage
    out: [..._#vSch] & list.FlattenN([for seqv, seq in lin.seqs {
        [for schv, sch in seq.schemas {
            v: [seqv, schv]
            sch: sch
        }]
    }], 1)
}

// Helper that constructs a one-dimensional list of all the schema versions that
// exist in a lineage.
_allv: {
    lin: #Lineage
    out: [...#SchemaVersion] & list.FlattenN(
        [for seqv, seq in lin.seqs {
            [for schv, _ in seq.schemas { [seqv, schv] }]
        }], 1)
}

// Select a single schema version from the lineage.
#Pick: {
    lin: #Lineage
    // The schema version to pick. Either:
    //
    //   * An exact #SchemaVersion: [1, 0]
    //   * Just the sequence number: [1]
    //
    // The latter form will select the latest schema within the given
    // sequence.
    v: #SchemaVersion | [int & >= 0]
    v: [<len(lin.seqs), <len(lin.seqs[v[0]].schemas)] | [<len(lin.seqs)]
    // TODO(must) https://github.com/cue-lang/cue/issues/943
    // must(isconcrete(v[0]), "must specify a concrete sequence number")

    let _v = #SchemaVersion & [
        v[0],
        if len(v) == 2 { v[1] },
        if len(v) == 1 { len(lin.seqs[v[0].schemas]) - 1 },
    ]

    out: lin.seqs[_v[0]].schemas[_v[1]]
    // TODO ^ apply object headers, etc.
}

// SchemaVersion represents the version of a schema within a lineage as a
// 2-tuple of integers > 0: coordinates, corresponding to the schema's index
// in a sequence, and that sequence's index within the list of sequences.
//
// Unlike most version numbering systems, a schema's version is not an arbitrary
// number declared by the lineage's author. Rather, version numbers are a
// property of the position of the schema within the lineage's list of
// sequences. Sequence position, in turn, is governed by thema's constraints on
// backwards compatibility and lens existence. By tying version numbers to these
// checkable invariants, thema versions gain a precise semantics absent from
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

// TODO functionize
_flatidx: {
    lin: #Lineage
    v: #SchemaVersion
    fidx: {for i, sch in (_all & { lin: lin }).out if sch.v == v { i }}
}