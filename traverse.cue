package scuemata

// TODO functionize
#SearchAndValidate: {
    args: {
        lin: #Lineage
        r: lin.JoinSchema
    }
    out: #ValidatedResource | *_|_

    // Disjunction approach. Probably a bad idea to use at least until
    // disjunction performance is addressed, and maybe just in general.
    // let allsch = or([for seqv, seq in args.lin.seqs {
        // for schv, sch in seq.schemas { 
            // TODO what about non-struct schemas?
            // TODO can we unify a hidden field onto a closed sch? can't imagine so
            // sch & { _v: [seqv, schv]}}
        // }
    // ])

    let iallsch = for seqv, seq in args.lin.seqs {
        for schv, sch in seq.schemas {
            // TODO need (?) proper validation check here, not unification
            // TODO object headers especially important here
            if ((sch & args.r) | *_|_) != _|_ {
                out: #ValidatedResource & {
                    _v: [seqv, schv]
                    _lin: args.lin
                    r: r
                }
            }
        }
    }
}

// A ValidatedResource represents a resource that is valid with respect to at
// least one schema in a particular lineage.
#ValidatedResource: {
    r: _lin.JoinSchema
    _lin: #Lineage
    _v: #SchemaVersion

    out: {
        r: _lin.JoinSchema
        lacunae: [...#Lacuna]
    }
}

#SearchCriteria: {
    lin: #Lineage
    from: #SchemaVersion
    to: #SchemaVersion & [<=lin._latest[0], <len(lin.seqs[to[0]].schemas)]
}

Latest: #SearchCriteria & {
    lin: #Lineage
    to: lin._latest
}

LatestWithinSequence: #SearchCriteria & {
    lin: #Lineage
    from: #SchemaVersion
    to: [from[0], len(lin.seqs[from[0]].schemas)]
}

Exact: #SearchCriteria & {
    to: #SchemaVersion
}