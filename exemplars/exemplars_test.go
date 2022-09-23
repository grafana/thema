package exemplars

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/grafana/thema"
)

var allctx = cuecontext.New()
var dirinst cue.Value
var allrt *thema.Runtime

func init() {
	dirinst = buildAll(allctx)
	allrt = thema.NewRuntime(allctx)
}

func TestExemplarValidity(t *testing.T) {
	iter, err := dirinst.Fields(cue.Definitions(false))
	if err != nil {
		t.Fatal(err)
	}

	for iter.Next() {
		v := iter.Value().LookupPath(cue.ParsePath("l"))
		name, _ := v.LookupPath(cue.ParsePath("name")).String()
		t.Run("Bind-"+name, func(t *testing.T) {
			t.Parallel()
			_, err := thema.BindLineage(v, allrt, nameOpts[name]...)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func BenchmarkBindLineage(b *testing.B) {
	for name, o := range nameOpts {
		b.Run(name, func(b *testing.B) {
			lib := thema.NewRuntime(cuecontext.New())
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				lineageForExemplar(name, lib, o...)
			}
		})
	}
}
