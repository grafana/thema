export const basic = {
    lineage: `package ship

import "github.com/grafana/thema"
thema.#Lineage
name: "ship"

seqs: [
    {
        schemas: [
        // v0.0
            {
                firstfield: string
            },
        ]
    },
]`,
    input: `{
    "firstfield": "value"
}`
}

export const multi = {
    lineage: `package ship

import "github.com/grafana/thema"
thema.#Lineage
name: "ship"

seqs: [
    {
        schemas: [
            // v0.0
            {
                firstfield: string
            },
            // v0.1
            {
                firstfield: string
                secondfield?: int
            },
        ]
    },
]`,
    input: `{
    "firstfield": "value",
    "secondfield": "100"
}`
}

export const lenses = {
    lineage: `package ship

import "github.com/grafana/thema"
thema.#Lineage
name: "ship"

seqs: [
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
                // Direct mapping of the first field
                firstfield: from.firstfield
                // Just some placeholder int, so we have a valid instance of schema 1.0
                secondfield: -1
            }
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
]`,
    input: `{
    "firstfield": "value"
}`
}
