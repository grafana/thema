package load

import (
	"embed"
	"io/fs"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
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

	inst, err := InstancesWithThema(tfs, ".")
	if err != nil {
		t.Fatal(err)
	}

	val := ctx.BuildInstance(inst)
	if val.Err() != nil {
		t.Fatal(val.Err())
	}

	_, err = thema.BindLineage(val.LookupPath(cue.ParsePath("lin")), thema.NewLibrary(ctx), thema.SkipBuggyChecks())
	if err != nil {
		t.Fatal(err)
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
