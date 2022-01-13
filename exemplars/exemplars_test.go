package exemplars

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/grafana/thema"
)

var allctx = cuecontext.New()
var dirinst cue.Value
var alllib thema.Library

var nameOpts = map[string][]thema.BindOption{
	"defaultchange": {thema.SkipBuggyChecks()},
	"narrowing":     {thema.SkipBuggyChecks()},
	"rename":        {thema.SkipBuggyChecks()},
	"expand":        {},
	"single":        {},
}

func init() {
	dirinst = buildAll(allctx)
	alllib = thema.NewLibrary(allctx)
}

func TestExemplarValidity(t *testing.T) {
	iter, err := dirinst.Fields(cue.Definitions(false))
	if err != nil {
		t.Fatal(err)
	}

	for iter.Next() {
		lin := iter.Value().LookupPath(cue.ParsePath("l"))
		name, _ := lin.LookupPath(cue.ParsePath("name")).String()
		t.Run("Bind"+name, func(t *testing.T) {
			var o []thema.BindOption
			switch name {
			case "defaultchange", "narrowing", "rename":
				// subsumption in cue v0.4.0 panics in all three of these cases
				o = append(o, thema.SkipBuggyChecks())
			}
			_, err = thema.BindLineage(lin, alllib, o...)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func BenchmarkBindLineage(b *testing.B) {
	for name, o := range nameOpts {
		b.Run(name, func(b *testing.B) {
			lib := thema.NewLibrary(cuecontext.New())
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				lineageForExemplar(name, lib, o...)
			}
		})
	}
}
