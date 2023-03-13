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
				field1: int
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
	res, err := handle(fn_validate, inputLineage, "0.0", `{"field1":"1"}`)
	assert.NoError(t, err)
	_ = res

	res, err = handle(fn_validate, inputLineage, "0.0", `{"field1":1}`)
	assert.Error(t, err)
	_ = res
}
