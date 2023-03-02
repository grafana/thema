package thema

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLess(t *testing.T) {
	var tests = []struct {
		v1       SyntacticVersion
		v2       SyntacticVersion
		expected bool
	}{
		{SV(0, 0), SV(0, 0), false},
		{SV(0, 0), SV(0, 1), true},
		{SV(0, 0), SV(1, 0), true},
		{SV(0, 0), SV(1, 1), true},
		{SV(0, 1), SV(0, 0), false},
		{SV(0, 1), SV(1, 0), true},
		{SV(1, 0), SV(0, 0), false},
		{SV(1, 0), SV(0, 1), false},
		{SV(1, 2), SV(0, 1), false},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("comparison between %s and %s", tc.v1, tc.v2), func(t *testing.T) {
			less := tc.v1.Less(tc.v2)
			assert.Equal(t, tc.expected, less)
		})
	}
}
