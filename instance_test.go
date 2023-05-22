package thema

import (
	"cuelang.org/go/cue"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"

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

		lin, lerr := bindTxtarLineage(tc, rt)
		require.NoError(tc, lerr)

		for from := lin.First(); from != nil; from = from.Successor() {
			for _, example := range from.Examples() {
				for to := lin.First(); to != nil; to = to.Successor() {
					tinst, lacunas := example.Translate(to.Version())
					require.NotNil(t, tinst)

					result := tinst.Underlying()
					require.True(t, result.Exists())
					require.NoError(t, result.Err())

					writeGolden(tc, to.Version(), example, result, lacunas)
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

// TestInstance_LookupPathOnTranslatedInstance is a regression test
// specifically written for https://github.com/grafana/thema/issues/155.
//
// Caused because [Instance.Translate] results were always non-concrete,
// when the result evaluates to something concrete.
//
// So, this test checks that [cue.Value.LookupPath] behaves as expected
// when used over an [Instance.Translate] result.
func TestInstance_LookupPathOnTranslatedInstance(t *testing.T) {
	ctx := cuecontext.New()
	rt := NewRuntime(ctx)

	// Initialize lineage for testing
	linstr := `name: "simple"
schemas: [
	{
		version: [0, 0]
		schema:
		{
			title: string
		},
	},
	{
		version: [0, 1]
		schema:
		{
			title: string
			header?: string
		},
	},
]`
	linval := rt.Context().CompileString(linstr)
	lin, err := BindLineage(linval, rt)
	require.NoError(t, err)

	// Initialize cue.Value
	expected := "foo"
	val := ctx.CompileString(fmt.Sprintf(`{"title": "%s"}`, expected))

	// Validate cue.Value
	inst := lin.ValidateAny(val)
	require.Equal(t, SV(0, 0), inst.Schema().Version())

	got, err := inst.Underlying().LookupPath(cue.ParsePath("title")).String()
	require.NoError(t, err)
	require.Equal(t, expected, got)

	// Translate cue.Value (no lacunas)
	tinst, _ := inst.Translate(SV(0, 1))
	require.Equal(t, SV(0, 0), inst.Schema().Version())

	got, err = tinst.Underlying().LookupPath(cue.ParsePath("title")).String()
	require.NoError(t, err)
	require.Equal(t, expected, got)
}
