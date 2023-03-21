package thema

import (
	"bytes"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/build"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/cockroachdb/errors"
	terrors "github.com/grafana/thema/errors"
)

var themaInst *build.Instance

func init() {
	bi := load.Instances(nil, &load.Config{
		Package: "thema",
	})
	themaInst = bi[0]
}

func TestBindLineage(t *testing.T) {
	table := map[string]struct {
		in       string
		path     string
		noimport bool
		err      error
	}{
		"empty": {
			err:      terrors.ErrValueNotALineage,
			noimport: true,
		},
		"badpath": {
			err:  terrors.ErrValueNotExist,
			path: "notexist",
		},
		"defonly": {
			err: terrors.ErrInvalidLineage,
			in: `
thema.#Lineage
`,
		},
		"schemaless": {
			err: terrors.ErrInvalidLineage,
			in: `
thema.#Lineage
name: "something"
`,
		},
		"empty-schemas-array": {
			err: terrors.ErrInvalidLineage,
			in: `
thema.#Lineage
name: "something"
schemas: []
`,
		},
	}

	ctx := cuecontext.New()
	rt := NewRuntime(ctx)

	for name, itt := range table {
		tt := itt
		t.Run(name, func(t *testing.T) {
			valstr := tt.in
			if !tt.noimport {
				valstr = "import \"github.com/grafana/thema\"\n" + tt.in
			}

			val := compileStringWithThema(ctx, valstr)
			if tt.path != "" {
				val = val.LookupPath(cue.ParsePath(tt.path))
			}
			_, err := BindLineage(val, rt)
			if !errors.Is(err, tt.err) {
				t.Fatalf("expected error %q, got %q", tt.err, err)
			}
			t.Logf("%+v\n", err)
		})
	}
}

func compileStringWithThema(ctx *cue.Context, src string) cue.Value {
	b := new(bytes.Buffer)
	b.WriteString(src)

	bi := load.Instances([]string{"-"}, &load.Config{
		Context: themaInst.Context(),
		Stdin:   b,
	})

	val := ctx.BuildInstance(bi[0])
	return val
}
