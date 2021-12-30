package example

import "github.com/grafana/thema"

lin: thema.#Lineage
lin: Name: "Ship"
lin: Seqs: [
    {
        schemas: [
            { // 0.0
                firstfield: string
            },
        ]
    },
    {
        schemas: [
            { // 1.0
                firstfield: string
                secondfield: int
            }
        ]

        lens: forward: {
            from: Seqs[0].schemas[0]
            to: Seqs[1].schemas[0]
            rel: {
                firstfield: from.firstfield
                secondfield: -1
            }
            lacunae: [
                thema.#Lacuna & {
                    targetFields: [{
                        path: "secondfield"
                        value: to.secondfield
                    }]
                }
                message: "-1 used as a placeholder value - replace with a real value before persisting!"
                type: thema.#LacunaTypes.Placeholder
            ]
            translated: to & rel
        }
        lens: reverse: {
            from: Seqs[1].schemas[0]
            to: Seqs[0].schemas[0]
            rel: {
                // Map the first field back
                firstfield: from.firstfield
            }
            translated: to & rel
        }
    }
]
