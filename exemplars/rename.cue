package exemplars

import "github.com/grafana/thema"

rename: {
	description: "A field is renamed - a breaking change, necessitating a new sequence."
	l:           thema.#Lineage & {
		seqs: [
			{
				schemas: [
					{
						before:    string
						unchanged: string
					},
				]
			},
			{
				schemas: [
					{
						after:     string
						unchanged: string
					},
				]

				lens: forward: {
					to:         seqs[1].schemas[0]
					from:       seqs[0].schemas[0]
					translated: to & rel
					rel: {
						after:     from.before
						unchanged: from.unchanged
					}
					lacunas: []
				}
				lens: reverse: {
					to:         seqs[0].schemas[0]
					from:       seqs[1].schemas[0]
					translated: to & rel
					rel: {
						before:    from.after
						unchanged: from.unchanged
					}
					lacunas: []
				}
			},
		]
	}
}
