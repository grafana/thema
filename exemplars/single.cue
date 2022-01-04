package exemplars

import "github.com/grafana/thema"

single: {
    description: "Lineage containing one sequence with a single, trivial schema."
    l: thema.#Lineage & {
        seqs: [
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