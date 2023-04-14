package exemplars

import "github.com/grafana/thema"

single: {
	description: "Lineage containing one sequence with a single, trivial schema."
	l:           thema.#Lineage & {
		schemas: [{
			version: [0, 0]
			schema: {
				astring: string
				anint:   int
				abool:   bool
			}
		}]
		lenses: []
	}
}
