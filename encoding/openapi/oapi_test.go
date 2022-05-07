package openapi

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/format"
	"github.com/grafana/thema"
	"github.com/grafana/thema/exemplars"
)

// TODO make a testscript-based golden file approach to this that captures all exemplar encoding
func testGenerateSchema(t *testing.T) {
	lib := thema.NewLibrary(cuecontext.New())
	lin, err := exemplars.ExpandLineage(lib)
	if err != nil {
		t.Fatal(err)
	}

	sch := thema.SchemaP(lin, thema.SV(0, 0))
	f, err := GenerateSchema(sch, nil)
	if err != nil {
		t.Fatal(errors.Details(err, nil))
	}

	b, err := format.Node(f)
	if err != nil {
		t.Fatal(err)
	}

	_ = b
}
