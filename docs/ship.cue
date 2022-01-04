package thema_example

import "github.com/grafana/thema"

lin: thema.#Lineage
lin: Name: "Ship"
lin: seqs: [
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
            from: seqs[0].schemas[0]
            to: seqs[1].schemas[0]
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
                    message: "-1 used as a placeholder value - replace with a real value before persisting!"
                    type: thema.#LacunaTypes.Placeholder
                }
            ]
            translated: to & rel
        }
        lens: reverse: {
            from: seqs[1].schemas[0]
            to: seqs[0].schemas[0]
            rel: {
                // Map the first field back
                firstfield: from.firstfield
            }
            translated: to & rel
        }
    }
]
