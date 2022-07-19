package exemplars

import "github.com/grafana/thema"

expand: {
	description: "A few schema in a single sequence, illustrating some simple expansions permitted by backwards compatibility rules."
	l:           thema.#Lineage & {
		seqs: [
			{
				schemas: [
					{// 0.0
						init: string
					},
					{// 0.1
						init:      string
						optional?: int
					},
					{// 0.2
						init:         string
						optional?:    int
						withDefault?: *"foo" | "bar"
					},
					{// 0.3
						init:         string
						optional?:    int
						withDefault?: *"foo" | "bar" | "baz"
					},
				]
			},
		]
	}
}
