package exemplars

import "github.com/grafana/scuemata"

single: {
    description: "Lineage containing one sequence with a single, trivial schema."
    l: scuemata.#Lineage & {
        Seqs: [
            {
                schemas: [
                    {
                        astring: string
                        anint: int
                        abool: bool
                    }
                ]
            }
        ]
    }
}