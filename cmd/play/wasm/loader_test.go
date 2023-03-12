package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestValidate(t *testing.T) {
	_, err := validate("/Users/tb/Desktop/ship.cue", "0.0", `{"field1":"1"}`)
	assert.NoError(t, err)

	_, err = validate("/Users/tb/Desktop/ship.cue", "0.0", `{"field1":1}`)
	assert.Error(t, err)
}
