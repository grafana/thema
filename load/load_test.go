package load

import (
	"embed"
	"io/fs"
	"testing"

	"cuelang.org/go/cue/cuecontext"
)

//go:embed testdata
var testdataFS embed.FS

func TestInstanceLoadHelper(t *testing.T) {
	ctx := cuecontext.New()

	tfs, err := fs.Sub(testdataFS, "testdata/testmod")
	if err != nil {
		t.Fatal(err)
	}

	insts, err := InstancesWithThema(nil, tfs)
	if err != nil {
		t.Fatal(err)
	}

	val := ctx.BuildInstance(insts[0])
	if val.Err() != nil {
		t.Fatal(val.Err())
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
