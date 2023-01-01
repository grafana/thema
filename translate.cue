package thema

import "list"

// Translate takes a instance, a lineage, and a rule for deciding a target
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

//	let VF = linst.v
//	let VT = to
//	let inlinst = linst
//	let inlinst = T.linst
//	let inlin = inlinst

	let cmp = (_cmpSV & { l: T.linst.v, r: T.to }).out

	out: {
		linst: #LinkedInstance
		lacunas: [...{
			v: #SyntacticVersion
			lacunas: [...#Lacuna]
		}]
	}
	out: [
		// FIXME For now, we don't support backwards translation. This must change.
		if cmp == 1 {_|_},
		if cmp == 0 { linst: T.linst, lacunas: [] },
		if cmp == -1 {
			(_forward & { VF: T.linst.v, VT: T.to, inst: T.linst.inst, lin: T.lin }).out
		}
	][0]

//    _transl: {
//        schemarange: [...#SchemaDecl]
//
//		_#step: {
////			inst: inlinst.lin.joinSchema
//			inst: _
//			v:    #SyntacticVersion
//			lacunas: [...#Lacuna]
//		}
//
//		// The accumulator holds the results of each translation step.
//		accum: list.Repeat([_#step], len(schemarange)+1)
//		accum: [{inst: inlinst.inst, v: VF, lacunas: []}, for i, vsch in schemarange {
//			let lasti = accum[i]
//			v: vsch.v
//
//			if vsch.v[0] == lasti.v[0] {
//				// Same sequence. Translation is through the implicit lens;
//				// simple unification.
//
//				// NOTE this unification drags along defaults; it's one of
//				// the key places where thema is maybe-sorta implicitly assuming
//				// its inputs are concrete instances, and won't work quite right
//				// with incomplete CUE structures
//				inst: lasti.inst & (#Pick & { lin: inlin, v: vsch.v }).out
//				lacunas: []
//			}
//			if vsch.v[0] > lasti.v[0] {
//				// Crossing sequences. Translate via the explicit lens.
//
//				// Feed the lens "from" input with the instance output of the
//				// last translation
//				let _lens = {from: lasti.inst} & inlin.seqs[vsch.v[0]].lens.forward
//				inst:    _lens.translated
//				lacunas: _lens.lacunas
//			}
//		}]
//
//		out: {
//			linst: {
//				inst: accum[len(accum)-1].inst
//				v:    accum[len(accum)-1].v
//				lin:  inlin
//			}
//			lacunas: [ for step in accum if len(step.lacunas) > 0 {v: step.v, lacunas: step.lacunas}]
//		}
//	}

//    schrange: {
//        if cmp == 0 { [] },
//        if cmp == -1 {
//            let lo = (_flatidx & { lin: inlin, v: VF }).out
//            let hi = (_flatidx & { lin: inlin, v: VT }).out
//            list.Slice(inlin._all, lo+1, hi+1)
//        },
//		// FIXME For now, we don't support backwards translation. This must change.
//        if cmp == 1 { _|_ },
//    }

//    schrange: (({
//    	op: 0
//    	out: []
//    } | {
//    	op: -1
//		let lo = (_flatidx & { lin: inlin, v: VF }).out
//		let hi = (_flatidx & { lin: inlin, v: VT }).out
//    	out: list.Slice(inlin._all, lo+1, hi+1)
//    } | {
//    	op: 1
//		// FIXME For now, we don't support backwards translation. This must change.
//		out: _|_
//    }) & (_cmpSV & { l: VF, r: VT })).out

//	out: (_transl & {schemarange: schrange}).out
}

_forward: {
	// lineage we're operating on
	lin: #Lineage
	// version translating from
	VF: #SyntacticVersion
	// version translating to
	VT: #SyntacticVersion
	// starting data instance
	inst: {...}

	let lo = (_flatidx & { lin: lin, v: VF }).out
	let hi = (_flatidx & { lin: lin, v: VT }).out
	out: (_fn & { schemarange: list.Slice(lin._all, lo+1, hi+1)}).out

	_fn: {
        schemarange: [...#SchemaDecl]

		_#step: {
			inst: _
			version:    #SyntacticVersion
			lacunas: [...#Lacuna]
		}

		// The accumulator holds the results of each translation step.
		accum: list.Repeat([_#step], len(schemarange)+1)
		accum: [{inst: inst, version: VF, lacunas: []}, for i, vsch in schemarange {
			let lasti = accum[i]
			v: vsch.version

			if vsch.version[0] == lasti.version[0] {
				// Same major version. Translation is through the implicit lens -
				// simple unification.

				// NOTE this unification drags along defaults; it's one of
				// the key places where thema is maybe-sorta implicitly assuming
				// its inputs are concrete instances, and won't work quite right
				// with incomplete CUE structures
				inst: lasti.inst & vsch._#schema
				lacunas: []
			}
			if vsch.version[0] > lasti.version[0] {
				// Crossing major versions. Translate via the explicit lens transform.

				// Feed the lens "from" input with the instance output of the
				// last translation
				let x = { from: lasti.inst } & vsch.lens.forward
				inst: x.map & x.to
				lacunas: x.lacunas
//				let _lens = {from: lasti.inst} & inlin.seqs[vsch.version[0]].lens.forward
//				inst:    _lens.translated
//				lacunas: _lens.lacunas
			}
		}]

		out: {
			linst: {
				inst: accum[len(accum)-1].inst
				version:    accum[len(accum)-1].version
				lin:  lin
			}
			lacunas: [ for step in accum if len(step.lacunas) > 0 {v: step.version, lacunas: step.lacunas}]
		}
	}
}

#TranslatedInstance: {
	linst: #LinkedInstance
}