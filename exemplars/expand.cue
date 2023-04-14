package exemplars

import "github.com/grafana/thema"

expand: {
	description: "A few schema in a single sequence, illustrating some simple expansions permitted by backwards compatibility rules."
	l:           thema.#Lineage & {
		schemas: [{
			version: [0, 0]
			schema: {
				init: string
			}
		},
			{
				version: [0, 1]
				schema: {
					init:      string
					optional?: int
				}
			},
			{
				version: [0, 2]
				schema: {
					init:         string
					optional?:    int
					withDefault?: *"foo" | "bar"
				}
			},
			{
				version: [0, 3]
				schema: {
					init:         string
					optional?:    int
					withDefault?: *"foo" | "bar" | "baz"
				}
			}]

		lenses: [{
			to: [0, 0]
			from: [0, 1]
			input: _
			result: {
				init: input.init
			}
		},
			{
				to: [0, 1]
				from: [0, 2]
				input: _
				result: {
					init: input.init
					if input.optional != _|_ {
						optional: input.optional
					}
				}
			},
			{
				to: [0, 2]
				from: [0, 3]
				input: _
				result: {
					init: input.init
					if input.optional != _|_ {
						optional: input.optional
					}
					if input.withDefault != _|_ {
						// if the value is "baz" (not allowed by the to schema), then the to
						// schema's default value "foo" will be selected by Thema's #Translate
						withDefault: input.withDefault
					}
				}
			}]
	}
}
