package errors_tests

import (
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"errors"
	"fmt"
	"github.com/grafana/thema"
	"github.com/grafana/thema/vmux"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

var ctx = cuecontext.New()
var rt = thema.NewRuntime(ctx)

func TestThemaErrors(t *testing.T) {
	linstr := `name: "simple"
schemas: [
	{
		version: [0, 0]
		schema:
		{
			title: int64
		},
	},
	{
		version: [0, 1]
		schema:
		{
			title: int64
			header?: string
		},
	},
]`
	linval := rt.Context().CompileString(linstr)
	lin, err := thema.BindLineage(linval, rt)
	require.NoError(t, err)

	data, err := decodeData(`{"title": null}`)
	if err != nil {
		t.Fatal(err)
	}

	res := validateAllVersions(lin, data)
	fmt.Println(strings.Join(res, "\n"))
}

func validateAllVersions(lin thema.Lineage, data cue.Value) (result []string) {
	latest := lin.Latest()

	res := []string{}
	for sch := latest; sch != nil; sch = sch.Predecessor() {
		if _, err := sch.Validate(data); err != nil {
			res = append(res, fmt.Sprintf("validation failed: version %s, %v", sch.Version(), err))
		} else {
			res = append(res, fmt.Sprintf("validation passed: version %s", sch.Version()))
		}
	}

	return res
}

func decodeData(inputJSON string) (cue.Value, error) {
	if inputJSON == "" {
		return cue.Value{}, errors.New("test error - data is missing")
	}

	jd := vmux.NewJSONCodec("test")
	datval, err := jd.Decode(rt.Underlying().Context(), []byte(inputJSON))
	if err != nil {
		return cue.Value{}, fmt.Errorf("test error - failed to decode input data: %w", err)
	}
	return datval, nil
}
