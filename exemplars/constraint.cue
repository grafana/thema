package exemplars

import "github.com/grafana/scuemata"

_#Exemplar: {
    l: scuemata.#Lineage
    description: string
    // tt: [string]: {
    //     r: l.JoinSchema
    //     to: scuemata.#SearchCriteria
    //     expect: {
    //         to: l.JoinSchema
    //     }
    // }
}

[N=string]: _#Exemplar & {
    l: Name: N
}

// Cases to create
// 
// 5. Complex combination and remapping of fields across seqs
// 6. Subtype/constrained JoinSchema

// Composed cases
//
// 1. Composed single sub-lineage
// 2. Composed multi-sublineage
