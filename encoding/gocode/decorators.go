package gocode

import (
	"go/token"

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

func DecoderCompactor() dstutil.ApplyFunc {
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

// Depointerizer returns an AST manipulator that removes redundant
// pointer indirection from the defined types.
func Depointerizer(exprs ...dst.Expr) dstutil.ApplyFunc {
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
