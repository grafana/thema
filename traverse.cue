package thema

// TODO functionize
#SearchAndValidate: {
    lin: #Lineage
    resource: lin.JoinSchema
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

    out: [for seqv, seq in lin.seqs {
        // TODO need (?) proper subsumption validation check here, not unification
        for schv, sch in seq.schemas if ((sch & resource) | *_|_) != _|_ {
            // TODO object headers especially important here
            #ValidatedResource & {
                _v: [seqv, schv]
                _lin: lin
                resource: resource
            }
        }
    }][0]
}

// A ValidatedResource represents a resource, and the schema from a particular
// lineage that it validates against.
#ValidatedResource: {
    r: _lin.JoinSchema
    _lin: #Lineage
    _v: #SchemaVersion

    // TODO need proper validation check here, not simple unification
    _valid: r & _lin.seqs[_v[0]].schemas[_v[1]]
}


#SearchCriteria: {
    lin: #Lineage
    from: #SchemaVersion
    to: #SchemaVersion & [<=lin._latest[0], <len(lin.seqs[to[0]].schemas)]
}

// Latest indicates that traversal should continue until the latest schema in
// the entire lineage is reached.
#Latest: #SearchCriteria & {
    lin: #Lineage
    to: lin._latest
}

// LatestWithinSequence indicates that, given a starting schema version (or a
// resource, whose version will be extracted), traversal should continue to the
// latest version within the starting version's sequence.
#LatestWithinSequence: #SearchCriteria & {
    lin: #Lineage
    from: #SchemaVersion
    fromResource?: lin.JoinSchema
    if fromResource != _|_ {
        from: (#SearchAndValidate & { resource: fromResource, lin: lin }).out._v
    }
    to: [from[0], len(lin.seqs[from[0]].schemas)]
}

// Exact indicates traversal should continue until an exact, explicitly
// specified version is reached.
#Exact: #SearchCriteria & {
    to: #SchemaVersion
}