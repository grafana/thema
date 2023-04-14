package cue

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/grafana/thema"
	"github.com/grafana/thema/exemplars"
)

var ctx = cuecontext.New()
var rt = thema.NewRuntime(ctx)

func TestDoSimpleGenLineage(t *testing.T) {
	cuestr := `
foo: string
bar: int
`
	v := ctx.CompileString(cuestr)

	f, err := NewLineage(v, "somelineage", "testpkg")
	if err != nil {
		t.Fatal(err)
	}
	_ = f
	// fmt.Println(string(astutil.FmtNode(f)))
}

func TestSimpleAppendLineage(t *testing.T) {
	lin, _ := exemplars.NarrowingLineage(rt)

	cuestr := `
properbool: bool
foo?: int
`
	v := ctx.CompileString(cuestr)

	f, err := Append(lin, v)
	if err != nil {
		t.Fatal(err)
	}
	_ = f
	// fmt.Println(string(astutil.FmtNodeP(f)))
}
