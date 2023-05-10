package thema_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/grafana/thema"
	"github.com/grafana/thema/internal/txtartest/bindlin"
	"github.com/grafana/thema/internal/txtartest/vanilla"
	"github.com/grafana/thema/vmux"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/require"
)

func TestInstance_Translate(t *testing.T) {
	test := vanilla.TxTarTest{
		Root: "./testdata/lineage",
		Name: "core/instance/translate",
	}

	ctx := cuecontext.New()
	rt := thema.NewRuntime(ctx)

	test.Run(t, func(tc *vanilla.Test) {
		if !tc.HasTag("multiversion") {
			return
		}

		tName := strings.Replace(tc.Name(), t.Name()+"/", "", -1)

		lin, lerr := bindlin.BindTxtarLineage(tc, rt)
		require.NoError(tc, lerr)

		for sch := lin.First(); sch != nil; sch = sch.Successor() {
			for name, ex := range sch.Examples() {
				for sch := lin.First(); sch != nil; sch = sch.Successor() {
					// TODO: Validate lacunas
					to := sch.Version()
					tinst, _ := ex.Translate(to)
					require.NotNil(t, tinst)

					raw := tinst.Underlying()
					require.True(t, raw.Exists())
					require.NoError(t, raw.Err())

					codec := vmux.NewJSONCodec("test")
					rawBytes, err := codec.Encode(raw)
					require.NoError(t, err)

					wName := fmt.Sprintf("%s-%s-%s->%s.json", tName, name, ex.Schema().Version().String(), to.String())
					w := tc.Writer(wName)
					_, err = w.Write(rawBytes)
					require.NoError(t, err)
				}
			}
		}

	})
}
