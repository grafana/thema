package exemplars

import "github.com/grafana/thema"

expand: {
	description: "A few schema in a single sequence, illustrating some simple expansions permitted by backwards compatibility rules."
	l:           thema.#Lineage & {
		schemas: [{
			version: [0, 0]
			schema: init: string
		}, {
			version: [0, 1]
			schema: {
				init:      string
				optional?: int // 0.1
			}
		}, {
			version: [0, 2]
			schema: {
				init:         string
				optional?:    int
				withDefault?: *"foo" | "bar" // 0.2
			}
		}, {
			version: [0, 3]
			schema: {
				init:         string
				optional?:    int
				withDefault?: *"foo" | "bar" | "baz" // 0.3
			}
		}]
		lenses: [{
			to: [0, 0]
			from: [0, 1]
			input: _
			result: {
				_|_// TODO implement this lens
			}
			lacunas: []
		}, {
			to: [0, 1]
			from: [0, 2]
			input: _
			result: {
				_|_// TODO implement this lens
			}
			lacunas: []
		}, {
			to: [0, 2]
			from: [0, 3]
			input: _
			result: {
				_|_// TODO implement this lens
			}
			lacunas: []
		}]
	}
}
