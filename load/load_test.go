package load

import (
	"embed"
	"io/fs"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/encoding/json"
	cuejson "cuelang.org/go/pkg/encoding/json"

	"github.com/grafana/thema"
)

//go:embed testdata
var testdataFS embed.FS

func TestInstanceLoadHelper(t *testing.T) {
	ctx := cuecontext.New()

	tfs, err := fs.Sub(testdataFS, "testdata/testmod")
	if err != nil {
		t.Fatal(err)
	}

	binst, err := InstancesWithThema(tfs, ".")
	if err != nil {
		t.Fatal(err)
	}

	val := ctx.BuildInstance(binst)
	if val.Err() != nil {
		t.Fatal(val.Err())
	}

	lin, err := thema.BindLineage(val.LookupPath(cue.ParsePath("lin")), thema.NewLibrary(ctx), thema.SkipBuggyChecks())
	if err != nil {
		t.Fatal(err)
	}

	expr, _ := json.Extract("input", []byte(`{
		"firstfield": "foo"
	}`))

	cv := ctx.BuildExpr(expr)
	sch1, err := lin.Schema(thema.SV(0, 0))
	if err != nil {
		t.Fatal("Could not get existing schema:", err)
	}

	_, err = sch1.Validate(cv)
	if err != nil {
		t.Fatal("validation failed:", err)
	}

	inst := lin.ValidateAny(cv)
	if inst == nil {
		t.Fatal("No schema validated the inst; should have validated against [0, 0]")
	}

	to := thema.SV(1, 0)
	tinst, _ := inst.Translate(to)
	if tinst.Schema().Version() != to {
		t.Logf("Expected output schema version %v, got %v", to, tinst.Schema().Version())
		t.Fail()
	}

	_, err = cuejson.Marshal(tinst.UnwrapCUE())
	if err != nil {
		t.Fatalf("Failed to marshal translation output to JSON with err: \n\t%s", err)
	}

	expr, _ = json.Extract("input", []byte(`{
		"firstfield": "foo",
		"secondfield": -1
	}`))
	wantval := ctx.BuildExpr(expr)
	if !wantval.Equals(tinst.UnwrapCUE()) {
		t.Fatalf("Did not receive expected value after translation:\nWANT: %s\nGOT: %s", wantval, inst.UnwrapCUE())
	}
}

// func printFS(f fs.FS) {
// 	fs.WalkDir(f, ".", (func(path string, d fs.DirEntry, err error) error {
// 		if err != nil {
// 			return err
// 		}
// 		fmt.Println(path)
// 		return nil
// 	}))
// }

// func printOverlay(o map[string]load.Source) {
// 	for p := range o {
// 		fmt.Println(p)
// 	}
// }
