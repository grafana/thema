package exemplars

import "github.com/grafana/scuemata"

expand: {
    description: "A few schema in a single sequence, illustrating some simple expansions permitted by backwards compatibility (subsumption) rules."
    l: scuemata.#Lineage & {
        Seqs: [
            {
                schemas: [
                    {
                        init: string
                    },
                    {
                        init: string
                        withDefault: *"foo" | "bar"
                    },
                    {
                        init: string
                        withDefault: *"foo" | "bar"
                        optional?: int
                    },
                    {
                        init: string
                        withDefault: *"foo" | "bar" | "baz"
                        optional?: int
                    }
                ]
            }
        ]
    }
}