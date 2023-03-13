package ts

import (
	"bufio"
	"strings"

	cast "cuelang.org/go/cue/ast"
	"github.com/grafana/cuetsy/ts/ast"
	"github.com/kr/text"
)

type (
	File = ast.File
	Node = ast.Node
	Decl = ast.Decl
	Expr = ast.Expr
)

func Ident(name string) ast.Ident {
	return ast.Ident{Name: name}
}

func Names(names ...string) ast.Names {
	idents := make(ast.Idents, len(names))
	for i, n := range names {
		idents[i] = Ident(n)
	}

	return ast.Names{
		Idents: idents,
	}
}

func Union(elems ...Expr) Expr {
	switch len(elems) {
	case 0:
		return nil
	case 1:
		return elems[0]
	}

	var U Expr = elems[0]
	for _, e := range elems[1:] {
		U = ast.BinaryExpr{
			Op: "|",
			X:  U,
			Y:  e,
		}
	}

	return ast.ParenExpr{Expr: U}
}

func Raw(data string) ast.Raw {
	// pc, file, no, ok := runtime.Caller(1)
	// details := runtime.FuncForPC(pc)
	// if ok && details != nil {
	// 	fmt.Printf("fix: ts.Raw used by %s at %s#%d\n", details.Name(), file, no)
	// }

	return ast.Raw{Data: data}
}

func Object(fields map[string]Expr) Expr {
	elems := make([]ast.KeyValueExpr, 0, len(fields))
	for k, v := range fields {
		elems = append(elems, ast.KeyValueExpr{
			Key:   Ident(k),
			Value: v,
		})
	}
	return ast.ObjectLit{Elems: elems}
}

func List(elems ...Expr) Expr {
	return ast.ListLit{Elems: elems}
}

func Null() Expr {
	return Ident("null")
}

func Str(s string) Expr {
	return ast.Str{Value: s}
}

// TODO: replace with generic num?
func Int(i int64) Expr {
	return ast.Num{N: i}
}
func Float(f float64) Expr {
	return ast.Num{N: f}
}

func Bool(b bool) Expr {
	if b {
		return Ident("true")
	}
	return Ident("false")
}

// CommentFromString takes a string input and formats it as an ast.CommentList.
//
// Line breaks are automatically inserted to minimize raggedness, with a loose
// width limit the provided lim.
//
// If the jsdoc param is true, the resulting comment will be formatted with
// JSDoc ( /** ... */ )-style. Otherwise, ordinary comment leader ( // ... ) will
// be used.
//
// The returned ast.CommentList will have the default CommentAbove position.
func CommentFromString(s string, lim int, jsdoc bool) ast.Comment {
	var b strings.Builder
	prefix := func() { b.WriteString("// ") }
	if jsdoc {
		b.WriteString("/**\n")
		prefix = func() { b.WriteString(" * ") }
	}

	scanner := bufio.NewScanner(strings.NewReader(text.Wrap(s, lim)))
	var i int
	for scanner.Scan() {
		if i != 0 {
			b.WriteString("\n")
		}
		prefix()
		b.WriteString(scanner.Text())
		i++
	}
	if jsdoc {
		b.WriteString("\n */\n")
	}

	return ast.Comment{
		Text: b.String(),
	}
}

// CommentFromCUEGroup creates an ast.CommentList from a CUE AST CommentGroup.
//
// Original line breaks are preserved, in keeping with principles of semantic line breaks.
func CommentFromCUEGroup(cg *cast.CommentGroup, jsdoc bool) ast.Comment {
	var b strings.Builder
	pos := ast.CommentAbove
	if cg.Line {
		pos = ast.CommentInline
	}

	prefix := func() { b.WriteString("// ") }
	if jsdoc {
		b.WriteString("/**")
		if cg.Line {
			prefix = func() { b.WriteString(" ") }
		} else {
			b.WriteString("\n")
			prefix = func() { b.WriteString(" * ") }
		}
	}

	scanner := bufio.NewScanner(strings.NewReader(cg.Text()))
	var i int
	for scanner.Scan() {
		if i != 0 {
			b.WriteString("\n")
		}
		prefix()
		b.WriteString(scanner.Text())
		i++
	}
	if jsdoc {
		if !cg.Line {
			b.WriteString("\n")
		}
		b.WriteString(" */")
	}

	return ast.Comment{
		Text: b.String(),
		Pos:  pos,
	}
}
