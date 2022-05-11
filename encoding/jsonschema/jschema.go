package jsonschema

import (
	"errors"
	"fmt"
	"strconv"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/ast/astutil"
	"cuelang.org/go/cue/token"
	"github.com/grafana/thema"
	"github.com/grafana/thema/encoding/openapi"
)

// GenerateSchema generates a JSON Schema (Draft 4) schema representation of the
// provided Thema schema.
func GenerateSchema(sch thema.Schema) (*ast.File, error) {
	f, err := openapi.GenerateSchema(sch, nil)
	if err != nil {
		return nil, err
	}

	return oapiToJSchema2(f).(*ast.File), nil
}

type schNode struct {
	parent     *schNode
	n          *ast.StructLit
	typ        string
	ensureNull bool
	scanf      scanfunc
}

type scanfunc func(p *schNode, n *ast.StructLit) error

func oapiToJSchema2(f ast.Node) ast.Node {
	err := scan(nil, f)
	if err != nil {
		panic(err)
	}
	return f
}

// Reports if the provided node is an oapi/json schema `"type": <val>` field,
// and if <val> is the given typeName. Always false if multiple types are
// allowed in a list.
func typeIs(n ast.Node, t string) bool {
	if !isFieldWithLabel(n, "type") {
		return false
	}

	switch x := n.(*ast.Field).Value.(type) {
	case *ast.BasicLit:
		return strEq(x, t)
	case *ast.ListLit:
		return false // todo allow multi in one
	default:
		return false
	}

	return false
}

// Reports if the provided node is an oapi/json schema `"type": <val>` field,
// and if the given typeName is present in <val>.
func typeContains(n ast.Node, t string) bool {
	if !isFieldWithLabel(n, "type") {
		return false
	}

	switch x := n.(*ast.Field).Value.(type) {
	case *ast.BasicLit:
		return strEq(x, t)
	case *ast.ListLit:
		return false // todo allow multi in one
	default:
		return false
	}

	return false
}

func isFieldWithLabel(n ast.Node, label string) bool {
	if x, is := n.(*ast.Field); is {
		if l, is := x.Label.(*ast.BasicLit); is {
			return strEq(l, label)
		}
	}
	return false
}

func strEq(lit *ast.BasicLit, str string) bool {
	if lit.Kind != token.STRING {
		return false
	}
	ls, _ := strconv.Unquote(lit.Value)
	return str == ls || str == lit.Value
}

// a schnode represents a single openapi schema node
type schnode struct {
	parent *schnode
	n      ast.Node
	t      string
}

func getFieldWithLabel(n *ast.StructLit, label string) *ast.Field {
	for _, el := range n.Elts {
		if x, is := el.(*ast.Field); is {
			if lit, is := x.Label.(*ast.BasicLit); is && strEq(lit, label) {
				return x
			}
		}
	}

	return nil
}

func getSchemaType(n *ast.StructLit) (string, error) {
	if f := getFieldWithLabel(n, "type"); f != nil {
		if lit, is := f.Value.(*ast.BasicLit); is {
			ls, _ := strconv.Unquote(lit.Value)
			if ls != "" {
				return ls, nil
			}
			return lit.Value, nil
		}
	}
	return "", errNotASchema
}

func isLogicOp(n ast.Node) bool {
	for _, op := range []string{"oneOf", "allOf", "anyOf", "not"} {
		if isFieldWithLabel(n, op) {
			return true
		}
	}
	return false
}

var errNotASchema = errors.New("not a schema node")

func newSchemaNode(parent schemaNode, in ast.Node) (schemaNode, error) {
	n, is := in.(*ast.StructLit)
	if !is {
		return nil, errNotASchema
	}
	inner := &schNode{
		parent: parent.(*schNode),
		n:      n,
	}

	typ, err := getSchemaType(n)
	if err != nil {
		return nil, err
	}
	switch typ {
	case "object":
		inner.scanf = func(p *schNode, n *ast.StructLit) error {
			p.ensureNull = checkNull(n)

			// Recurse down the properties
			if pf := getFieldWithLabel(n, "properties"); pf != nil {
				err = scan(p, pf)
				if err != nil {
					return err
				}
			}

			// And additionalProperties
			if apf := getFieldWithLabel(n, "additionalProperties"); apf != nil {
				return scan(p, apf)
			}

			return nil
		}

	case "array":
		inner.scanf = func(p *schNode, n *ast.StructLit) error {
			p.ensureNull = checkNull(n)

			// Recurse down the items
			if items := getFieldWithLabel(n, "items"); items != nil {
				return scan(p, items)
			}

			return nil
		}
	case "integer", "number", "boolean", "string":
		inner.scanf = func(p *schNode, n *ast.StructLit) error {
			p.ensureNull = checkNull(n)
			return nil
		}
	default:
		return nil, fmt.Errorf("unrecognized schema node type %s", typ)
	}

	return inner, nil

	// Try scanning down to see if we have allOf/oneOf/anyOf/not
	// if isLogicOp(n) {
	// 	inner.scanf = func(p *schNode, n *ast.StructLit) error {
	// 		return scan(p, n.Value)
	// 	}
	// 	return inner, nil
	// }

	// return nil, nil
}

func checkNull(n *ast.StructLit) bool {
	if f := getFieldWithLabel(n, "nullable"); f != nil {
		if x, is := f.Value.(*ast.BasicLit); is {
			return x.Kind == token.TRUE
		}
	}
	return false
}

// func (s *schNode) process() error {
func (s *schNode) process() {
	if err := s.scanf(s, s.n); err != nil {
		panic(err)
	}

	astutil.Apply(s.n, func(c astutil.Cursor) bool {
		switch x := c.Node().(type) {
		case *ast.StructLit:
			return true
		case *ast.Field:
			if l, is := x.Label.(*ast.BasicLit); is {
				var lval string
				lval = l.Value
				if ulv, _ := strconv.Unquote(l.Value); ulv != "" {
					lval = ulv
				}

				switch lval {
				// None of these are allowed in JSON Schema
				case "example", "readOnly", "writeOnly", "discriminator", "nullable", "xml":
					c.Delete()
				case "type":
					if s.ensureNull && !typeContains(x, "null") {
						x.Value = ast.NewList(x.Value, ast.NewString("null"))
					}
				}
			}
		}
		return false
	}, nil)

	// Add the $schema field to root only
	if s.parent == nil {
		s.n.Elts = append(s.n.Elts, &ast.Field{
			Label: ast.NewString("$schema"),
			Value: ast.NewString("http://json-schema.org/draft-04/schema#"),
		})
	}
}

func scan(p *schNode, n ast.Node) error {
	var fatal error
	ast.Walk(n, func(n ast.Node) bool {
		if fatal != nil {
			return false
		}

		sch, err := newSchemaNode(p, n)
		if err != nil {
			if errors.Is(err, errNotASchema) {
				return true
			}

			// Unexpected error, abort walk
			fatal = err
			return false
		}

		if sch != nil {
			sch.process()
		}
		return sch == nil
	}, nil)

	return fatal
}

type schemaNode interface {
	process()
}
