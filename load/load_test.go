package load

import (
	"embed"
	"github.com/stretchr/testify/require"
	"io/fs"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/encoding/json"
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

	binst, err := InstanceWithThema(tfs, ".")
	if err != nil {
		t.Fatal(err)
	}

	val := ctx.BuildInstance(binst)
	if val.Err() != nil {
		t.Fatal(val.Err())
	}

	lin, err := thema.BindLineage(val.LookupPath(cue.ParsePath("lin")), thema.NewRuntime(ctx))
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
	require.Error(t, err, "Validate version must fail because secondfield is not present in expr")

	inst := lin.ValidateAny(cv)
	require.Nil(t, inst, "ValidateAny must return nil inst because secondfield is not present in expr")
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
