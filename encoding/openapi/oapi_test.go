package openapi

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/format"
	"github.com/google/go-cmp/cmp"
	"github.com/grafana/thema"
	"github.com/grafana/thema/exemplars"
	cuetxtar "github.com/grafana/thema/internal/txtartest"
)

// TODO make a testscript-based golden file approach to this that captures all exemplar encoding
func TestGenerateSchema(t *testing.T) {
	rt := thema.NewRuntime(cuecontext.New())
	lin, err := exemplars.ExpandLineage(rt)
	if err != nil {
		t.Fatal(err)
	}

	f, err := GenerateSchema(lin.First(), nil)
	if err != nil {
		t.Fatal(errors.Details(err, nil))
	}

	b, err := format.Node(f)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != want {
		t.Fatal(cmp.Diff(string(b), want))
	}
}

func TestGenerate(t *testing.T) {
	test := cuetxtar.CueTest{
		Root: "./testdata",
		Name: "generate",
	}

	test.Run(t, func(t *cuetxtar.Test) {
		lin := t.BindLineage(nil)

		f, err := GenerateSchema(lin.Latest(), nil)
		if err != nil {
			// FIXME we should probably accrue errors
			t.WriteErrors(errors.Promote(err, ""))
		} else {
			f.Filename = "base.json"
			t.WriteFile(f)
		}
	})
}

var want = `"openapi": "3.0.0"
"info": {
	"title":   "expand"
	"version": "0.0"
}
"paths": {}
"components": {
	"schemas": {
		"expand": {
			"type": "object"
			"required": ["init"]
			"properties": {
				"init": {
					"type": "string"
				}
			}
		}
	}
}
`
