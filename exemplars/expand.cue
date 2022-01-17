package exemplars

import "github.com/grafana/thema"

expand: {
    description: "A few schema in a single sequence, illustrating some simple expansions permitted by backwards compatibility rules."
    l: thema.#Lineage & {
        seqs: [
            {
                schemas: [
                    close({
                        init: string
                    }),
                    close({
                        init: string
                        optional?: int
                    }),
                    close({
                        init: string
                        optional?: int
                        withDefault?: *"foo" | "bar"
                    }),
                    close({
                        init: string
                        optional?: int
                        withDefault?: *"foo" | "bar" | "baz"
                    })
                ]
            }
        ]
    }
}