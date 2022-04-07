package kernel

import (
	"fmt"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/grafana/thema"
	"github.com/grafana/thema/exemplars"
)

var jsondl = NewJSONDecoder("test")

type type00 struct {
	Before    string `json:"before"`
	Unchanged string `json:"unchanged"`
}
type type10 struct {
	After     string `json:"after"`
	Unchanged string `json:"unchanged"`
}

var tf00 = &type00{}
var tf10 = &type10{}

func TestInputKernelInputs(t *testing.T) {
	ctx := cuecontext.New()
	lib := thema.NewLibrary(ctx)

	lin, err := exemplars.RenameLineage(lib)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("missing-lineage-panic", func(t *testing.T) {
		cfg := InputKernelConfig{
			Loader: jsondl,
			Typ:    tf10,
		}
		defer func() {
			recover()
		}()

		NewInputKernel(cfg)
		t.Fatal("should panic")
	})

	t.Run("missing-loader-panic", func(t *testing.T) {
		cfg := InputKernelConfig{
			Typ:     tf10,
			Lineage: lin,
		}
		defer func() {
			recover()
		}()

		NewInputKernel(cfg)
		t.Fatal("should panic")
	})

	t.Run("missing-tf-panic", func(t *testing.T) {
		cfg := InputKernelConfig{
			Loader:  jsondl,
			Lineage: lin,
		}
		defer func() {
			recover()
		}()

		NewInputKernel(cfg)
		t.Fatal("should panic")
	})

	t.Run("err-non-pointer-tf", func(t *testing.T) {
		cfg := InputKernelConfig{
			Loader: jsondl,
			Typ: func() interface{} {
				return type00{}
			},
			Lineage: lin,
			To:      thema.SV(0, 0),
		}
		_, err := NewInputKernel(cfg)
		if err == nil {
			t.Fatal("should fail when non-pointer type is emitted from type factory")
		}
	})

	t.Run("invalid-type", func(t *testing.T) {
		cfg := InputKernelConfig{
			Loader:  jsondl,
			Typ:     tf00,
			Lineage: lin,
			To:      thema.SV(1, 0),
		}
		_, err := NewInputKernel(cfg)
		if err == nil {
			t.Fatal("should fail when type incompatible with schema is emitted from type factory")
		}

		cfg.To = thema.SV(0, 0)
		cfg.Typ = tf10
		_, err = NewInputKernel(cfg)
		if err == nil {
			t.Fatal("should fail when type incompatible with schema is emitted from type factory")
		}
	})
}

func TestInputKernelConverge(t *testing.T) {
	ctx := cuecontext.New()
	lib := thema.NewLibrary(ctx)

	lin, err := exemplars.RenameLineage(lib)
	if err != nil {
		t.Fatal(err)
	}

	k00, err := NewInputKernel(InputKernelConfig{
		Loader:  jsondl,
		Typ:     tf00,
		Lineage: lin,
		To:      thema.SV(0, 0),
	})
	if err != nil {
		t.Fatal(err)
	}

	k10, err := NewInputKernel(InputKernelConfig{
		Loader:  jsondl,
		Typ:     tf10,
		Lineage: lin,
		To:      thema.SV(1, 0),
	})
	if err != nil {
		t.Fatal(err)
	}

	invalidtt := map[string]struct {
		jsonstr string
	}{
		"malformed-json": {
			jsonstr: `
			{
				foo": "bar"
			}
			`,
		},
		"invalid": {
			jsonstr: `
			{
				"foo": "bar"
			}
			`,
		},
	}

	validtt := map[string]struct {
		jsonstr  string
		from     thema.SyntacticVersion
		output00 type00
		output10 type10
	}{
		"00good": {
			jsonstr: `
			{
				"before": "renamedstr",
				"unchanged": "unchanged str val"
			}
			`,
			from: thema.SV(0, 0),
			output00: type00{
				Before:    "renamedstr",
				Unchanged: "unchanged str val",
			},
			output10: type10{
				After:     "renamedstr",
				Unchanged: "unchanged str val",
			},
		},
		"10good": {
			jsonstr: `
			{
				"after": "renamedstr",
				"unchanged": "unchanged str val"
			}
			`,
			from: thema.SV(1, 0),
			output00: type00{
				Before:    "renamedstr",
				Unchanged: "unchanged str val",
			},
			output10: type10{
				After:     "renamedstr",
				Unchanged: "unchanged str val",
			},
		},
		"00empty": {
			jsonstr: `
			{
				"before": "",
				"unchanged": ""
			}
			`,
			from: thema.SV(0, 0),
			output00: type00{
				Before:    "",
				Unchanged: "",
			},
			output10: type10{
				After:     "",
				Unchanged: "",
			},
		},
		"10empty": {
			jsonstr: `
			{
				"after": "",
				"unchanged": ""
			}
			`,
			from: thema.SV(1, 0),
			output00: type00{
				Before:    "",
				Unchanged: "",
			},
			output10: type10{
				After:     "",
				Unchanged: "",
			},
		},
	}

	testfunc := func(t *testing.T, k InputKernel) {
		to := k.Config().To
		to00 := to == thema.SV(0, 0)
		t.Run(fmt.Sprintf("to-%v", to), func(t *testing.T) {
			for testname, tab := range invalidtt {
				t.Run(testname, func(t *testing.T) {
					_, _, err := k.Converge([]byte(tab.jsonstr))
					if err == nil {
						t.Fatal("should have failed to converge, but no err received")
					}
				})
			}
			for testname, tab := range validtt {
				t.Run(testname, func(t *testing.T) {
					if k.Config().To.Less(tab.from) {
						t.Skip("reverse translation is not yet supported")
					}
					out, _, err := k.Converge([]byte(tab.jsonstr))
					if err != nil {
						t.Fatal(err)
					}

					if to00 {
						oval := *out.(*type00)
						if tab.output00 != oval {
							t.Fatalf("output targeting 0.0 was not as expected:\n\tWNT:%+v\n\tGOT:%+v\n", tab.output00, oval)
						}
					} else if tab.output10 != *out.(*type10) {
						t.Fatalf("output targeting 1.0 was not as expected:\n\tWNT:%+v\n\tGOT:%+v\n", tab.output10, *out.(*type10))
					}
				})
			}
		})
	}

	testfunc(t, k00)
	testfunc(t, k10)
}
