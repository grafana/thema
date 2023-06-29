package gocode

import (
	"fmt"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	copenapi "cuelang.org/go/encoding/openapi"
	"github.com/grafana/thema"
	"github.com/grafana/thema/encoding/openapi"
	"github.com/grafana/thema/internal/txtartest/bindlin"
	"github.com/grafana/thema/internal/txtartest/vanilla"
	"github.com/grafana/thema/internal/util"
)

func TestGenerate(t *testing.T) {
	test := vanilla.TxTarTest{
		Root: "../../testdata/lineage",
		Name: "encoding/gocode/TestGenerate",
		Skip: map[string]string{
			"lineage/unordered-lenses":  "lineage loading must fail, so this test is skipped",
			"lineage/unordered-schemas": "lineage loading must fail, so this test is skipped",
		},
		ToDo: map[string]string{
			"lineage/defaultchange": "default backcompat invariants not working properly yet",
			"lineage/optional":      "Optional fields do not satisfy struct.MinFields(), causing #Lineage constraints to fail",
			"lineage/union":         "Test is abominably slow, cue evaluator is choking up on disjunctions",
		},
	}

	ctx := cuecontext.New()
	rt := thema.NewRuntime(ctx)

	table := []struct {
		name string
		cfg  *TypeConfigOpenAPI
	}{
		{
			name: "nilcfg",
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

	for _, cfg := range table {
		tcfg := cfg
		t.Run(tcfg.name, func(t *testing.T) {
			// TODO parallelize
			testcpy := test
			testcpy.Name += "/" + tcfg.name
			testcpy.Run(t, func(tc *vanilla.Test) {
				if testing.Short() && tc.HasTag("slow") {
					t.Skip("case is tagged #slow, skipping for -short")
				}

				lin, err := bindlin.BindTxtarLineage(tc, rt)
				if err != nil {
					tc.Fatal(err)
				}
				cfg := tcfg.cfg
				if cfg == nil {
					cfg = &TypeConfigOpenAPI{}
				}
				saniname := util.SanitizeLabelString(lin.Name())
				cfg.PackageName = saniname
				for sch := lin.First(); sch != nil; sch = sch.Successor() {
					f, err := GenerateTypesOpenAPI(sch, cfg)
					if err != nil {
						tc.Fatal(err)
					}

					// TODO add support for file name writing more generically
					fmt.Fprintf(tc, "== %s_type_%s_gen.go\n", saniname, sch.Version())
					tc.Write(f) //nolint:gosec,errcheck
				}
			})
		})
	}
}
