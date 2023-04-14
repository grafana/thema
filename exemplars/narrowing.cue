package exemplars

import "github.com/grafana/thema"

narrowing: {
	description: "Lineage that narrows a sloppily-specified boolean/string-ish type to a proper boolean over a single breaking change."
	l:           thema.#Lineage & {
		schemas: [{
			version: [0, 0]
			schema: boolish: "true" | "false" | bool | string
		}, {
			version: [1, 0]
			schema: properbool: bool
		}]
		lenses: [{
			to: [0, 0]
			from: [1, 0]
			input: _
			result: {
				// Preserving precise original form is a non-goal of thema in general.
				boolish: input.properbool
			}
			lacunas: []
		}, {
			to: [1, 0]
			from: [0, 0]
			input: _
			result: {
				if ((input.boolish & string) != _|_) {
					properbool: input.boolish == "true"
				}
				if ((input.boolish & bool) != _|_) {
					properbool: input.boolish
				}
			}
			lacunas: [
				if ((input.boolish & string) != _|_) && ((input.boolish & ("true" | "false")) == _|_) {
					thema.#Lacuna & {
						sourceFields: [{
							path:  "boolish"
							value: input.boolish
						}]
						targetFields: [{
							path:  "properbool"
							value: result.properbool
						}]
						message: "boolish was a string but neither \"true\" nor \"false\"; fallback to treating as false"
						type:    thema.#LacunaTypes.LossyFieldMapping
					}
				},
			]
		}]
	}
}
