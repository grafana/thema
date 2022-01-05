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
    inst: lin.joinSchema
    lin: #Lineage
    to: #SearchCriteria

    _transl: {
        init: #LinkedInstance
        schemarange: [..._#vSch]

        _#step: {
            inst: lin.joinSchema
            v: #SchemaVersion
            lacunae: [...#Lacuna]
        }

        // The accumulator holds the results of each translation step.
        accum: list.Repeat([_#step], len(schemarange)+1)
        accum: [{ inst: init.inst, v: init._v, lacunae: [] }, for i, vsch in schemarange {
            let lasti = accum[i-1]
            v: vsch.v

            if vsch.v[0] == lasti._v[0] {
                // Same sequence. Translation is through the implicit lens;
                // simple unification.

                // NOTE this unification drags along defaults; it's one of
                // the key places where thema is maybe-sorta implicitly assuming
                // its inputs are concrete instances, and won't work quite right
                // with incomplete CUE structures
                inst:  lasti.inst.inst & (#Pick & { lin: lin, v: vsch }).out
                lacunae: []
            }

            if vsch.v[0] > lasti._v[0] {
                // Crossing sequences. Translate via the explicit lens.

                // Feed the lens "from" input with the instance output of the
                // last translation (or init)
                let lens = { from: lasti.inst } & lin.seqs[vsch.v[0]].lens.forward
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

    let inputinst = (#SearchAndValidate & { inst: inst, lin: lin }).out
    // FIXME Must necessarily anchor translation at the input instance's schema
    // version. Nevertheless, this has an unfortunate, magical smell.
    to: from: inputinst._v
    let cmp = (_cmpSV & { l: inputinst._v, r: to.to }).out
    out: {
        if cmp == 0 {
            (_transl & { init: inputinst, schemarange: [] }).out
        }
        if cmp == -1 {
            let lo = (_flatidx & { lin: lin, inputinst._v}).fidx
            let hi = (_flatidx & { lin: lin, to.to[0]}).fidx
            (_transl & { init: inputinst, schemarange: (_all & { lin: lin }).out[lo+1:hi]}).out
        }
        if cmp == 1 {
            // FIXME For now, we don't support backwards translation. This must change.
            _|_
        }
    }
}
