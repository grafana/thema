package thema

import "list"

// Translate takes a instance, a lineage, and a rule for deciding a target
// schema version. The instance is iteratively transformed through the lineage's
// list of schemas, starting at the version the instance is valid against, and
// continuing until the target schema version is reached.
//
// The out values are the instance in final translated form, the schema versions
// at which the translation started and ended, and any lacunae emitted during
// translation.
//
// TODO functionize
// TODO backwards-translation is not yet supported
#Translate: {
    linst: #LinkedInstance
    to: #SchemaVersion

    // make em stand out
    let VF = linst._v
    let VT = to

    _transl: {
        init: #LinkedInstance
        schemarange: [..._#vSch]

        _#step: {
            inst: init._lin.joinSchema
            v: #SchemaVersion
            lacunae: [...#Lacuna]
        }

        // The accumulator holds the results of each translation step.
        accum: list.Repeat([_#step], len(schemarange)+1)
        accum: [{ inst: init.inst, v: VF, lacunae: [] }, for i, vsch in schemarange {
            let lasti = accum[i-1]
            v: vsch.v

            if vsch.v[0] == lasti._v[0] {
                // Same sequence. Translation is through the implicit lens;
                // simple unification.

                // NOTE this unification drags along defaults; it's one of
                // the key places where thema is maybe-sorta implicitly assuming
                // its inputs are concrete instances, and won't work quite right
                // with incomplete CUE structures
                inst:  lasti.inst.inst & (#Pick & { lin: inst._lin, v: vsch }).out
                lacunae: []
            }

            if vsch.v[0] > lasti._v[0] {
                // Crossing sequences. Translate via the explicit lens.

                // Feed the lens "from" input with the instance output of the
                // last translation (or init)
                let lens = { from: lasti.inst } & inst._lin.seqs[vsch.v[0]].lens.forward
                inst: lens.translated
                lacunae: lens.lacunae
            }
        }]

        out: {
            from: init._v
            to: accum[len(accum)-1].v
            inst: accum[len(accum)-1].v
            lacunae: [for step in accum if len(step.lacunae) > 0 { v: step.v, lacunae: step.lacunae }]
        }
    }

    out: {
        let cmp = (_cmpSV & { l: VF, r: VT }).out
        if cmp == 0 {
            (_transl & { init: linst, schemarange: [] }).out
        }
        if cmp == -1 {
            let lo = (_flatidx & { lin: linst._lin, VT}).fidx
            let hi = (_flatidx & { lin: linst._lin, VT[0]}).fidx
            (_transl & { init: linst, schemarange: (_all & { lin: linst._lin }).out[lo+1:hi]}).out
        }
        if cmp == 1 {
            // FIXME For now, we don't support backwards translation. This must change.
            _|_
        }
    }
}
