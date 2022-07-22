package astutil

import (
	"fmt"
	"strconv"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/ast/astutil"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/token"
)

// FindSeqs finds the seqs field within what is expected to be a valid lineage ast.Node
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

// LatestSchemaList finds the ListLit for the latest sequence in what is expected
// to be a valid lineage ast.Node
func LatestSchemaList(n ast.Node) (*ast.ListLit, error) {
	seqlist := FindSeqs(n)
	if seqlist == nil {
		return nil, fmt.Errorf("could not find seqs list in input")
	}
	return listForField(seqlist.Elts[len(seqlist.Elts)-1], "schemas")
}

// SchemaListFor finds the ListLit for a particular major version number in what
// is expected to be a valid lineage ast.Node
func SchemaListFor(n ast.Node, majv uint) (*ast.ListLit, error) {
	seqlist := FindSeqs(n)
	if seqlist == nil {
		return nil, fmt.Errorf("could not find seqs list in input")
	}
	if majv >= uint(len(seqlist.Elts)) {
		return nil, fmt.Errorf("major version %v not present in lineage", majv)
	}

	return listForField(seqlist.Elts[majv], "schemas")
}

func listForField(n ast.Node, label string) (*ast.ListLit, error) {
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

// FmtNode exports a node to CUE standard-fmt'd bytes using standard Thema configuration.
func FmtNode(n ast.Node) ([]byte, error) {
	if x, ok := n.(*ast.File); ok {
		err := astutil.Sanitize(x)
		if err != nil {
			return nil, err
		}
	}
	b, err := format.Node(n, format.TabIndent(true), format.Simplify())
	if err != nil {
		return nil, fmt.Errorf("failed to convert input schema to string: %w", err)
	}
	b, err = format.Source(b, format.TabIndent(true), format.Simplify())
	if err != nil {
		return nil, fmt.Errorf("could not reformat to canonical source form: %w", err)
	}
	return b, nil
}

// FmtNodeP is FmtNode but panics on error.
func FmtNodeP(n ast.Node) []byte {
	b, err := FmtNode(n)
	if err != nil {
		panic(err)
	}
	return b
}

// Format formats a cue.Value using a Thema-standard set of options.
//
// It also sanitizes out weird insertions the CUE compiler makes, as necessary.
func Format(v cue.Value) ast.Node {
	n := v.Syntax(
		cue.All(),
		cue.Definitions(true),
		cue.Docs(true),
	)

	sanitizeBottomLiteral(n)
	return n
}

// Removes the comment that the CUE internal exporter adds on bottom literals,
// which can cause format.Node to produce invalid CUE. This seems to only happen
// because the CUE compiler injects these comments on a bottom when it's a
// literal in the source
//
// TODO file a bug upstream, we shouldn't have to do this
func sanitizeBottomLiteral(n ast.Node) {
	ast.Walk(n, func(n ast.Node) bool {
		if x, ok := n.(*ast.BottomLit); ok {
			x.SetComments(nil)
		}
		return true
	}, nil)
}

// ToExpr converts a node to an expression. If it is a file, it will return it
// as a struct. If it is an expression, it will return it as is. Otherwise it
// panics.
//
// Copied from cuelang.org/go/internal
func ToExpr(n ast.Node) ast.Expr {
	switch x := n.(type) {
	case nil:
		return nil

	case ast.Expr:
		return x

	case *ast.File:
		start := 0
	outer:
		for i, d := range x.Decls {
			switch d.(type) {
			case *ast.Package, *ast.ImportDecl:
				start = i + 1
			case *ast.CommentGroup, *ast.Attribute:
			default:
				break outer
			}
		}
		decls := x.Decls[start:]
		if len(decls) == 1 {
			if e, ok := decls[0].(*ast.EmbedDecl); ok {
				return e.Expr
			}
		}
		return &ast.StructLit{Elts: decls}

	default:
		panic(fmt.Sprintf("Unsupported node type %T", x))
	}
}
