package gocode

import (
	"fmt"
	"go/token"
	"regexp"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/dstutil"
)

func isSingleTypeDecl(gd *dst.GenDecl) bool {
	if gd.Tok == token.TYPE && len(gd.Specs) == 1 {
		_, is := gd.Specs[0].(*dst.TypeSpec)
		return is
	}
	return false
}

func isAdditionalPropertiesStruct(tspec *dst.TypeSpec) (dst.Expr, bool) {
	strct, is := tspec.Type.(*dst.StructType)
	if is && len(strct.Fields.List) == 1 && strct.Fields.List[0].Names[0].Name == "AdditionalProperties" {
		return strct.Fields.List[0].Type, true
	}
	return nil, false
}

func decoderCompactor() dstutil.ApplyFunc {
	return func(c *dstutil.Cursor) bool {
		f, is := c.Node().(*dst.File)
		if !is {
			return false
		}

		compact := make(map[string]bool)
		// walk the file decls
		for _, decl := range f.Decls {
			if fd, is := decl.(*dst.FuncDecl); is {
				compact[ddepoint(fd.Recv.List[0].Type).(*dst.Ident).Name] = true
			}
		}
		if len(compact) == 0 {
			return false
		}

		replace := make(map[string]dst.Expr)
		// Walk again, looking for types we found
		for _, decl := range f.Decls {
			if gd, is := decl.(*dst.GenDecl); is && isSingleTypeDecl(gd) {
				if tspec := gd.Specs[0].(*dst.TypeSpec); compact[tspec.Name.Name] {
					if expr, is := isAdditionalPropertiesStruct(tspec); is {
						replace[tspec.Name.Name] = expr
					}
				}
			}
		}
		dstutil.Apply(f, func(c *dstutil.Cursor) bool {
			switch x := c.Node().(type) {
			case *dst.FuncDecl:
				c.Delete()
			case *dst.GenDecl:
				if isSingleTypeDecl(x) && compact[x.Specs[0].(*dst.TypeSpec).Name.Name] {
					c.Delete()
				}
			case *dst.Field:
				if id, is := ddepoint(x.Type).(*dst.Ident); is {
					if expr, has := replace[id.Name]; has {
						x.Type = expr
					}
				}
			}
			return true
		}, nil)
		return false
	}
}

func ddepoint(e dst.Expr) dst.Expr {
	if star, is := e.(*dst.StarExpr); is {
		return star.X
	}
	return e
}

// depointerizer returns an AST manipulator that removes redundant
// pointer indirection from the defined types.
func depointerizer(exprs ...dst.Expr) dstutil.ApplyFunc {
	depointers := make(map[dst.Expr]bool)
	for _, expr := range exprs {
		depointers[expr] = true
	}
	return func(c *dstutil.Cursor) bool {
		switch x := c.Node().(type) {
		case *dst.Field:
			if s, is := x.Type.(*dst.StarExpr); is {
				if len(exprs) == 0 {
					x.Type = depoint(s)
					return true
				}
				if _, ok := depointers[s]; ok {
					x.Type = depoint(s)
				}
			}
		}
		return true
	}
}

func depoint(e dst.Expr) dst.Expr {
	if star, is := e.(*dst.StarExpr); is {
		return star.X
	}
	return e
}

func fixTODOComments() dstutil.ApplyFunc {
	return func(cursor *dstutil.Cursor) bool {
		switch f := cursor.Node().(type) {
		case *dst.File:
			for _, d := range f.Decls {
				fixTODOComment(d.Decorations().Start.All())
			}
		case *dst.Field:
			fixTODOComment(f.Decorations().Start.All())
		}

		return true
	}
}

func fixTODOComment(comments []string) {
	todoRegex := regexp.MustCompile("(//) (.*) (TODO.*)")
	if len(comments) > 0 {
		comments[0] = todoRegex.ReplaceAllString(comments[0], "$1 $3")
	}
}

func fixRawData() dstutil.ApplyFunc {
	return func(c *dstutil.Cursor) bool {
		f, is := c.Node().(*dst.File)
		if !is {
			return false
		}

		rawFields := make(map[string]string)
		for _, decl := range f.Decls {
			if gd, is := decl.(*dst.GenDecl); is {
				for _, t := range gd.Specs {
					if ts, ok := t.(*dst.TypeSpec); ok {
						if tp, ok := ts.Type.(*dst.StructType); ok && len(tp.Fields.List) == 1 {
							if fn, ok := tp.Fields.List[0].Type.(*dst.SelectorExpr); ok {
								ts.Name.Name = strings.ReplaceAll(ts.Name.Name, "_", "")
								rawFields[ts.Name.Name] = fmt.Sprintf("%s.%s", fn.X, fn.Sel.Name)
							}
						}
					}
				}
			}
		}

		typesWithFunc := make(map[string][]string)
		for _, decl := range f.Decls {
			if fd, is := decl.(*dst.FuncDecl); is {
				fnType := ddepoint(fd.Recv.List[0].Type).(*dst.Ident).Name
				if rawFields[fnType] != "" {
					typesWithFunc[fnType] = append(typesWithFunc[fnType], fd.Name.Name)
				}
			}
		}

		return true
	}
}
