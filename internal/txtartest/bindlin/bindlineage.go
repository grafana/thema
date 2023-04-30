package bindlin

import (
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/grafana/thema"
	"github.com/grafana/thema/internal/txtartest/vanilla"
)

// BindTxtarLineage attempts to bind a lineage from the CUE package instance at
// the txtar fs root. By default, it will assume the entire instance is intended
// to be a lineage. However, if a #lineagePath key exists with a value, that
// path will be used instead.
//
// This is here so that the thema root package can import the txtar framework
// without creating an import cycle
func BindTxtarLineage(t *vanilla.Test, rt *thema.Runtime) (thema.Lineage, error) {
	if rt == nil {
		rt = thema.NewRuntime(cuecontext.New())
	}
	ctx := rt.Context()

	t.Helper()
	inst := t.Instance()
	val := ctx.BuildInstance(inst)
	if p, ok := t.Value("lineagePath"); ok {
		t.Log(p)
		pp := cue.ParsePath(p)
		if len(pp.Selectors()) == 0 {
			t.Fatalf("%q is not a valid value for the #lineagePath key", p)
		}
		val = val.LookupPath(pp)
		if !val.Exists() {
			t.Fatalf("path %q specified in #lineagePath does not exist in input cue instance", p)
		}
	}

	return thema.BindLineage(val, rt)
}
