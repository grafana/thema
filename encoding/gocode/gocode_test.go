package gocode

import (
	"testing"

	"github.com/grafana/thema"
	"github.com/grafana/thema/encoding/openapi"
	cuetxtar "github.com/grafana/thema/internal/txtartest"
)

func TestGenerate(t *testing.T) {
	test := cuetxtar.LineageSuite{
		Root:             "./testdata",
		Name:             "generate",
		IncludeExemplars: true,
		ToDo: map[string]string{
			"TestGenerate/dashboard/0.0/expandrefs": "unexpected problem with converting unification",
		},
	}

	vars := []struct {
		name string
		cfg  *TypeConfigOpenAPI
	}{
		{
			name: "nil",
			cfg:  nil,
		},
		{
			name: "group",
			cfg: &TypeConfigOpenAPI{
				Config: &openapi.Config{Group: true},
			},
		},
	}

	test.Run(t, func(t *cuetxtar.LineageTest) {
		lin := t.BindLineage(nil)

		cuetxtar.ForEachSchema(t, lin, func(t *cuetxtar.LineageTest, sch thema.Schema) {
			for _, tc := range vars {
				t.Run(tc.name, func(gt *testing.T) {
					t.WriteFileOrErrBytes(tc.name + ".go")(GenerateTypesOpenAPI(sch, tc.cfg))
				})
			}
		})
	})
}
