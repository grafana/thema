package thema

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/require"
)

// TODO: This is a very minimal test added to test a bug fix; please extend!
func TestValidate(t *testing.T) {
	lin := testLin()
	sch, _ := lin.Schema(SyntacticVersion{0, 0})
	ctx := cuecontext.New()
	rt := NewRuntime(ctx)

	tests := map[string]struct {
		input   string
		wantErr string
	}{
		"empty input": {
			`{}`, "",
		},
		"invalid input - field not allowed": {
			`{"cat": "cheetarah"}`, "no Thema handler for CUE error, please file an issue against github.com/grafana/thema\nto improve this error output:\n#Lineage._sortedSchemas.0._#schema.cat: field not allowed\n\n",
		},
		"invalid input - type mismatch": {
			`{"abool": 42}`, "<single@v0.0>._sortedSchemas.0._#schema.abool: validation failed, data is not an instance:\n\tschema expected `42`\n\tbut data contained `bool`\n\t\t7:12\n\t\t1:11\n\t\t1:1\n",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			data := rt.Context().CompileString(test.input)
			_, err := sch.Validate(data)
			if test.wantErr != "" {
				require.EqualError(t, err, test.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
