package openapi

import (
	"testing"

	"cuelang.org/go/cue/errors"
	"github.com/grafana/thema"
	cuetxtar "github.com/grafana/thema/internal/txtartest"
)

func TestGenerate(t *testing.T) {
	test := cuetxtar.CueTest{
		Root: "./testdata",
		Name: "generate",
	}

	test.Run(t, func(t *cuetxtar.Test) {
		lin := t.BindLineage(nil)

		cuetxtar.ForEachSchema(t, lin, func(t *cuetxtar.Test, sch thema.Schema) {
			f, err := GenerateSchema(sch, nil)
			if err != nil {
				// FIXME we should probably accrue errors
				t.WriteErrors(errors.Promote(err, ""))
			} else {
				f.Filename = "base.json"
				t.WriteNamedFile(f)
			}
		})
	})
}
