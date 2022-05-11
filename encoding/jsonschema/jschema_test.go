package jsonschema

import (
	"testing"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/pkg/encoding/json"
	"github.com/grafana/thema"
	"github.com/grafana/thema/exemplars"
	"github.com/xeipuuv/gojsonschema"
)

var sl = gojsonschema.NewSchemaLoader()
var lib = thema.NewLibrary(cuecontext.New())

func init() {
	sl.Validate = true
	sl.Draft = gojsonschema.Draft4
}

func TestExemplarExportIsValid(t *testing.T) {
	all := exemplars.All(lib)
	for name, lin := range all {
		t.Run(name, func(t *testing.T) {
			for sch := thema.SchemaP(lin, thema.SV(0, 0)); sch != nil; sch = sch.Successor() {
				isch := sch
				t.Run(isch.Version().String(), func(t *testing.T) {
					f, err := GenerateSchema(isch)
					if err != nil {
						t.Fatal(err)
					}

					j, err := json.Marshal(cuecontext.New().BuildFile(f))
					if err != nil {
						t.Fatal(err)
					}
					if err = sl.AddSchemas(gojsonschema.NewStringLoader(j)); err != nil {
						t.Fatal(err)
					}
				})
			}
		})
	}
}

func TestJSONSchemaRewrite(t *testing.T) {
	exp, err := json.Unmarshal([]byte(complexIn))
	if err != nil {
		t.Fatal(err)
	}

	mod, err := oapiToJSchema(exp)
	if err != nil {
		t.Fatal(err)
	}

	j, err := json.Marshal(cuecontext.New().BuildFile(&ast.File{
		Decls: []ast.Decl{mod.(ast.Expr)},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if err = sl.AddSchemas(gojsonschema.NewStringLoader(j)); err != nil {
		t.Fatal(err)
	}
}

var complexIn = `
{
	"allOf": [
    {
      "anyOf": [
        {
          "type": "object",
          "properties": {
            "cats": {
              "type": "array",
              "items": {
                "type": "integer",
                "example": [
                  1
                ]
              }
            }
          }
        },
        {
          "type": "object",
          "properties": {
            "dogs": {
              "type": "array",
              "items": {
                "type": "integer",
                "example": [
                  1
                ]
              }
            }
          }
        },
        {
          "type": "object",
          "properties": {
            "bring_cats": {
              "type": "array",
              "items": {
                "allOf": [
                  {
                    "type": "object",
                    "properties": {
                      "email": {
                        "type": "string",
                        "example": "cats@email.com"
                      },
                      "sms": {
                        "type": "string",
                        "nullable": true,
                        "example": "+12345678"
                      },
                      "properties": {
                        "type": "object",
                        "additionalProperties": {
                          "type": "string"
                        },
                        "example": {
                          "name": "Wookie"
                        }
                      }
                    }
                  },
                  {
                    "required": [
                      "email"
                    ]
                  }
                ]
              }
            }
          }
        }
      ]
    },
    {
      "type": "object",
      "properties": {
        "playground": {
          "type": "object",
          "required": [
            "feeling",
            "child"
          ],
          "properties": {
            "feeling": {
              "type": "string",
              "example": "Good feeling"
            },
            "child": {
              "type": "object",
              "required": [
                "name",
                "age"
              ],
              "properties": {
                "name": {
                  "type": "string",
                  "example": "Steven"
                },
                "age": {
                  "type": "integer",
                  "example": 5
                }
              }
            },
            "toy": {
              "type": "object",
              "properties": {
                "breaks_easily": {
                  "type": "boolean",
                  "default": false
                },
                "color": {
                  "type": "string",
                  "description": "Color of the toy"
                },
                "type": {
                  "type": "string",
									"enum": ["bucket", "shovel"],
                  "description": "Toy type"
                }
              }
            }
          }
        }
      }
    }
  ]
}
`
