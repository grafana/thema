package scuemata

// A Lineage is the top-level container in scuemata. It contains the 
// evolutionary history of a particular object - every schema definition
#Lineage: {
    // MetaSchema governs the shape of all schema expressed in a lineage. It is
    // unconstrained/top in the base case.
    //
    // A lineage's MetaSchema may never (backwards incompatibly) change as the
    // lineage evolves.
    //
    // TODO can this be a def? should it?
    metaSchema: _

    // A Sequence is an ordered list of schema, with the invariant that
    // successive schemas are backwards compatible with their predecessors.
    #Sequence: [metaSchema, ...metaSchema]
    // TODO check subsumption (backwards compat) of each schema with its successor

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

    #Lens: {
        // The last schema in the previous sequence; logical predecessor
        ancestor: metaSchema
        // The first schema in this sequence; logical successor
        descendant: metaSchema
        forward: {
            to: descendant
            from: ancestor
            rel: descendant
            lacunae: [...#Lacuna]
        }
        reverse: {
            to: ancestor
            from: descendant
            rel: ancestor
            lacunae: [...#Lacuna]
        }
    }

    // TODO may (?) be possible to do this directly on Lens def if CUE adds
    // relative-list-index keywords
    for lv, l in seqs {
        if lv < len(seqs)-1 {
            let nextl = seqs[lv+1]
            nextl.lens.ancestor: l.schemas[len(l.schemas)-1]
            nextl.lens.descendant: nextl.schemas[0]
        }
    }
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
#SchemaVersion: [int, int]