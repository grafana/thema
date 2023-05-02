package exemplars

import "github.com/grafana/thema"

rename: {
	description: "A field is renamed - a breaking change, necessitating a new sequence."
	l:           thema.#Lineage & {
		schemas: [{
			version: [0, 0]
			schema: {
				before:    string
				unchanged: string
			}
		}, {
			version: [1, 0]
			schema: {
				after:     string
				unchanged: string
			}
		}]

		lenses: [{
			to: [0, 0]
			from: [1, 0]
			input: _
			result: {
				before:    input.after
				unchanged: input.unchanged
			}
			lacunas: []
		}, {
			to: [1, 0]
			from: [0, 0]
			input: _
			result: {
				after:     input.before
				unchanged: input.unchanged
			}
			lacunas: []
		}]
	}
}
