package thema

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	cjson "cuelang.org/go/encoding/json"
	"cuelang.org/go/pkg/strings"
	"github.com/grafana/thema/internal/txtartest/vanilla"
	"github.com/stretchr/testify/require"
)

// Validation-related test cases look for `*.data.json` files within
// the txtar archives, describing input data to validate against the lineage.
// The expected results are described in a file matching the input file.
// Example:
// * data file name: `firstfieldAsInt32.data.json`
// * result file name: `firstfieldAsInt32`
func TestValidate(t *testing.T) {
	test := vanilla.TxTarTest{
		Root:    "./testdata/lineage",
		Name:    "validate/TestValidate",
		ThemaFS: CueJointFS,
	}

	ctx := cuecontext.New()
	rt := NewRuntime(ctx)

	test.Run(t, func(tc *vanilla.Test) {
		req := require.New(tc)

		val := ctx.BuildInstance(tc.Instance())
		lineage, err := BindLineage(val, rt)
		req.NoError(err)

		for _, file := range tc.Archive.Files {
			if !strings.HasSuffix(file.Name, ".data.json") {
				continue
			}

			data, err := decodeData(rt, string(file.Data))
			req.NoError(err)

			_, err = lineage.Latest().Validate(data)
			req.Error(err, "The data shouldn't be valid for the schema")

			outputFileName := filepath.Base(strings.TrimSuffix(file.Name, ".data.json"))

			_, err = tc.Writer(outputFileName).Write([]byte(err.Error()))
			req.NoError(err)
		}
	})
}

func decodeData(rt *Runtime, inputJSON string) (cue.Value, error) {
	if inputJSON == "" {
		return cue.Value{}, errors.New("test error - data is missing")
	}

	ctx := rt.Underlying().Context()
	expr, err := cjson.Extract("test", []byte(inputJSON))
	if err != nil {
		return cue.Value{}, fmt.Errorf("test error - failed to decode input data: %w", err)
	}

	return ctx.BuildExpr(expr), nil
}
