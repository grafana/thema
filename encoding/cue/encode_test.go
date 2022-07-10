package cue

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/grafana/thema"
	"github.com/grafana/thema/exemplars"
)

var ctx = cuecontext.New()
var lib = thema.NewLibrary(ctx)

func TestDoSimpleGenLineage(t *testing.T) {
	cuestr := `
foo: string
bar: int
`
	v := ctx.CompileString(cuestr)

	_, err := NewLineage(v, "somelineage", "testpkg")
	if err != nil {
		t.Fatal(err)
	}
	// fmt.Println(string(fmtn(f)))
}

func TestSimpleAppendLineage(t *testing.T) {
	lin, _ := exemplars.NarrowingLineage(lib)

	cuestr := `
properbool: bool
foo?: int
`
	v := ctx.CompileString(cuestr)

	_, err := Append(lin, v)
	if err != nil {
		t.Fatal(err)
	}
	// fmt.Println(string(fmtn(f)))
}
