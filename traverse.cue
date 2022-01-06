package thema

// TODO functionize
#SearchAndValidate: {
    lin: #Lineage
    inst: lin.joinSchema
    out: #LinkedInstance

    let ininst = inst
    let inlin = lin
    let all = (_all & { lin: inlin }).out
    out: [for _, vSch in all {
        #LinkedInstance & {
            v: vSch.v
            lin: inlin
            inst: ininst
        }
    }][0]
}

// #LinkedInstance represents data that is an instance of some schema, the
// version of that schema, and the lineage from which they all hail.
#LinkedInstance: {
    inst: lin.joinSchema
    lin: #Lineage
    v: #SyntacticVersion

    // TODO need proper validation/subsumption check here, not simple unification
    _valid: inst & lin.seqs[v[0]].schemas[v[1]]
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
    from: #SyntacticVersion
    to: [from[0], len(lin.seqs[from[0]].schemas)]
}

// common type over #Latest and #LatestWithinSequence
_#resolver: {
    lin: #Lineage
    from?: #SyntacticVersion
    to: #SyntacticVersion & [<=lin._latest[0], <len(lin.seqs[to[0]].schemas)]
}