package thema

import "list"

// TODO docs
#TranslatedInstance: {
	#LinkedInstance
	from: #SyntacticVersion
	lacunas: [...#Lacuna]
}

// Translate takes an instance, a lineage, and a rule for deciding a target
// schema version. The instance is iteratively transformed through the lineage's
// list of schemas, starting at the version the instance is valid against, and
// continuing until the target schema version is reached.
//
// The out values are the instance in final translated form, the schema versions
// at which the translation started and ended, and any lacunas emitted during
// translation.
//
// TODO backwards-translation is not yet supported
// TODO functionize
#Translate: T={
	linst: #LinkedInstance
	to:    #SyntacticVersion

	//		let VF = linst.v
	//		let VT = to
	//	let inlinst = linst
	//	let inlinst = T.linst

	//	let inlin = inlinst.lin
	let inlin = linst.lin

	let cmp = (_cmpSV & {l: T.linst.v, r: T.to}).out

	out: {
		steps: [...#TranslatedInstance]
		result: steps[len(steps)-1]
	}
	out: {
		steps: [
			// to version same as from version
			if cmp == 0 {result: T.linst, lacunas: []},
			// to version older/smaller than from version
			if cmp == 1 {
				let hi = (_flatidx & {lin: inlin, v: linst.v}).out

				let lo = (_flatidx & {lin: inlin, v: to}).out

				// schrange is the subset of schemas being traversed in this
				// translation, inclusive of the starting schema. It is in descending
				// order, such that iterating the list moves backwards through schema versions.
				//				let schrange = list.Sort(list.Slice(inlin.schemas, lo, hi),
				//				{
				//					x:    #SchemaDef
				//					y:    #SchemaDef
				//					less: (_cmpSV & {l: x, r: y}).out == -1
				//				},
				//				)

				// schrange is the subset of schemas being traversed in this
				// translation, inclusive of the starting schema. It is in descending
				// order, such that iterating the list moves backwards through schema versions.
				let schrange = list.Slice(inlin.schemas, lo, hi)

				_accum: [linst, for i, pos in list.Repeat(hi, lo-1, -1) {
					// alias pointing to the previous item in the list we're building
					let prior = _accum[i]

					// the actual schema def
					let schdef = schrange[pos]

					// "call" predecessor pseudofunc to process the object through the lens
					(_predecessor & {
						lin: inlin
						VF:  prior.v
						VT:  schdef.version
						arg: prior.instance
					}).out
				}]
				list.Slice(_accum, 1, -1)
			},
			// to version newer/larger than from version
			if cmp == -1 {
				let lo = (_flatidx & {lin: inlin, v: linst.v}).out
				let hi = (_flatidx & {lin: inlin, v: to}).out

				// _schrange contains the subset of schemas being traversed in this
				// translation, inclusive of the starting schema.
				let schrange = list.Slice(inlin.schemas, lo, hi)
				_accum: [linst, for i, schdef in schrange {
					// alias pointing to the previous item in the list we're building
					let prior = _accum[i]

					// "call" predecessor pseudofunc to process the object through the lens
					(_successor & {
						lin: inlin
						VF:  prior.v
						VT:  schdef.version
						arg: prior.instance
					}).out
				}]
				list.Slice(_accum, 1, -1)

				//			_accum: list.Repeat([#TranslatedInstance], hi-lo+1)
				//			_accum: [linst, for i, vsch in list.Slice(inlin.schemas, lo+1, hi+1) {
				//				let lasti = _accum[i]
				//			}]
				//			_accum: [linst, for i in list.Range(1, , 1)]
				//
				//			(_forward & {VF: T.linst.v, VT: T.to, inst: T.linst.inst, lin: T.lin}).out
			},
		][0]
	}
}

_successor: {
	// lineage we're operating on
	lin: #Lineage
	// version translating from
	VF: #SyntacticVersion
	// version translating to
	VT: #SyntacticVersion
	// starting data instance
	arg: {...}

	out: #TranslatedInstance & {
		linst: {
			lin:     lin
			version: VT
		}
	}
	let sch = (#Pick & {lin: lin, v: VT}).out
	if VF[0] < VT[0] {
		// Crossing a major version. The forward lens explicitly defined in the schema
		// provides the mapping algorithm.
		out: {
			let L = sch.lenses & {input: prior: arg}
			inst: L.priorToSelf.result
			lacunas: [ for lac in L.priorToSelf.lacunas if lac.condition {lac}]
		}
	}

	if VF[0] == VT[0] {
		// Only a minor version. Backwards compatibility rules dictate that the mapping
		// algorithm for the forward lens is generic: simple unification.
		out: inst: arg & sch._#schema
	}
}

_predecessor: {
	// lineage we're operating on
	lin: #Lineage
	// version translating from
	VF: #SyntacticVersion
	// version translating to
	VT: #SyntacticVersion
	// starting data instance
	arg: {...}

	let sch = (#Pick & {lin: lin, v: VT}).out
	let L = sch.lenses & {input: self: arg}

	out: #TranslatedInstance & {
		inst: L.selfToPrior.result
		v:    VT
		from: VF
		lacunas: [ for lac in L.selfToPrior.lacunas if lac.condition {lac.lacuna}]
		lin: lin
	}
}
