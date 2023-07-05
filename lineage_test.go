package thema

import (
	"bytes"
	"fmt"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/build"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/load"
	"github.com/grafana/thema/internal/txtartest/vanilla"
)

func TestBindLineage(t *testing.T) {
	test := vanilla.TxTarTest{
		Root: "./testdata/lineage",
		Name: "bind",
		ToDo: map[string]string{
			"lineage/defaultchange": "Thema compat analyzer fails to classify changes to default values as breaking",
			"lineage/optional":      "Optional fields do not satisfy struct.MinFields(), causing #Lineage constraints to fail",
		},
	}

	ctx := cuecontext.New()
	rt := NewRuntime(ctx)

	test.Run(t, func(tc *vanilla.Test) {
		lin, err := bindTxtarLineage(tc, rt, "lineagePath")
		if testing.Short() && tc.HasTag("slow") {
			t.Skip("case is tagged #slow, skipping for -short")
		}

		if err != nil {
			tc.ValidateErrorOrFail(fmt.Errorf("error binding lineage: %+v", err.Error()))
			return
		}
		schemaselem := cue.Str("schemas")
		sspath := cue.MakePath(schemaselem)
		slen, err := lin.Underlying().LookupPath(sspath).Len().Int64()
		if err != nil {
			tc.Fatal("error getting schemas len", err)
		}
		fmt.Fprintf(tc, "Schema count: %v\n", slen)
		fmt.Fprintf(tc, "Schema versions: %s\n", lin.allVersions())

		var llen int64
		if slen > 1 {
			lenseselem := cue.Str("lenses")
			slpath := cue.MakePath(lenseselem)
			llen, err = lin.Underlying().LookupPath(slpath).Len().Int64()
			if err != nil {
				tc.Fatal("error getting lenses len", err)
			}
		}
		fmt.Fprintf(tc, "Lenses count: %v\n", llen)
	})
}

func TestInvalidLineages(t *testing.T) {
	test := vanilla.TxTarTest{
		Root: "./testdata/invalidlineage",
		Name: "bindfail",
		ToDo: map[string]string{
			"invalidlineage/joindef":                "no invariant checker written to disallow definitions from joinSchema",
			"invalidlineage/onlydef":                "Lineage schema non-emptiness constraints are temporarily suspended while migrating grafana to flattened lineage structure",
			"invalidlineage/compat/change-default":  "Thema compat analyzer fails to classify changes to default values as breaking",
			"invalidlineage/compat/remove-required": "Required field removal is not detected as breaking changes",
			"invalidlineage/compat/remove-optional": "Optional field removal is not detected as breaking changes",
		},
	}

	ctx := cuecontext.New()
	rt := NewRuntime(ctx)

	test.Run(t, func(tc *vanilla.Test) {
		_, err := bindTxtarLineage(tc, rt, "lineagePath")
		if testing.Short() && tc.HasTag("slow") {
			tc.Skip("case is tagged #slow, skipping for -short")
		}

		if err == nil {
			tc.Fatal("expected error from known-invalid lineage")
		}
		// TODO more verbose error output, should include CUE line-level analysis
		tc.WriteErrors(errors.Promote(err, ""))
	})
}

func TestIsAppendOnly(t *testing.T) {
	test := vanilla.TxTarTest{
		Root: "./testdata/isappendonly/valid",
		Name: "isappendonly",
		ToDo: map[string]string{
			"isappendonly/valid/withconstraints": "Subsume doesn't support constraints using built-in validators",
			"isappendonly/valid/disjunction":     "Subsume requires the Final() option to consider two complex disjunctions as equal but this creates false negatives",
			"isappendonly/valid/maps":            "Subsume requires the Final() option to consider two maps as equal but this creates false negatives",
		},
	}

	ctx := cuecontext.New()
	rt := NewRuntime(ctx)

	test.Run(t, func(tc *vanilla.Test) {
		if testing.Short() && tc.HasTag("slow") {
			t.Skip("case is tagged #slow, skipping for -short")
		}

		lin1, err := bindTxtarLineage(tc, rt, "firstLin")
		if err != nil {
			tc.Fatalf("error binding first lineage: %+v", err)
		}

		lin2, err := bindTxtarLineage(tc, rt, "secondLin")
		if err != nil {
			tc.Fatalf("error binding second lineage: %+v", err)
		}

		err = IsAppendOnly(lin1, lin2)
		if err != nil {
			tc.Fatalf("IsAppendOnly returned an error: %+v", err)
		}
	})
}

