# Playground examples

## Validate against a certain version

action: `Validate`

version: `0.1`

json: 
```json
{
    "firstfield": "hello",
    "secondfield": 10
}
```

lineage:
```go
package ship

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
]
```

## Validate against any version in the lineage

action: `Validate Any`

json:
```json
{
    "firstfield": "hello",
    "secondfield": 15
}
```

lineage:
```go
package ship

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
]
```

## Translate to the latest version, output with lacunas

action: `Translate to latest`

json:
```json
{
    "firstfield": "hello"
}
```

lineage:
```go
package ship

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
]
```

## Format

action: `Format`

lineage:
```go
package ship

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
]
```
