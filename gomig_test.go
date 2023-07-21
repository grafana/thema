package thema

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/require"
)

func TestGoMigrations(t *testing.T) {
	multivlin := `
name: "basic-multiversion"

schemas: [{
    version: [0, 0]
    schema: {
        init: string
    }
    examples: {
        simple: {
            init: "some string"
        }
    }
},
{
    version: [0, 1]
    schema: {
        init:      string
        optional?: int32
    }
    examples: {
        withoutOptional: {
            init: "some string"
        }
        withOptional: {
            init: "some string"
            optional: 32
        }
    }
},
{
    version: [0, 2]
    schema: {
        init:        string
        optional?:   int32
        withDefault?: *"foo" | "bar"
    }
    examples: {
        withoutOptional: {
            init: "some string"
            withDefault: "foo"
        }
        withOptional: {
            init: "some string"
            optional: 32
            withDefault: "bar"
        }
    }
},
{
    version: [0, 3]
    schema: {
        init:        string
        optional?:   int32
        withDefault?: *"foo" | "bar" | "baz"
    }
    examples: {
        withoutOptional: {
            init: "some string"
            withDefault: "baz"
        }
        withOptional: {
            init: "some string"
            optional: 32
            withDefault: "baz"
        }
    }
},
{
    version: [1, 0]
    schema: {
        renamed:     string
        optional?:   int32
        withDefault: "foo" | *"bar" | "baz"
    }
    examples: {
        withoutOptional: {
            renamed: "some string"
            withDefault: "foo"
        }
        withOptional: {
            renamed: "some string"
            optional: 32
            withDefault: "bar"
        }
    }
},
{
    version: [1, 1]
    schema: {
        renamed:     string
        optional?:   int32
        withDefault: "foo" | *"bar" | "baz" | "bing"
    }
    examples: {
        withoutOptional: {
            renamed: "some string"
            withDefault: "bing"
        }
        withOptional: {
            renamed: "some string"
            optional: 32
            withDefault: "bing"
        }
    }
},
{
    version: [2, 0]
    schema: {
        toObj:     {
            init: string
        }
        optional?:   int32
        withDefault: "foo" | *"bar" | "baz" | "bing"
    }
    examples: {
        withoutOptional: {
            toObj: {
                init: "some string"
            }
            withDefault: "bing"
        }
        withOptional: {
            toObj: {
                init: "some string"
            }
            optional: 32
            withDefault: "bing"
        }
    }
}]
`

	lenses := []ImperativeLens{
		{
			To:   SV(0, 0),
			From: SV(0, 1),
			Mapper: func(inst *Instance, to Schema) (*Instance, error) {
				m := tomap(inst)

				tom := map[string]any{
					"init": m["init"],
				}

				return to.Validate(to.Underlying().Context().Encode(tom))
			},
		},
		{
			To:   SV(0, 1),
			From: SV(0, 2),
			Mapper: func(inst *Instance, to Schema) (*Instance, error) {
				m := tomap(inst)

				tom := map[string]any{
					"init": m["init"],
				}
				if v, has := m["optional"]; has {
					tom["optional"] = v
				}

				return to.Validate(to.Underlying().Context().Encode(tom))
			},
		},
		{
			To:   SV(0, 2),
			From: SV(0, 3),
			Mapper: func(inst *Instance, to Schema) (*Instance, error) {
				m := tomap(inst)

				tom := map[string]any{
					"init": m["init"],
				}
				if v, has := m["optional"]; has {
					tom["optional"] = v
				}
				// optional fields with defaults are weeeeird
				if v, has := m["withDefault"]; has {
					tom["withDefault"] = v
				} else {
					tom["withDefault"] = "foo"
				}

				return to.Validate(to.Underlying().Context().Encode(tom))
			},
		},
		{
			To:   SV(0, 3),
			From: SV(1, 0),
			Mapper: func(inst *Instance, to Schema) (*Instance, error) {
				m := tomap(inst)

				tom := map[string]any{
					"init": m["renamed"],
				}
				if v, has := m["optional"]; has {
					tom["optional"] = v
				}
				// optional fields with defaults are weeeeird
				if v, has := m["withDefault"]; has {
					if v == "bar" {
						tom["withDefault"] = "foo"
					} else {
						tom["withDefault"] = v
					}
				} else {
					tom["withDefault"] = "foo"
				}

				return to.Validate(to.Underlying().Context().Encode(tom))
			},
		},
		{
			To:   SV(1, 0),
			From: SV(0, 3),
			Mapper: func(inst *Instance, to Schema) (*Instance, error) {
				m := tomap(inst)

				tom := map[string]any{
					"renamed": m["init"],
				}
				if v, has := m["optional"]; has {
					tom["optional"] = v
				}
				if v, has := m["withDefault"]; has {
					if v == "foo" {
						tom["withDefault"] = "bar"
					} else {
						tom["withDefault"] = v
					}
				} else {
					tom["withDefault"] = "bar"
				}

				return to.Validate(to.Underlying().Context().Encode(tom))
			},
		},
		{
			To:   SV(1, 0),
			From: SV(1, 1),
			Mapper: func(inst *Instance, to Schema) (*Instance, error) {
				m := tomap(inst)

				tom := map[string]any{
					"renamed": m["renamed"],
				}
				if v, has := m["optional"]; has {
					tom["optional"] = v
				}
				if v, has := m["withDefault"]; has {
					tom["withDefault"] = v
				} else {
					tom["withDefault"] = "bar"
				}

				return to.Validate(to.Underlying().Context().Encode(tom))
			},
		},
		{
			To:   SV(1, 1),
			From: SV(2, 0),
			Mapper: func(inst *Instance, to Schema) (*Instance, error) {
				m := tomap(inst)

				tom := map[string]any{
					"renamed": m["toObj"].(map[string]any)["init"],
				}
				if v, has := m["optional"]; has {
					tom["optional"] = v
				}
				if v, has := m["withDefault"]; has {
					tom["withDefault"] = v
				} else {
					tom["withDefault"] = "bar"
				}

				return to.Validate(to.Underlying().Context().Encode(tom))
			},
		},
		{
			To:   SV(2, 0),
			From: SV(1, 1),
			Mapper: func(inst *Instance, to Schema) (*Instance, error) {
				m := tomap(inst)

				tom := map[string]any{
					"toObj": map[string]any{
						"init": m["renamed"],
					},
				}
				if v, has := m["optional"]; has {
					tom["optional"] = v
				}
				if v, has := m["withDefault"]; has {
					tom["withDefault"] = v
				} else {
					tom["withDefault"] = "bar"
				}

				return to.Validate(to.Underlying().Context().Encode(tom))
			},
		},
	}
	ctx := cuecontext.New()
	rt := NewRuntime(ctx)
	linval := rt.Context().CompileString(multivlin)
	_, err := BindLineage(linval, rt, ImperativeLenses(lenses...))
	require.NoError(t, err)
}

func tomap(inst *Instance) map[string]any {
	m := make(map[string]any)
	err := inst.Underlying().Decode(&m)
	if err != nil {
		panic(err)
	}
	return m
}