func TestIsAppendOnlyFail(t *testing.T) {
	test := vanilla.TxTarTest{
		Root: "./testdata/isappendonly/invalid",
		Name: "isappendonly-fail",
	}

	ctx := cuecontext.New()
	rt := NewRuntime(ctx)

	test.Run(t, func(tc *vanilla.Test) {
		if testing.Short() && tc.HasTag("slow") {
			t.Skip("case is tagged #slow, skipping for -short")
		}

		lin1, err := bindTxtarLineage(tc, rt, "firstLin")
		if err != nil {
			tc.Fatalf("error binding first lineage: %+v", err)
		}

		lin2, err := bindTxtarLineage(tc, rt, "secondLin")
		if err != nil {
			tc.Fatalf("error binding second lineage: %+v", err)
		}

		err = IsAppendOnly(lin1, lin2)
		if err == nil {
			tc.Fatalf("expected error from known invalid updates")
		}

		// TODO more verbose error output, should include CUE line-level analysis
		tc.WriteErrors(errors.Promote(err, "IsAppendOnly fail"))
	})
}

func bindTxtarLineage(t *vanilla.Test, rt *Runtime, path string) (Lineage, error) {
	if rt == nil {
		rt = NewRuntime(cuecontext.New())
	}
	ctx := rt.Context()

	t.Helper()
	inst := t.Instance()
	val := ctx.BuildInstance(inst)
	if p, ok := t.Value(path); ok {
		pp := cue.ParsePath(p)
		if len(pp.Selectors()) == 0 {
			t.Fatalf("%q is not a valid value for the #%s key", p, path)
		}
		val = val.LookupPath(pp)
		if !val.Exists() {
			t.Fatalf("path %q specified in #%s does not exist in input cue instance", p, path)
		}
	}

	return BindLineage(val, rt)
}

var benchBindstr = `
name: "trivial-two"
schemas: [{
    version: [0, 0]
    schema: {
        firstfield: string
    }
},
{
    version: [0, 1]
    schema: {
        firstfield: string
        secondfield?: int32
    }
}]

lenses: [{
    from: [0, 1]
    to: [0, 0]
    input: _
    result: {
        firstfield: input.firstfield
    }
}]
`

func BenchmarkUnifyLineage(b *testing.B) {
	bi := getCaseWithImport()
	val := cuecontext.New().BuildInstance(getCaseWithImport())
	if val.Err() != nil {
		b.Fatal(val.Err())
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cuecontext.New().BuildInstance(bi)
	}
}

func getCaseWithImport() *build.Instance {
	themaInst := load.Instances(nil, &load.Config{
		Package: "thema",
	})[0]

	buf := new(bytes.Buffer)
	buf.WriteString(`import "github.com/grafana/thema"

thema.#Lineage
`)
	buf.WriteString(benchBindstr)

	bi := load.Instances([]string{"-"}, &load.Config{
		Context: themaInst.Context(),
		Stdin:   buf,
	})
	return bi[0]
}

// BenchmarkBindLineage benchmarks binding a lineage in Go, with and without
// explicitly unifying the input lineage with thema.#Lineage in the CUE source.
//
// Keeping these separate lets us see the difference between the performance cost
// of just the pure, native CUE logic, vs. the cost of the Go code that wraps it.
func BenchmarkBindLineage(b *testing.B) {
	b.Run("PreUnified", func(b *testing.B) {
		ctx := cuecontext.New()
		rt := NewRuntime(ctx)
		linv := ctx.BuildInstance(getCaseWithImport())

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := BindLineage(linv, rt)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("NotUnified", func(b *testing.B) {
		ctx := cuecontext.New()
		rt := NewRuntime(ctx)
		linv := ctx.CompileString(benchBindstr)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := BindLineage(linv, rt)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
