package astutil

import (
	"fmt"
	"strconv"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/token"
)

// Structish comprises AST types that contain a list of declarations that may be fields.
type Structish interface {
	*ast.File | *ast.StructLit
}

func FindSeqs(n ast.Node) *ast.ListLit {
	var ret *ast.ListLit
	ast.Walk(n, func(n ast.Node) bool {
		if ret != nil {
			return false
		}
		if isFieldWithLabel(n, "seqs") {
			if x, ok := n.(*ast.Field).Value.(*ast.ListLit); ok {
				ret = x
				return false
			}
		}
		return true
	}, nil)
	return ret
}

func LatestSchemaList(n ast.Node) (*ast.ListLit, error) {
	seqlist := FindSeqs(n)
	if seqlist == nil {
		return nil, fmt.Errorf("could not find seqs list in input")
	}
	return ListForField(seqlist.Elts[len(seqlist.Elts)-1], "schemas")
}

func ListForField(n ast.Node, label string) (*ast.ListLit, error) {
	seqs, err := GetFieldByLabel(n, label)
	if err != nil {
		return nil, err
	}
	seqlist, is := seqs.Value.(*ast.ListLit)
	if !is {
		return nil, fmt.Errorf("expected %q field to be an ast.ListLit, got %T", label, seqs)
	}

	return seqlist, nil
}

// GetFieldByLabel returns the ast.Field with a given label from a struct-ish input.
func GetFieldByLabel(n ast.Node, label string) (*ast.Field, error) {
	var d []ast.Decl
	switch x := n.(type) {
	case *ast.File:
		d = x.Decls
	case *ast.StructLit:
		d = x.Elts
	default:
		return nil, fmt.Errorf("not an *ast.File or *ast.StructLit")
	}

	for _, el := range d {
		if isFieldWithLabel(el, label) {
			return el.(*ast.Field), nil
		}
	}

	return nil, fmt.Errorf("no field with label %q", label)
}

func strEq(lit *ast.BasicLit, str string) bool {
	if lit.Kind != token.STRING {
		return false
	}
	ls, _ := strconv.Unquote(lit.Value)
	return str == ls || str == lit.Value
}

func identStrEq(id *ast.Ident, str string) bool {
	if str == id.Name {
		return true
	}
	ls, _ := strconv.Unquote(id.Name)
	return str == ls
}

func isFieldWithLabel(n ast.Node, label string) bool {
	if x, is := n.(*ast.Field); is {
		if l, is := x.Label.(*ast.BasicLit); is {
			return strEq(l, label)
		}
		if l, is := x.Label.(*ast.Ident); is {
			return identStrEq(l, label)
		}
	}
	return false
}
