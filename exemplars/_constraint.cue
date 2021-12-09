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
// 1. Narrowing of sloppy type
// 2. Simple, single schema case
// 3. Multiple schema, one sequence, simple field addition but also disjunction expansion
// 4. Simple rename of fields across seqs
// 5. Complex combination and remapping of fields across seqs
// 6. Subtype/constrained JoinSchema
// 7. Change to defaults across seqs

// Composed cases
//
// 1. Composed single sub-lineage
// 2. Composed multi-sublineage
