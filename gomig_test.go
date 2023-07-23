package thema

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

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
        TOwithOptional: {
            init: "some string"
        }
        TOwithoutOptional: {
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
},
{
    version: [0, 2]
    schema: {
        init:        string
        optional?:   int32
        withDefault?: *"foo" | "bar"
    }
},
{
    version: [0, 3]
    schema: {
        init:        string
        optional?:   int32
        withDefault?: *"foo" | "bar" | "baz"
    }
},
{
    version: [1, 0]
    schema: {
        renamed:     string
        optional?:   int32
        withDefault: "foo" | *"bar" | "baz"
    }
},
{
    version: [1, 1]
    schema: {
        renamed:     string
        optional?:   int32
        withDefault: "foo" | *"bar" | "baz" | "bing"
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
		TOsimple: {
            toObj: {
                init: "some string"
            }
            withDefault: "bar"
		}
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

	correctLenses := []ImperativeLens{
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
				if v, has := m["withDefault"]; has && v != "bing" {
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
	lin, err := BindLineage(linval, rt, ImperativeLenses(correctLenses...))
	require.NoError(t, err)

	transtest := func(t *testing.T, start, end Schema) {
		for name, ex := range start.Examples() {
			if strings.HasPrefix(name, "TO") {
				continue
			}
			tex, tname := ex, name
			t.Run(tname, func(t *testing.T) {
				t.Log(start.Version(), end.Version())
				tinst, lacunas, err := tex.Translate(end.Version())
				require.NoError(t, err)
				assert.Nil(t, lacunas, "pure go migrations cannot emit lacunas")

				b, err := tinst.Underlying().MarshalJSON()
				require.NoError(t, err)

				eb, err := end.Examples()["TO"+tname].Underlying().MarshalJSON()
				require.NoError(t, err)
				assert.Equal(t, b, eb)
			})
		}
	}

	t.Run("forward", func(t *testing.T) {
		transtest(t, lin.First(), lin.Latest())
	})

	t.Run("reverse", func(t *testing.T) {
		transtest(t, lin.Latest(), lin.First())
	})

	t.Run("bind-invalid", func(t *testing.T) {
		_, err = BindLineage(linval, rt, ImperativeLenses(correctLenses[1:]...))
		assert.Error(t, err, "expected error when missing a reverse Go migration")
		_, err = BindLineage(linval, rt, ImperativeLenses(correctLenses[:1]...))
		assert.Error(t, err, "expected error when missing a forward Go migration")

		_, err = BindLineage(linval, rt, ImperativeLenses(append(correctLenses, ImperativeLens{
			To:     SV(2, 0),
			From:   SV(2, 1),
			Mapper: func(inst *Instance, to Schema) (*Instance, error) { return nil, nil },
		})...))
		assert.Error(t, err, "expected error when adding Go migration pointing to nonexistent version")

		_, err = BindLineage(linval, rt, ImperativeLenses(append(correctLenses, ImperativeLens{
			To:     SV(2, 1),
			From:   SV(2, 0),
			Mapper: func(inst *Instance, to Schema) (*Instance, error) { return nil, nil },
		})...))
		assert.Error(t, err, "expected error when adding Go migration pointing to nonexistent version")

		_, err = BindLineage(linval, rt, ImperativeLenses(append(correctLenses, ImperativeLens{
			To:     SV(2, 0),
			From:   SV(1, 1),
			Mapper: func(inst *Instance, to Schema) (*Instance, error) { return nil, nil },
		})...))
		assert.Error(t, err, "expected error when adding duplicate Go migration")

		_, err = BindLineage(linval, rt, ImperativeLenses(append(correctLenses, ImperativeLens{
			Mapper: func(inst *Instance, to Schema) (*Instance, error) { return nil, nil },
		})...))
		assert.Error(t, err, "expected error when providing a Go migration with same to and from")

		_, err = BindLineage(linval, rt, ImperativeLenses(append(correctLenses, ImperativeLens{
			To:     SV(2, 0),
			From:   SV(1, 0),
			Mapper: func(inst *Instance, to Schema) (*Instance, error) { return nil, nil },
		})...))
		assert.Error(t, err, "expected error when providing Go migration with wrong successor")

		_, err = BindLineage(linval, rt, ImperativeLenses(append(correctLenses, ImperativeLens{
			To:     SV(1, 0),
			From:   SV(2, 0),
			Mapper: func(inst *Instance, to Schema) (*Instance, error) { return nil, nil },
		})...))
		assert.Error(t, err, "expected error when providing Go migration with wrong predecessor")

		_, err = BindLineage(linval, rt, ImperativeLenses(append(correctLenses, ImperativeLens{
			To:     SV(1, 1),
			From:   SV(1, 0),
			Mapper: func(inst *Instance, to Schema) (*Instance, error) { return nil, nil },
		})...))
		assert.Error(t, err, "expected error when providing Go migration for minor version upgrade")
	})
}

func tomap(inst *Instance) map[string]any {
	m := make(map[string]any)
	err := inst.Underlying().Decode(&m)
	if err != nil {
		panic(err)
	}
	return m
}
