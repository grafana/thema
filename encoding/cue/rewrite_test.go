package cue

import (
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/build"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
	tastutil "github.com/grafana/thema/internal/astutil"
	"github.com/grafana/thema/internal/txtartest/vanilla"
)

func TestRewriteLegacyLineage(t *testing.T) {
	ctx := cuecontext.New()
	(&vanilla.TxTarTest{
		Root: "./testdata/legacylineage",
		Name: "rewrite-legacy-lineage",
	}).Run(t, func(tc *vanilla.Test) {
		inst := ctx.BuildInstance(tc.Instance())
		val, _ := tc.Value("sub")
		path := cue.ParsePath(val)
		f, err := RewriteLegacyLineage(inst, path)
		if err != nil {
			tc.Fatal(err)
		}

		b, err := format.Node(f, format.Simplify())
		if err != nil {
			t.Fatal(err)
		}
		tc.Write(b)
	})
}

// This test is primarily intended as a demonstration of how to work together
// combining cue.Value with AST rewrites. It's also a bit of a regression guard
// against changes in upstream CUE. It doesn't test any specific, exposed thema
// functionality.
func TestRewriteFundamentals(t *testing.T) {
	ctx := cuecontext.New()

	// 1. test rewriting an instance with one root file
	// 2. test rewriting an instance with two root files
	// 3. test rewriting a subpath within an instance all defined in one file
	// 4. test rewriting a subpath within an instance that's got parts defined in multiple files via unification
	// 5. test rewriting a subpath within an instance that's got parts defined in multiple files via references

	// case 1
	(&vanilla.TxTarTest{
		Root: "./testdata/basicrewrite",
		Name: "oneroot",
	}).Run(t, func(tc *vanilla.Test) {
		// An instance with two input files - len(bi.Files) == 2
		bi := tc.Instance()

		if count := len(bi.Files); count != 2 {
			tc.Fatalf("expected two files in build instance, got %d", count)
		}
		v, src := getMainFile(tc, ctx, bi)

		subv := v.LookupPath(cue.ParsePath("rootfield"))
		if !subv.Exists() {
			tc.Fatal("rootfield path does not exist in instance")
		}

		f, err := tastutil.GetFieldByLabel(src, "rootfield")
		if err != nil {
			tc.Fatal(err)
		}

		if subv.Source() != f {
			tc.Fatal("different ast node pointers for underlying Source of built value and that obtained by navigating ast directly")
		}

		f.Value = ast.NewIdent("float64")

		for _, f := range bi.Files {
			tc.WriteFile(f)
		}
	})

	// (&vanilla.TxTarTest{
	// 	Root: "./testdata/basicrewrite",
	// 	Name: "multiroot",
	// }).Run(t, func(tc *vanilla.Test) {
	// 	// An instance with two input files - len(bi.Files) == 2
	// 	bi := tc.Instance()
	//
	// 	if count := len(bi.Files); count != 2 {
	// 		tc.Fatalf("expected two files in build instance, got %d", count)
	// 	}
	// 	v, src := getMainFile(tc, ctx, bi)
	//
	// }
}

func getMainFile(tc *vanilla.Test, ctx *cue.Context, bi *build.Instance) (cue.Value, *ast.File) {
	tc.Helper()

	var v cue.Value
	var src *ast.File
	var found bool
	for _, f := range bi.Files {
		if filepath.Base(f.Filename) == "in.cue" {
			found = true
			src, v = f, ctx.BuildFile(f)
			break
		}
	}

	if !found {
		tc.Fatal("could not find in.cue file in build instance")
	}

	return v, src
}
