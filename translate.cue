package thema

import "list"

// Translate takes a resource, a lineage, and a rule for deciding a target
// schema version. The resource is iteratively transformed through the lineage's
// list of schemas, starting at the version the resource is valid against, and
// continuing until the target schema version is reached.
//
// The out values are the resource in final translated form, the schema versions
// at which the translation started and ended, and any lacunae emitted during
// translation.
//
// TODO functionize
// TODO backwards-translation is not yet supported
#Translate: {
    resource: lin.JoinSchema
    lin: #Lineage
    to: #SearchCriteria

    _transl: {
        init: #ValidatedResource
        schemarange: [..._#vSch]

        _#step: {
            resource: lin.JoinSchema
            v: #SchemaVersion
            lacunae: [...#Lacuna]
        }

        // The accumulator holds the results of each translation step.
        accum: list.Repeat([_#step], len(schemarange)+1)
        accum: [{ resource: init.r, v: init._v, lacunae: [] }, for i, vsch in schemarange {
            let lastr = accum[i-1]
            v: vsch.v

            if vsch.v[0] == lastr._v[0] {
                // Same sequence. Translation is through implicit lens; simple unification.

                // NOTE this unification drags along defaults; it's one of
                // the key places where thema is maybe-sorta implicitly assuming
                // its inputs are concrete resources, and won't work quite right
                // with incomplete CUE structures
                resource:  lastr.resource.r & (#Pick & { lin: lin, v: vsch }).out
                lacunae: []
            }

            if vsch.v[0] > lastr._v[0] {
                // Crossing sequences. Translate via the explicit lens.

                // Feed the lens "from" input with the resource output of the
                // last translation (or init)
                let lens = { from: lastr.resource } & lin.seqs[vsch.v[0]].lens.forward
                resource: lens.translated
                lacunae: lens.lacunae
            }
        }]

        out: {
            from: init._v
            to: accum[len(accum)-1].v
            resource: accum[len(accum)-1].v
            lacunae: [for step in accum if len(step.lacunae) > 0 { v: step.v, lacunae: step.lacunae }]
        }
    }

    let rarg = (#SearchAndValidate & { resource: resource, lin: lin }).out
    // FIXME Must necessarily anchor translation at the input resource's schema
    // version. Nevertheless, this has an unfortunate, magical smell.
    to: from: rarg._v
    let cmp = (_cmpSV & { l: rarg._v, r: to.to }).out
    out: {
        if cmp == 0 {
            (_transl & { init: rarg, schemarange: [] }).out
        }
        if cmp == -1 {
            let lo = (_flatidx & { lin: lin, rarg._v}).fidx
            let hi = (_flatidx & { lin: lin, to.to[0]}).fidx
            (_transl & { init: rarg, schemarange: (_all & { lin: lin }).out[lo+1:hi]}).out
        }
        if cmp == 1 {
            // FIXME For now, we don't support backwards translation. This must change.
            _|_
        }
    }
}
