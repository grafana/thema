package exemplars

import "github.com/grafana/scuemata"

_#Exemplar: {
    lin: scuemata.#Lineage
    description: string
    tt: [string]: {
        r: lin.JoinSchema
        to: scuemata.#SearchCriteria
        expect: {
            to: lin.JoinSchema
        }
    }
}

[N=string]: _#Exemplar & {
    lin: Name: N
}

// Cases to create
// 
// 5. Complex combination and remapping of fields across seqs
// 6. Subtype/constrained JoinSchema
// 7. Change to defaults across seqs

// Composed cases
//
// 1. Composed single sub-lineage
// 2. Composed multi-sublineage
