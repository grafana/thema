package typescript

import (
	"github.com/stretchr/testify/require"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/grafana/thema"
	"github.com/grafana/thema/internal/txtartest/bindlin"
	"github.com/grafana/thema/internal/txtartest/vanilla"
)

func TestGenerate(t *testing.T) {
	test := vanilla.TxTarTest{
		Root:    "../../testdata/lineage",
		Name:    "encoding/typescript/TestGenerate",
		ThemaFS: thema.CueJointFS,
		Skip: map[string]string{
			"lineage/refexscalar": "bounds constraints are not supported as they lack a direct typescript equivalent",
			"lineage/refscalar":   "bounds constraints are not supported as they lack a direct typescript equivalent",
		},
	}

	ctx := cuecontext.New()
	rt := thema.NewRuntime(ctx)

	table := []struct {
		name string
		cfg  *TypeConfig
	}{
		{
			name: "nilcfg",
			cfg:  nil,
		},
	}

	for _, tb := range table {
		t.Run(tb.name, func(t *testing.T) {
			testcpy := test
			testcpy.Name += "/" + tb.name
			testcpy.Run(t, func(tc *vanilla.Test) {
				if testing.Short() && tc.HasTag("slow") {
					t.Skip("case is tagged #slow, skipping for -short")
				}
				lin, err := bindlin.BindTxtarLineage(tc, rt)
				if err != nil {
					tc.Fatal(err)
				}

				for sch := lin.First(); sch != nil; sch = sch.Successor() {
					f, err := GenerateTypes(sch, tb.cfg)
					if err != nil {
						tc.Fatal(err)
					}
					_, err = tc.Write([]byte(f.String())) //nolint:gosec,errcheck
					require.NoError(t, err)
				}
			})
		})
	}
}
