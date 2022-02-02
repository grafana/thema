package kernel

import (
	"fmt"
	"strings"
	"testing"

	"cuelang.org/go/cue"
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

var tf00 = func() interface{} {
	return &type00{}
}
var tf10 = func() interface{} {
	return &type10{}
}

func TestInputKernelInputs(t *testing.T) {
	ctx := cuecontext.New()
	lib := thema.NewLibrary(ctx)

	lin, err := exemplars.RenameLineage(lib)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("missing-lineage-panic", func(t *testing.T) {
		cfg := InputKernelConfig{
			Loader:      jsondl,
			TypeFactory: tf10,
		}
		defer func() {
			recover()
		}()

		NewInputKernel(cfg)
		t.Fatal("should panic")
	})

	t.Run("missing-loader-panic", func(t *testing.T) {
		cfg := InputKernelConfig{
			TypeFactory: tf10,
			Lineage:     lin,
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
			TypeFactory: func() interface{} {
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
			Loader:      jsondl,
			TypeFactory: tf00,
			Lineage:     lin,
			To:          thema.SV(1, 0),
		}
		_, err := NewInputKernel(cfg)
		if err == nil {
			t.Fatal("should fail when type incompatible with schema is emitted from type factory")
		}

		cfg.To = thema.SV(0, 0)
		cfg.TypeFactory = tf10
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
		Loader:      jsondl,
		TypeFactory: tf00,
		Lineage:     lin,
		To:          thema.SV(0, 0),
	})
	if err != nil {
		t.Fatal(err)
	}

	k10, err := NewInputKernel(InputKernelConfig{
		Loader:      jsondl,
		TypeFactory: tf10,
		Lineage:     lin,
		To:          thema.SV(1, 0),
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
		// Commented cases fail due to what appear to be the same underlying bug
		// with having the schema declared within lists
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

func TestAssignable(t *testing.T) {
	ctx := cuecontext.New()

	var blah string
	tt := map[string]struct {
		// The raw CUE that will be checked for assignability to a Go type
		cue string
		// The Go type the raw CUE will be checked for assignability to
		T       interface{}
		invalid bool
	}{
		"nonpointer": {
			T: struct{}{},
			cue: `typ: _
			`,
			invalid: true,
		},
		"nonstruct": {
			T: &blah,
			cue: `typ: _
			`,
			invalid: true,
		},
		"simpleMatch": {
			T: &struct {
				Foo string `json:"foo"`
			}{},
			cue: `typ: {
				foo: string
			}
			`,
		},
		"stringEnum": {
			T: &struct {
				Foo string `json:"foo"`
			}{},
			cue: `typ: {
				foo: "foo" | "bar"
			}
			`,
		},
		"multiEnum": {
			T: &struct {
				Foo string `json:"foo"`
			}{},
			cue: `typ: {
				foo: "foo" | 14
			}
			`,
			invalid: true,
		},
		"extraSchField": {
			T: &struct {
				Foo string `json:"foo"`
			}{},
			cue: `typ: {
				foo: "foo" | "bar"
				bar: int
			}
			`,
			invalid: true,
		},
		"extraGoField": {
			T: &struct {
				Foo string `json:"foo"`
				Bar int    `json:"bar"`
			}{},
			cue: `typ: {
				foo: "foo" | "bar"
			}
			`,
			invalid: true,
		},
		"optionalField": {
			T: &struct {
				Foo string `json:"foo"`
			}{},
			cue: `typ: {
				foo?: string
			}
			`,
		},
		"optionalFieldOmit": {
			T: &struct {
				Foo string `json:"foo,omitempty"`
			}{},
			cue: `typ: {
				foo?: string
			}
			`,
		},
		"wrongType": {
			T: &struct {
				Foo string `json:"foo"`
			}{},
			cue: `typ: {
				foo: int
			}
			`,
			invalid: true,
		},
		"convertMap": {
			T: &struct {
				Amap map[string]bool `json:"amap"`
			}{},
			cue: `typ: {
				amap: [string]: bool
			}
			`,
		},
		"nestedStruct": {
			T: &struct {
				Foo   string `json:"foo"`
				Nest1 struct {
					Inner string `json:"inner"`
					Nest2 struct {
						DoubleInner string `json:"doubleinner"`
					} `json:"nest2"`
				} `json:"nest1"`
			}{},
			cue: `typ: {
				foo: string
				nest1: {
					inner: string
					nest2: {
						doubleinner: string
					}
				}
			}
			`,
		},
		"simpleSlice": {
			T: &struct {
				Slice []string `json:"slice"`
			}{},
			cue: `typ: {
				slice: [...string]
			}
			`,
		},
		"closedList": {
			T: &struct {
				Slice []string `json:"slice"`
			}{},
			cue: `typ: {
				slice: []
			}
			`,
			invalid: true,
		},
		"closedStringList": {
			T: &struct {
				Slice []string `json:"slice"`
			}{},
			cue: `typ: {
				slice: [string]
			}
			`,
			invalid: true,
		},
		"arrayEmpty": {
			T: &struct {
				Arr [0]string `json:"arr"`
			}{},
			cue: `typ: {
				arr: []
			}
			`,
		},
		"arrayClosedList": {
			T: &struct {
				Arr [2]string `json:"arr"`
			}{},
			cue: `typ: {
				arr: [string, string]
			}
			`,
		},
		"simpleSliceMistype": {
			T: &struct {
				Slice []string `json:"slice"`
			}{},
			cue: `typ: {
				slice: [...int]
			}
			`,
			invalid: true,
		},
		"listMinLen": {
			T: &struct {
				Slice []string `json:"slice"`
			}{},
			cue: `typ: {
				slice: [string, ...string]
			}
			`,
		},
		"listMinLenMultitype": {
			T: &struct {
				Slice []string `json:"slice"`
			}{},
			cue: `typ: {
				slice: [int, ...string]
			}
			`,
			invalid: true,
		},
		"structInList": {
			T: &struct {
				Slice []struct {
					Listfield string `json:"listfield"`
				} `json:"slice"`
			}{},
			cue: `typ: {
				slice: [...{
					listfield: "foo" | "bar"
				}]
			}
			`,
		},
		"listInStruct": {
			T: &struct {
				Foo  string `json:"foo"`
				Nest struct {
					Slice []struct {
						Listfield string `json:"listfield"`
					} `json:"slice"`
				} `json:"nest"`
			}{},
			cue: `typ: {
				foo: string
				nest: {
					slice: [...{
						listfield: string
					}]
				}
			}
			`,
		},
		"listInList": {
			T: &struct {
				Foo         string     `json:"foo"`
				DoubleSlice [][]string `json:"doubleslice"`
			}{},
			cue: `typ: {
				foo: string
				doubleslice: [...[...string]]
			}
			`,
		},
	}

	for name, tst := range tt {
		t.Run(name, func(t *testing.T) {
			f := func(def bool) func(t *testing.T) {
				return func(t *testing.T) {
					path, cuestr := "typ", tst.cue
					if def {
						cuestr = strings.Replace(cuestr, "typ", "#typ", 1)
						path = "#typ"
					}
					// fmt.Println(name, ".", path)
					sch := ctx.CompileString(cuestr).LookupPath(cue.ParsePath(path))

					err := assignable(sch, tst.T)
					if tst.invalid {
						if err == nil {
							t.Fatal("expected unassignable err")
						}
						t.Log(err)
						return
					}

					if err != nil {
						t.Fatal(err)
					}
				}
			}
			t.Run("normal", f(false))
			t.Run("definition", f(true))
		})
	}
}
