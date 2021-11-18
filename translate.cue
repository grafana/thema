package scuemata

import "list"

// TODO functionize
#Translate: {
    args: {
        resource: _lin.JoinSchema
        lin: #Lineage
        to: #SearchCriteria
    }

    _transl: {
        init: #ValidatedResource
        domain: [..._#vSch]

        _#step: {
            resource: _lin.JoinSchema
            v: #SchemaVersion
            lacunae: [...#Lacuna]
        }

        // The accumulator holds the results of each translation step.
        accum: [list.Repeat([_#step]), len(domain)+1]
        accum: [{ resource: init.r, v: init._v, lacunae: [] }, for i, vsch in domain {
            let lastr = accum[i-1]
            v: vsch.v

            if vsch.v[0] == lastr._v[0] {
                // Same sequence. Translation is through implicit lens; simple unification.

                // NOTE this unification drags along defaults; it's one of
                // the key places where scuemata is maybe-sorta implicitly assuming
                // its inputs are concrete resources, and won't work quite right
                // with incomplete CUE structures
                resource:  lastr.resource.r & args.lin.seqs[vsch.v[0]].schemas[vsch.v[1]]
                lacunae: []
            }

            if vsch.v[0] > lastr._v[0] {
                // Crossing sequences. Translate via the explicit lens.

                // Feed the lens "from" input with the resource output of the
                // last translation (or init)
                let lens = { from: lastr.resource } & args.lin.seqs[vsch.v[0]].lens.forward
                resource: lens.translated
                lacunae: lens.lacunae
            }
        }]

        out: {
            // TODO FLATTEN
            from: init._v
            to: accum[len(accum)-1].v
            resource: accum[len(accum)-1].v
            lacunae: [for step in accum if len(step.lacunae) > 0 { v: step.v, lacunae: step.lacunae }]
        }
    }

    let rarg = (#SearchAndValidate & { args: { resource: args.resource, lin: args.lin }}).out
    let cmp = (_cmpSV & { l: rarg._v, r: args.to.to }).out
    out: {
        if cmp == 0 {
            (_transl & { init: rarg, domain: [] }).out
        }
        if cmp == -1 {
            let lo = (_flatidx & { lin: args.lin, rarg._v}).fidx
            let hi = (_flatidx & { lin: args.lin, args.to.to[0]}).fidx
            (_transl & { init: rarg, domain: args.lin._all[lo+1:hi]}).out
        }
        if cmp == 1 {
            // FIXME For now, we don't support backwards translation. This must change.
            _|_
        }
    }
}
