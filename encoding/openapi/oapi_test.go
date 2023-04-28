package openapi

import (
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/encoding/openapi"
	"github.com/grafana/thema"
	"github.com/grafana/thema/internal/txtartest/vanilla"
)

func TestGenerateVanilla(t *testing.T) {
	test := vanilla.TxTarTest{
		Root: "../../testdata/lineage",
		Name: "encoding/openapi/TestGenerateVanilla",
		ToDo: map[string]string{
			"lineage/defaultchange": "default backcompat invariants not working properly yet",
		},
	}

	ctx := cuecontext.New()
	rt := thema.NewRuntime(ctx)

	for _, cfg := range []struct {
		name string
		cfg  *Config
	}{
		{
			name: "nilcfg",
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
				Subpath:  cue.ParsePath("someField"),
				RootName: "overriddenName",
			},
		},
	} {
		tcfg := cfg
		t.Run(tcfg.name, func(t *testing.T) {
			// TODO parallelize
			// t.Parallel()
			// parallel causes "unpaired free" panic in cue:
			//
			// 	/usr/local/Cellar/go/1.20.2/libexec/src/runtime/panic.go:884 +0x213
			// cuelang.org/go/internal/core/adt.(*Vertex).freeNode(0x0?, 0x0?)
			//	/Users/sdboyer/ws/go/pkg/mod/github.com/sdboyer/cue@v0.5.0-beta.2.0.20221218111347-341999f48bdb/internal/core/adt/eval.go:1128 +0x138

			testcpy := test
			testcpy.Name += "/" + tcfg.name

			testcpy.Run(t, func(tc *vanilla.Test) {
				if strings.HasPrefix(tcfg.name, "subpath") && !tc.HasTag("subpath") {
					return
				}
				if testing.Short() && tc.HasTag("slow") {
					t.Skip("case is tagged #slow, skipping for -short")
				}

				val := ctx.BuildInstance(tc.Instance())
				if p, ok := tc.Value("lineagePath"); ok {
					pp := cue.ParsePath(p)
					if len(pp.Selectors()) == 0 {
						tc.Fatalf("%q is not a valid value for the #lineagePath key", p)
					}
					val = val.LookupPath(pp)
					if !val.Exists() {
						tc.Fatalf("path %q specified in #lineagePath does not exist in input cue instance", p)
					}
				}

				lin, err := thema.BindLineage(val, rt)
				if err != nil {
					t.Fatal(err)
				}
				for sch := lin.First(); sch != nil; sch = sch.Successor() {
					f, err := GenerateSchema(sch, tcfg.cfg)
					if err != nil {
						tc.WriteErrors(errors.Promote(err, sch.Version().String()))
					} else {
						f.Filename = sch.Version().String() + ".json"
						tc.WriteFile(f)
					}
				}
			})
		})
	}
}
