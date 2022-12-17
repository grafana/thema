package openapi

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/encoding/openapi"
	"github.com/grafana/thema"
	cuetxtar "github.com/grafana/thema/internal/txtartest"
)

func TestGenerate(t *testing.T) {
	test := cuetxtar.CueTest{
		Root:             "./testdata",
		Name:             "generate",
		IncludeExemplars: true,
		ToDo: map[string]string{
			"TestGenerate/dashboard/0.0/expandrefs": "unexpected problem with converting unification",
		},
	}

	vars := []struct {
		name string
		cfg  *Config
	}{
		{
			name: "nil",
			cfg:  nil,
		},
		{
			name: "group",
			cfg: &Config{
				Group: true,
			},
		},
		{
			name: "expandrefs",
			cfg: &Config{
				Config: &openapi.Config{
					ExpandReferences: true,
				},
			},
		},
		{
			name: "selfcontained",
			cfg: &Config{
				Config: &openapi.Config{
					SelfContained: true,
				},
			},
		},
		{
			name: "subpath",
			cfg: &Config{
				Subpath: cue.ParsePath("someField"),
			},
		},
		{
			name: "subpathroot",
			cfg: &Config{
				Subpath: cue.ParsePath("someField"),
				RootName: "overriddenName",
			},
		},
	}

	test.Run(t, func(t *cuetxtar.Test) {
		lin := t.BindLineage(nil)

		cuetxtar.ForEachSchema(t, lin, func(t *cuetxtar.Test, sch thema.Schema) {
			for _, tc := range vars {
				itest := tc
				t.T.Run(itest.name, func(gt *testing.T) {
					t.WriteFileOrErr(itest.name)(GenerateSchema(sch, itest.cfg))
				})
			}
		})
	})
}
