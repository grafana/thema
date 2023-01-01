package thema

// SearchAndValidate is a pseudofunction that takes a lineage and some candidate
// data as an argument, and searches the lineage for a schema against which
// that data is valid.
// TODO functionize
#SearchAndValidate: fn={
	lin:  #Lineage
	inst: {...}
	out:  #LinkedInstance

//    let ininst = inst
//    let inlin = lin
    out: [for _, sch in fn.lin.schemas if ((fn.inst & sch._#schema) != _|_) {
		v: sch.version
		lin: lin
		inst: fn.inst & sch._#schema
    }][0]
}

#ValidFor: fn2={
	lin:  #Lineage
	inst: {...}
//	out: #SyntacticVersion

//    let ininst = inst
//    let inlin = lin
    out: [for _, sch in fn2.lin.schemas if ((fn2.inst & sch._#schema) != _|_) { sch.version }][0]
}

// #LinkedInstance represents data that is an instance of some schema, the
// version of that schema, and the lineage of the schema.
#LinkedInstance: {
//	inst: I.lin.joinSchema
	inst: {...}
	lin:  #Lineage
	v:    #SyntacticVersion

    // TODO need proper validation/subsumption check here, not simple unification
    _valid: inst & (#Pick & {lin: lin, v: v}).out
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
