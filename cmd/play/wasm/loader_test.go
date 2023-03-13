package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var inputLineage = `
seqs: [
	{
		schemas: [
			// v0.0
			{
				field1: string
			},
			// v0.1
			{
				field1: string
			},
		]
	},
]
`

func TestValidate(t *testing.T) {
	_, err := validate(inputLineage, "0.0", `{"field1":"1"}`)
	assert.NoError(t, err)

	_, err = validate(inputLineage, "0.0", `{"field1":1}`)
	assert.Error(t, err)
}
