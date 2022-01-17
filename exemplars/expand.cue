package exemplars

import "github.com/grafana/thema"

expand: {
    description: "A few schema in a single sequence, illustrating some simple expansions permitted by backwards compatibility rules."
    l: thema.#Lineage & {
        seqs: [
            {
                schemas: [
                    {
                        init: string
                    },
                    {
                        init: string
                        optional?: int
                    },
                    {
                        init: string
                        optional?: int
                        withDefault?: *"foo" | "bar"
                    },
                    {
                        init: string
                        optional?: int
                        withDefault?: *"foo" | "bar" | "baz"
                    }
                ]
            }
        ]
    }
}