package gocode

import (
	"testing"

	copenapi "cuelang.org/go/encoding/openapi"
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
			"embed":       "struct embeddings and inlined fields not rendered properly",
			"map_pointer": "group doesn't render maps",
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
		{
			name: "depointerized",
			cfg: &TypeConfigOpenAPI{
				NoOptionalPointers: true,
			},
		},
		{
			name: "godeclincomments",
			cfg: &TypeConfigOpenAPI{
				UseGoDeclInComments: true,
			},
		},
		{
			name: "expandref",
			cfg: &TypeConfigOpenAPI{
				Config: &openapi.Config{
					Config: &copenapi.Config{
						ExpandReferences: true,
					},
				},
			},
		},
	}

	test.Run(t, func(t *cuetxtar.LineageTest) {
		lin := t.BindLineage(nil)

		cuetxtar.ForEachSchema(t, lin, func(t *cuetxtar.LineageTest, sch thema.Schema) {
			for _, tc := range vars {
				itc := tc
				t.Run(itc.name, func(gt *testing.T) {
					t.WriteFileOrErrBytes(itc.name + ".go")(GenerateTypesOpenAPI(sch, itc.cfg))
				})
			}
		})
	})
}
