package gocode

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/dstutil"
)

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

func setStar(e dst.Expr) string {
	if _, is := e.(*dst.StarExpr); is {
		return "*"
	}
	return ""
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

// It fixes the "generic" fields. It happens when a value in cue could be different structs.
// For Go it generates a struct with a json.RawMessage field inside and multiple functions to map
// between the different possibilities.
// These structs usually have underscores, and we need to fix these names since isn't a correct way to do it in Go.
func fixRawData() dstutil.ApplyFunc {
	return func(c *dstutil.Cursor) bool {
		f, is := c.Node().(*dst.File)
		if !is {
			return false
		}

		rawFields := make(map[string]string)
		typesWithFunc := make(map[string][]string)
		for _, decl := range f.Decls {
			switch x := decl.(type) {
			// Find the structs that only contains json.RawMessage inside
			case *dst.GenDecl:
				for _, t := range x.Specs {
					if ts, ok := t.(*dst.TypeSpec); ok {
						if tp, ok := ts.Type.(*dst.StructType); ok && len(tp.Fields.List) == 1 {
							if fn, ok := tp.Fields.List[0].Type.(*dst.SelectorExpr); ok {
								rawFields[ts.Name.Name] = fmt.Sprintf("%s.%s", fn.X, fn.Sel.Name)
							}
						}
					}
				}
			// Find the functions of the previous structs to verify that are the ones that we are looking for
			// With this information we can create validators for these json.RawMessage
			case *dst.FuncDecl:
				for _, recv := range x.Recv.List {
					fnType := depoint(recv.Type).(*dst.Ident).Name
					if rawFields[fnType] != "" {
						typesWithFunc[fnType] = append(typesWithFunc[fnType], x.Name.Name)
					}
				}
			}
		}

		// Verify that are the structs that we are looking for.
		existing := make(map[string]bool)
		for k, _ := range typesWithFunc {
			if rawFields[k] != "" {
				existing[k] = true
			}
		}

		dstutil.Apply(f, func(c *dstutil.Cursor) bool {
			switch x := c.Node().(type) {
			case *dst.FuncDecl:
				c.Delete()
			case *dst.GenDecl:
				for _, spec := range x.Specs {
					if tp, ok := spec.(*dst.TypeSpec); ok {
						if existing[tp.Name.Name] {
							c.Delete()
						}
						if st, ok := tp.Type.(*dst.StructType); ok {
							for _, f := range st.Fields.List {
								star := setStar(f.Type)
								switch tx := depoint(f.Type).(type) {
								case *dst.Ident:
									if existing[tx.Name] {
										f.Type = dst.NewIdent(star + "interface{}")
										//f.Type = dst.NewIdent(star + strings.ReplaceAll(tx.Name, "_", ""))
									}
								case *dst.ArrayType:
									if _, ok := tx.Elt.(*dst.Ident); ok {
										//tx.Elt = dst.NewIdent(star + "interface{}")
										tx.Elt = dst.NewIdent(strings.ReplaceAll(tx.Elt.(*dst.Ident).Name, "_", ""))
									}
								}
							}
						}
					}
				}
			}
			return true
		}, nil)

		return true
	}
}
