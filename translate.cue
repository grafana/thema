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
    to: #SyntacticVersion

    // Shape of output
    out: {
        linst: #LinkedInstance
        lacunae: [...{
            v: #SyntacticVersion
            lacunae: [...#Lacuna]
        }]
    }

    let VF = linst.v
    let VT = to
    let inlin = linst.lin
    let inlinst = linst

    _transl: {
        schemarange: [..._#vSch]

        _#step: {
            inst: inlinst.lin.joinSchema
            v: #SyntacticVersion
            lacunae: [...#Lacuna]
        }

        // The accumulator holds the results of each translation step.
        accum: list.Repeat([_#step], len(schemarange)+1)
        accum: [{ inst: inlinst.inst, v: VF, lacunae: [] }, for i, vsch in schemarange {
            let lasti = accum[i]
            v: vsch.v

            if vsch.v[0] == lasti.v[0] {
                // Same sequence. Translation is through the implicit lens;
                // simple unification.

                // NOTE this unification drags along defaults; it's one of
                // the key places where thema is maybe-sorta implicitly assuming
                // its inputs are concrete instances, and won't work quite right
                // with incomplete CUE structures
                // inst: lasti.inst & (#Pick & { lin: inlin, v: vsch.v }).out
                inst: lasti.inst & inlin.seqs[vsch.v[0]].schemas[vsch.v[1]]
                lacunae: []
            }
            if vsch.v[0] > lasti.v[0] {
                // Crossing sequences. Translate via the explicit lens.

                // Feed the lens "from" input with the instance output of the
                // last translation
                let _lens = { from: lasti.inst } & inlin.seqs[vsch.v[0]].lens.forward
                inst: _lens.translated
                lacunae: _lens.lacunae
            }
        }]

        out: {
            linst: {
                inst: accum[len(accum)-1].inst
                v: accum[len(accum)-1].v
                lin: inlin
            }
            lacunae: [for step in accum if len(step.lacunae) > 0 { v: step.v, lacunae: step.lacunae }]
        }
    }

    schrange: {
        let cmp = (_cmpSV & { l: VF, r: VT }).out
        if cmp == 0 {
            []
        }
        if cmp == -1 {
            let lo = (_flatidx & { lin: inlin, v: VF }).out
            let hi = (_flatidx & { lin: inlin, v: VT }).out
            list.Slice((_all & { lin: inlin }).out, lo+1, hi+1)
        }
        if cmp == 1 {
            // FIXME For now, we don't support backwards translation. This must change.
            _|_
        }
    }

    out: (_transl & { schemarange: schrange }).out
}
