package thema_test

import (
	"errors"
	"fmt"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/grafana/thema"
	"github.com/grafana/thema/vmux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ctx = cuecontext.New()
var rt = thema.NewRuntime(ctx)

func TestBasicValidate(t *testing.T) {
	linstrs := []struct {
		name   string
		linstr string
	}{{
		name: "int32",
		linstr: `name: "simple"
schemas: [
	{
		version: [0, 0]
		schema:
		{
			title: int32
			header?: string
		},
	},
]`,
	}, {
		name: "int64",
		linstr: `name: "simple"
schemas: [
	{
		version: [0, 0]
		schema:
		{
			title: int64
			header?: string
		},
	},
]`},
		{
			name: "float32",
			linstr: `name: "simple"
schemas: [
	{
		version: [0, 0]
		schema:
		{
			title: float32
			header?: string
		},
	},
]`,
		}, {
			name: "float64",
			linstr: `name: "simple"
schemas: [
	{
		version: [0, 0]
		schema:
		{
			title: float64
			header?: string
		},
	},
]`}, {
			name: "custom range",
			linstr: `name: "simple"
schemas: [
	{
		version: [0, 0]
		schema:
		{
			title: int32 & > 10 & < 20
			header?: string
		},
	},
]`}, {
			name: "disjunction",
			linstr: `name: "simple"
schemas: [
	{
		version: [0, 0]
		schema:
		{
			title: float64 | null
			header?: string
		},
	},
]`,
		}}

	for _, tc := range linstrs {
		t.Run(tc.name, func(t *testing.T) {
			linval := rt.Context().CompileString(tc.linstr)
			lin, err := thema.BindLineage(linval, rt)
			require.NoError(t, err)

			data, err := decodeData(`{"title": "null"}`)
			if err != nil {
				require.NoError(t, err)
			}

			latest := lin.Latest()

			_, err = latest.Validate(data)
			if err != nil {
				assert.NoError(t, err)
			}
		})
	}
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
