package main

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestValidateAny(t *testing.T) {
	lin, err := loadLineage(inputLineage)
	require.NoError(t, err)

	datval, err := decodeData(`{"firstfield":"1"}`)
	require.NoError(t, err)
	res, err := validateAny(lin, datval)
	assert.NoError(t, err)
	_ = res

	datval, err = decodeData(`{"firstfield":1}`)
	require.NoError(t, err)
	res, err = validateAny(lin, datval)
	assert.Error(t, err)
	_ = res
}

func TestTranslateToLatest(t *testing.T) {
	lin, err := loadLineage(inputLineage)
	require.NoError(t, err)

	datval, err := decodeData(`{"firstfield":"1"}`)
	require.NoError(t, err)
	res, err := translateToLatest(lin, datval)
	assert.NoError(t, err)
	assert.True(t, len(res) > 0)

	datval, err = decodeData(`{"secondfield":"1"}`)
	require.NoError(t, err)
	res, err = translateToLatest(lin, datval)
	assert.Error(t, err)
	_ = res
}

func TestGetLineageVersions(t *testing.T) {
	lin, err := loadLineage(inputLineage)
	require.NoError(t, err)

	res, err := lineageVersions(lin)
	assert.NoError(t, err)
	expected, _ := json.Marshal([]string{"0.0", "1.0"})
	assert.Equal(t, string(expected), res)
}
