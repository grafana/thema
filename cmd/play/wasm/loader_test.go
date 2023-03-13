package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var inputLineage = `
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
`

func TestValidate(t *testing.T) {
	res, err := handle(validateAny, inputLineage, "", `{"firstfield":"1"}`)
	assert.NoError(t, err)
	_ = res

	res, err = handle(validateAny, inputLineage, "", `{"firstfield":1}`)
	assert.Error(t, err)
	_ = res
}

func TestTranslateToLatest(t *testing.T) {
	res, err := handle(translateToLatest, inputLineage, "", `{"firstfield":"1"}`)
	assert.NoError(t, err)
	assert.True(t, len(res) > 0)

	res, err = handle(translateToLatest, inputLineage, "", `{"secondfield":"1"}`)
	assert.Error(t, err)
	_ = res
}
