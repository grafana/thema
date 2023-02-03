package gocode

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/dave/dst"
	"github.com/dave/dst/dstutil"
)

// depointerizer returns an AST manipulator that removes redundant
// pointer indirection from the defined types.
func depointerizer(allTypes bool) dstutil.ApplyFunc {
	return func(c *dstutil.Cursor) bool {
		switch x := c.Node().(type) {
		case *dst.Field:
			if s, is := x.Type.(*dst.StarExpr); is {
				if allTypes {
					x.Type = depoint(s)
					return true
				}
				switch deref := depoint(s).(type) {
				case *dst.ArrayType, *dst.MapType:
					x.Type = deref
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
				if isTypeSpec(d) {
					removeGoFieldComment(d.Decorations().Start.All())
				}
				fixTODOComment(d.Decorations().Start.All())
			}
		case *dst.Field:
			if len(f.Names) > 0 {
				removeGoFieldComment(f.Decorations().Start.All())
			}
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

func removeGoFieldComment(comments []string) {
	todoRegex := regexp.MustCompile("(//) ([A-Z].*?) ([A-Z]?.*?) (.*)")
	if len(comments) > 0 {
		matches := todoRegex.FindAllStringSubmatch(comments[0], -1)
		if len(matches) > 0 {
			if strings.EqualFold(matches[0][3], matches[0][2]) {
				comments[0] = fmt.Sprintf("%s %s %s", matches[0][1], matches[0][3], matches[0][4])
			} else {
				r := []rune(matches[0][3])
				if !unicode.IsLower(r[0]) {
					comments[0] = fmt.Sprintf("%s %s %s", matches[0][1], matches[0][3], matches[0][4])
				}
			}
		}
	}
}

func isTypeSpec(d dst.Decl) bool {
	gd, ok := d.(*dst.GenDecl)
	if !ok {
		return false
	}

	_, is := gd.Specs[0].(*dst.TypeSpec)
	return is
}

// It fixes the "generic" fields. It happens when a value in cue could be different structs.
// For Go it generates a struct with a json.RawMessage field inside and multiple functions to map it between the different possibilities.
func fixRawData() dstutil.ApplyFunc {
	return func(c *dstutil.Cursor) bool {
		f, is := c.Node().(*dst.File)
		if !is {
			return false
		}

		rawFields := make(map[string]bool)
		existingRawFields := make(map[string]bool)
		for _, decl := range f.Decls {
			switch x := decl.(type) {
			// Find the structs that only contains one json.RawMessage inside
			case *dst.GenDecl:
				for _, t := range x.Specs {
					if ts, ok := t.(*dst.TypeSpec); ok {
						if tp, ok := ts.Type.(*dst.StructType); ok && len(tp.Fields.List) == 1 {
							if fn, ok := tp.Fields.List[0].Type.(*dst.SelectorExpr); ok {
								if fmt.Sprintf("%s.%s", fn.X, fn.Sel.Name) == "json.RawMessage" {
									rawFields[ts.Name.Name] = true
								}
							}
						}
					}
				}
			// Find the functions of the previous structs to verify that are the ones that we are looking for.
			case *dst.FuncDecl:
				for _, recv := range x.Recv.List {
					fnType := depoint(recv.Type).(*dst.Ident).Name
					if rawFields[fnType] {
						existingRawFields[fnType] = true
					}
				}
			}
		}

		dstutil.Apply(f, func(c *dstutil.Cursor) bool {
			switch x := c.Node().(type) {
			// Delete the functions
			case *dst.FuncDecl:
				c.Delete()
			case *dst.GenDecl:
				for _, spec := range x.Specs {
					if tp, ok := spec.(*dst.TypeSpec); ok {
						// Delete the structs
						if existingRawFields[tp.Name.Name] {
							c.Delete()
						}
						// Set types that was using these structs as interface{}
						if st, ok := tp.Type.(*dst.StructType); ok {
							iterateStruct(st, existingRawFields)
						}
					}
				}
			}
			return true
		}, nil)

		return true
	}
}

func iterateStruct(s *dst.StructType, existingRawFields map[string]bool) {
	for _, f := range s.Fields.List {
		star := setStar(f.Type)
		switch tx := depoint(f.Type).(type) {
		case *dst.Ident:
			if existingRawFields[tx.Name] {
				f.Type = dst.NewIdent(star + "interface{}")
			}
		case *dst.ArrayType:
			if id, ok := tx.Elt.(*dst.Ident); ok {
				if existingRawFields[id.Name] {
					tx.Elt = dst.NewIdent(star + "interface{}")
				}
			}
		case *dst.StructType:
			iterateStruct(tx, existingRawFields)
		}
	}
}
