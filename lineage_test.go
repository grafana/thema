package thema

import (
	"bytes"
	"fmt"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/build"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/grafana/thema/internal/txtartest/vanilla"
)

var themaInst *build.Instance

func init() {
	bi := load.Instances(nil, &load.Config{
		Package: "thema",
	})
	themaInst = bi[0]
}

func compileStringWithThema(ctx *cue.Context, src string) cue.Value {
	b := new(bytes.Buffer)
	b.WriteString(src)

	bi := load.Instances([]string{"-"}, &load.Config{
		Context: themaInst.Context(),
		Stdin:   b,
	})

	val := ctx.BuildInstance(bi[0])
	return val
}

func TestBindLineage(t *testing.T) {
	test := vanilla.TxTarTest{
		Root: "./testdata/lineage",
		Name: "bind",
	}

	ctx := cuecontext.New()
	rt := NewRuntime(ctx)

	test.Run(t, func(tc *vanilla.Test) {
		v := ctx.BuildInstance(tc.Instance())
		lin, err := BindLineage(v, rt)
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

		fmt.Fprintf(tc, "%+v\n", err)
	})
}
