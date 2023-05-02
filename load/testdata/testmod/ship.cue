package testmod

import "github.com/grafana/thema"

lin: thema.#Lineage
lin: name: "Ship"
lin: {
	schemas: [{
		version: [0, 0]
		schema: firstfield: string
	}, {
		version: [1, 0]
		schema: {
			firstfield:  string
			secondfield: int // 1.0
		}
	}]
	lenses: [{
		to: [0, 0]
		from: [1, 0]
		input: _
		result: {
			// Map the first field back
			firstfield: input.firstfield
		}
		lacunas: []
	}, {
		to: [1, 0]
		from: [0, 0]
		input: _
		result: {
			firstfield:  input.firstfield
			secondfield: -1
		}
		lacunas: [
			thema.#Lacuna & {
				targetFields: [{
					path:  "secondfield"
					value: result.secondfield
				}]
				message: "-1 used as a placeholder value - replace with a real value before persisting!"
				type:    thema.#LacunaTypes.Placeholder
			},
		]
	}]
}
