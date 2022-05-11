package jsonschema

import (
	"fmt"
	"testing"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/pkg/encoding/json"
	"github.com/xeipuuv/gojsonschema"
)

func TestJSONSchemaRewrite(t *testing.T) {
	sl := gojsonschema.NewSchemaLoader()
	sl.Validate = true
	sl.Draft = gojsonschema.Draft4
	err := sl.AddSchemas(gojsonschema.NewStringLoader(complexWant))
	if err != nil {
		t.Fatal(err)
	}

	exp, err := json.Unmarshal([]byte(complexIn))
	if err != nil {
		t.Fatal(err)
	}
	want, err := json.Unmarshal([]byte(complexWant))
	if err != nil {
		t.Fatal(err)
	}

	wantf, err := format.Node(want)
	if err != nil {
		t.Fatal(err)
	}
	_ = wantf

	mod := oapiToJSchema2(exp)

	// modf, err := format.Node(mod)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// t.Log(cmp.Diff(string(wantf), string(modf)))
	j, err := json.Marshal(cuecontext.New().BuildFile(&ast.File{
		Decls: []ast.Decl{mod.(ast.Expr)},
	}))
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(json.Indent([]byte(j), "", "  "))
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

var complexWant = `
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
                "type": "integer"
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
                "type": "integer"
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
                        "type": "string"
                      },
                      "sms": {
                        "type": [
                          "string",
                          "null"
                        ]
                      },
                      "properties": {
                        "type": "object",
                        "additionalProperties": {
                          "type": "string"
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
      "$schema": "http://json-schema.org/draft-04/schema#",
      "properties": {
        "playground": {
          "type": "object",
          "$schema": "http://json-schema.org/draft-04/schema#",
          "required": [
            "feeling",
            "child"
          ],
          "properties": {
            "feeling": {
              "type": "string"
            },
            "child": {
              "type": "object",
              "required": [
                "name",
                "age"
              ],
              "properties": {
                "name": {
                  "type": "string"
                },
                "age": {
                  "type": "integer"
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
                  "enum": [
                    "bucket",
                    "shovel"
                  ],
                  "description": "Toy type"
                }
              }
            }
          }
        }
      }
    }
  ],
  "$schema": "http://json-schema.org/draft-04/schema#"
}
`
