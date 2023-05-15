package thema

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"

	"cuelang.org/go/cue"

	"github.com/grafana/thema/internal/txtartest/vanilla"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/require"
)

func TestInstance_Translate(t *testing.T) {
	test := vanilla.TxTarTest{
		Root: "./testdata/lineage",
		Name: "core/instance/translate",
	}

	ctx := cuecontext.New()
	rt := NewRuntime(ctx)

	test.Run(t, func(tc *vanilla.Test) {
		if !tc.HasTag("multiversion") {
			return
		}

		lin, lerr := bindTxtarLineage(tc, rt, "lineagePath")
		require.NoError(tc, lerr)

		for sch := lin.First(); sch != nil; sch = sch.Successor() {
			for _, example := range sch.Examples() {
				for sch := lin.First(); sch != nil; sch = sch.Successor() {
					to := sch.Version()
					tinst, lacunas := example.Translate(to)
					require.NotNil(t, tinst)

					result := tinst.Underlying()
					require.True(t, result.Exists())
					require.NoError(t, result.Err())

					writeGolden(tc, to, example, result, lacunas)
				}
			}
		}
	})
}

func writeGolden(tc *vanilla.Test, to SyntacticVersion, example *Instance, result cue.Value, lacunas TranslationLacunas) {
	tc.Helper()

	fromStr := example.Schema().Version().String()
	toStr := to.String()

	exName := example.name
	tName := strings.Replace(tc.Name(), tc.Name()+"/", "", -1)
	wName := fmt.Sprintf("%s-%s-%s->%s.json", tName, fromStr, exName, toStr)

	w := tc.Writer(wName)

	// From (example)
	marshalAndWrite(tc, w, example.Underlying())
	// To (result)
	marshalAndWrite(tc, w, result)
	// Lacunas
	marshalAndWrite(tc, w, lacunas)
}

func marshalAndWrite(tc *vanilla.Test, w io.Writer, any interface{}) {
	tc.Helper()

	bytes, err := json.Marshal(any)
	require.NoError(tc, err)

	_, err = w.Write(append(bytes, '\n'))
	require.NoError(tc, err)
}
