package exemplars

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	"github.com/grafana/thema"
)

var allctx = cuecontext.New()
var dirinst cue.Value
var alllib thema.Library

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
			switch name {
			case "defaultchange", "narrowing", "rename":
				// subsumption in cue v0.4.0 panics in all three of these cases
				t.Skip()
			}
			_, err = thema.BindLineage(lin, alllib)
			if err != nil {
				t.Fatal(errors.Details(err, nil))
			}
		})
	}
}
