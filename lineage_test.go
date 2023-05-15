package thema

import (
	"fmt"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
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
			tc.Fatalf("error binding lineage: %+v", err)
		}

		sspath := cue.MakePath(cue.Hid("_sortedSchemas", "github.com/grafana/thema"))
		slen, err := lin.Underlying().LookupPath(sspath).Len().Int64()
		if err != nil {
			tc.Fatal("error getting schemas len", err)
		}
		fmt.Fprintf(tc, "Schema count: %v\n", slen)
		fmt.Fprintf(tc, "Schema versions: %s\n", lin.allVersions())

		slpath := cue.MakePath(cue.Hid("_sortedLenses", "github.com/grafana/thema"))
		llen, err := lin.Underlying().LookupPath(slpath).Len().Int64()
		if err != nil {
			tc.Fatal("error getting schemas len", err)
		}
		fmt.Fprintf(tc, "Lenses count: %v\n", llen)
	})
}

func TestInvalidLineages(t *testing.T) {
	test := vanilla.TxTarTest{
		Root: "./testdata/invalidlineage",
		Name: "bindfail",
		ToDo: map[string]string{
			"invalidlineage/defaultchange": "Thema compat analyzer fails to classify changes to default values as breaking",
			"invalidlineage/joindef":       "no invariant checker written to disallow definitions from joinSchema",
			"invalidlineage/onlydef":       "Lineage schema non-emptiness constraints are temporarily suspended while migrating grafana to flattened lineage structure",
			"invalidlineage/addremove":     "Required field addition is not detected as breaking changes",
		},
	}

	ctx := cuecontext.New()
	rt := NewRuntime(ctx)

	test.Run(t, func(tc *vanilla.Test) {
		v := ctx.BuildInstance(tc.Instance())
		_, err := BindLineage(v, rt)
		if testing.Short() && tc.HasTag("slow") {
			t.Skip("case is tagged #slow, skipping for -short")
		}

		if err == nil {
			tc.Fatal("expected error from known-invalid lineage")
		}
		// TODO more verbose error output, should include CUE line-level analysis
		tc.WriteErrors(errors.Promote(err, "bind fail"))
	})
}

func TestIsAppendOnly(t *testing.T) {
	test := vanilla.TxTarTest{
		Root: "./testdata/isappendonly/valid",
		Name: "isappendonly",
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
		t.Log(p)
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
