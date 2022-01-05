package thema

// TODO functionize
#SearchAndValidate: {
    lin: #Lineage
    inst: lin.joinSchema
    out: #LinkedInstance | *_|_

    // Disjunction approach. Probably a bad idea to use at least until
    // disjunction performance is addressed, and maybe just in general.
    // let allsch = or([for seqv, seq in args.lin.seqs {
        // for schv, sch in seq.schemas { 
            // TODO what about non-struct schemas?
            // TODO can we unify a hidden field onto a closed sch? can't imagine so
            // sch & { _v: [seqv, schv]}}
        // }
    // ])

    out: [for seqv, seq in lin.seqs {
        // TODO need (?) proper subsumption validation check here, not unification
        for schv, sch in seq.schemas if ((sch & inst) | *_|_) != _|_ {
            // TODO object headers especially important here
            #LinkedInstance & {
                _v: [seqv, schv]
                _lin: lin
                inst: inst
            }
        }
    }][0]
}

// #LinkedInstance represents data that is an instance of some schema, that
// schema, the version of that schema, and the lineage from which they all hail.
#LinkedInstance: {
    inst: _lin.joinSchema
    _lin: #Lineage
    _v: #SchemaVersion

    // TODO need proper validation/subsumption check here, not simple unification
    _valid: inst & _lin.seqs[_v[0]].schemas[_v[1]]
}

// Latest indicates that traversal should continue until the latest schema in
// the entire lineage is reached.
#Latest: _#resolver & {
    lin: #Lineage
    to: (_latest & { lin: lin }).out
}

// LatestWithinSequence indicates that, given a starting schema version,
// traversal should continue to the latest version within the starting version's
// sequence.
#LatestWithinSequence: _#resolver & {
    lin: #Lineage
    from: #SchemaVersion
    to: [from[0], len(lin.seqs[from[0]].schemas)]
}

// common type over #Latest and #LatestWithinSequence
_#resolver: {
    lin: #Lineage
    from?: #SchemaVersion
    to: #SchemaVersion & [<=lin._latest[0], <len(lin.seqs[to[0]].schemas)]
}