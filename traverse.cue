package thema

// TODO functionize
#SearchAndValidate: {
	lin:  #Lineage
	inst: lin.joinSchema
	out:  #LinkedInstance

    let ininst = inst
    let inlin = lin
    out: [for _, sch in lin.schemas {
        #LinkedInstance & {
            v: sch.version
            lin: inlin
            inst: ininst
        }
    }][0]
}

// #LinkedInstance represents data that is an instance of some schema, the
// version of that schema, and the lineage from which they all hail.
#LinkedInstance: {
	inst: lin.joinSchema
	lin:  #Lineage
	v:    #SyntacticVersion

    // TODO need proper validation/subsumption check here, not simple unification
    _valid: inst & (#Pick & {lin: lin}).out
}

// Latest indicates that traversal should continue until the latest schema in
// the entire lineage is reached.
#Latest: {
    lin: #Lineage
    to: (#LatestVersion & { lin: lin }).out
}

// LatestWithinSequence indicates that, given a starting schema version,
// traversal should continue to the latest version within the starting version's
// sequence.
#LatestWithinSequence: {
    lin: #Lineage
    from: #SyntacticVersion
    to: [from[0], lin._counts[from[0]]-1]
}
