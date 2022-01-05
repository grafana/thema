package exemplars

import "github.com/grafana/thema"

_#Exemplar: {
    l: thema.#Lineage
    description: string
    // tt: [string]: {
    //     r: l.joinSchema
    //     to: thema.#SearchCriteria
    //     expect: {
    //         to: l.joinSchema
    //     }
    // }
}

[N=string]: _#Exemplar & {
    l: name: N
}

// Cases to create
// 
// 5. Complex combination and remapping of fields across seqs
// 6. Subtype/constrained joinSchema

// Composed cases
//
// 1. Composed single sub-lineage
// 2. Composed multi-sublineage
