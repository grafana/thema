package thema

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

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
			cue: `typ: {}
			`,
		},
		"nonstruct": {
			T: blah,
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
		"integerArch": {
			T: &struct {
				UintField uint `json:"uintField"`
				IntField  int  `json:"intField"`
			}{},
			cue: fmt.Sprintf(`typ: {
				uintField: uint%v
				intField: int%v
			}
			`, strconv.IntSize, strconv.IntSize),
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
