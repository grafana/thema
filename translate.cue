package scuemata

// TODO functionize
#Translate: {
    args: {
        r: #ValidatedResource
        to: #SearchCriteria
    }

    _transl: {
        init: #ValidatedResource
        domain: [..._#vSch]

        reducer: [...{
            r: #ValidatedResource,
            l: [...#Lacuna]
        }]

        reducer: [{ r: init, l: [] }, for i, vsch in domain {
            let lastr = reducer[i-1]
            if vsch.v[0] == lastr._v[0] {
                // Same sequence. Translation is through implicit lens; simple unification.
                r: {
                    // NOTE this unification drags along defaults; it's one of
                    // the key places where scuemata is maybe-sorta implicitly assuming
                    // its inputs are concrete resources, and won't work quite right
                    // with incomplete CUE structures
                    r: lastr.r & lastr._lin.seqs[vsch.v[0]].schemas[vsch.v[1]]
                    _v: vsch.v
                    _lin: lastr._lin
                }
                l: []
            }
            if vsch.v[0] > lastr._v[0] {
                // Crossing sequences. Translate via the explicit lens.
                let lens = lastr.r & lastr._lin.seqs[vsch.v[0]].lens.forward.from
                r: {
                    r: lens.rel & lens.to
                    _v: vsch.v
                    _lin: lastr._lin
                }
                l: lens.lacunae
            }
        }]
    }

    let cmp = (_cmpSV & { l: args.r._v, r: args.to.to }).out
    out: {
        if cmp == 0 {
            [{
                r: args.r
                lacunae: []
            }]
        }
        if cmp == -1 {
            let lo = (_flatidx & { lin: args.r._lin, args.r._v}).fidx
            let hi = (_flatidx & { lin: args.r._lin, args.to.to[0]}).fidx
            _transl & { init: args.r, domain: args.r._lin._all[lo+1:hi]}
        }
        if cmp == 1 {
            // FIXME For now, we don't support backwards translation. This must change.
            _|_
        }
    }
}
