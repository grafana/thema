package analyze

import (
	"fmt"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/build"
	"cuelang.org/go/cue/cuecontext"
	"github.com/grafana/thema/internal/txtartest/vanilla"
)

func TestFindInstance(t *testing.T) {
	test := vanilla.TxTarTest{
		Root: "./testdata",
		Name: "rootpath",
	}

	ctx := cuecontext.New()

	buildInst := func(tc *vanilla.Test) cue.Value {
		if sub, ok := tc.Value("subinst"); ok {
			return ctx.BuildInstance(tc.Instances(sub)[0])
		}
		return ctx.BuildInstance(tc.Instance())
	}

	dumpBI := func(tc *vanilla.Test, bi *build.Instance, val cue.Value) {
		fmt.Fprintln(tc, "ID:", bi.ID())

		op, _ := val.Expr()
		fmt.Fprintln(tc, "ROOT OP:", op)

		iter, _ := val.Fields(cue.All())
		fmt.Fprint(tc, "FIELDS: ")
		for iter.Next() {
			fmt.Fprint(tc, iter.Selector(), " ")
		}
		fmt.Fprintln(tc)
	}

	test.Run(t, func(tc *vanilla.Test) {
		val := buildInst(tc)
		bi := FindBuildInstance(val)
		if bi == nil {
			tc.Fatalf("could not find instance")
		} else {
			dumpBI(tc, bi, val)
		}
	})

	test.Run(t, func(tc *vanilla.Test) {
		val := buildInst(tc)
		bi := FindBuildInstance(val)
		if bi == nil {
			tc.Fatalf("could not find instance")
		} else {
			dumpBI(tc, bi, val)
		}
	})

	test = vanilla.TxTarTest{
		Root: "./testdata",
		Name: "subpath",
	}

	test.Run(t, func(tc *vanilla.Test) {
		subp, has := tc.Value("subpath")
		if !has {
			tc.Skip("no subpath txtar tag")
			return
		}

		t.Logf("%q", subp)
		val := buildInst(tc).LookupPath(cue.ParsePath(subp))
		if !val.Exists() {
			tc.Fatalf("subpath %q not found", subp)
		}
		bi := FindBuildInstance(val)
		if bi == nil {
			tc.Fatalf("could not find instance")
		} else {
			dumpBI(tc, bi, val)
		}
	})
}
