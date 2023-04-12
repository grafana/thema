package cue

import (
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/ast/astutil"
	"cuelang.org/go/cue/token"
	"github.com/grafana/thema"
)

// RewriteLegacyLineage takes a [cue.Value] and a [cue.Path] to a legacy Thema
// lineage declaration within that path, and rewrites the lineage to conform to
// the new form expected for declaring lineages.
//
// It is required that the provided inst [cue.Value] point to the root of a
// package instance. If the entire package instance is the thema lineage, the
// provided path may be empty.
//
// Lineage definitions implicity unified across multiple files in the same package
// cannot be rewritten by this function.
func RewriteLegacyLineage(inst cue.Value, path cue.Path) (*ast.File, error) {
	// TODO preserve the old #Lineage and do just a basic validity check of the input against it
	if inst.BuildInstance() == nil {
		return nil, fmt.Errorf("provided cue.Value must be the root of a CUE package instance")
	}
	v := inst.LookupPath(path)

	// If the input imports and unifies thema.#Lineage, it will fail here because
	// this version of thema has changed that definition. So, walk down the
	// unification list and look for a thing with a Source that's a struct literal,
	// or a field with a value that's a struct literal, where in either case the struct
	// literal contains a field named seqs.
	if found := findSeqsVal(v); found != nil {
		v = *found
	} else if len(path.Selectors()) != 0 {
		return nil, fmt.Errorf("could not find parent of seqs field in AST")
	}

	parts, err := rewrite(v)
	if err != nil {
		return nil, err
	}
	// make structs to have the fields
	fields := ast.NewStruct(
		"schemas", parts.schemas,
		"lenses", parts.lenses,
	)

	src := v.Source()
	var ret *ast.File
	// find which file the seqs node we're going to modify appears in
	var found bool
	for _, f := range inst.BuildInstance().Files {
		ast.Walk(f, func(node ast.Node) bool {
			if !found {
				found = node == src
			}
			return !found
		}, nil)

		if found {
			ret = f
			break
		}
	}
	if !found {
		// shouldn't be reachable - v.Source() can't be nil
		panic("should be unreachable")
	}

	astutil.Apply(src, func(cursor astutil.Cursor) bool {
		switch x := cursor.Node().(type) {
		case *ast.Field:
			if str, _, err := ast.LabelName(x.Label); err == nil && str == "seqs" {
				cursor.Replace(fields.Elts[0])
				cursor.InsertAfter(fields.Elts[1])
			}
		}
		return true
	}, nil)

	return ret, nil
}

type lineageParts struct {
	schemas *ast.ListLit
	lenses  *ast.ListLit
}

func rewrite(v cue.Value) (*lineageParts, error) {
	// extract the schemas and lenses from the provided cue.Value
	schlist, lenslist, err := extractParts(v)
	if err != nil {
		return nil, fmt.Errorf("failed extracting parts from legacy lineage: %w", err)
	}

	parts := &lineageParts{
		schemas: &ast.ListLit{},
		lenses:  &ast.ListLit{},
	}
	for _, sch := range schlist {
		schast, err := sch.toAST()
		if err != nil {
			return nil, err
		}
		parts.schemas.Elts = append(parts.schemas.Elts, schast)
	}

	for _, lens := range lenslist {
		lensast, err := lens.toAST()
		if err != nil {
			return nil, err
		}
		parts.lenses.Elts = append(parts.lenses.Elts, lensast)
	}

	return parts, nil
}

type legacySchema struct {
	synv thema.SyntacticVersion
	raw  cue.Value
}

func (sch legacySchema) toAST() (ast.Expr, error) {
	if sch.raw.Source() == nil {
		return nil, fmt.Errorf("schema %s returns nil for source, cannot migrate", sch.synv)
	}

	schsrc, is := sch.raw.Source().(*ast.StructLit)
	if !is {
		return nil, fmt.Errorf("source for schema %s is not a struct literal", sch.synv)
	}
	schsrc.Lbrace = token.NoPos

	return ast.NewStruct(
		"version", synvToAST(sch.synv),
		"schema", schsrc,
	), nil
}

type legacyLens struct {
	to, from        thema.SyntacticVersion
	mapper, lacunas cue.Value
}

// Rewrite the bodies of the lenses and any lacunas to have selectors using the
// new field names
func (ll legacyLens) rewriteFromTo(cursor astutil.Cursor) bool {
	switch x := cursor.Node().(type) {
	case *ast.SelectorExpr:
		if id, is := x.X.(*ast.Ident); is {
			switch id.Name {
			case "from":
				x.X = ast.NewLit(token.STRING, "input")
			case "to":
				x.X = ast.NewLit(token.STRING, "result")
			default:
				return false
			}
		}
		// ONLY want to change the first element in any chain of selectors. So
		// always stop searching once we've encountered a single one of these
		return false
	}
	return true
}

func (ll legacyLens) toAST() (ast.Expr, error) {
	var mapsrc ast.Expr
	if !ll.mapper.Exists() {
		missing := &ast.BottomLit{}
		missing.AddComment(&ast.CommentGroup{
			List: []*ast.Comment{
				{
					Text: "// TODO implement this lens",
				},
			},
			Position: 4, //nolint:gomnd
		})
		mapsrc = missing
	} else if ll.mapper.Source() == nil {
		return nil, fmt.Errorf("lens %s->%s returns nil for source, cannot migrate", ll.from, ll.to)
	} else {
		field := ll.mapper.Source().(*ast.Field)
		existing, is := field.Value.(*ast.StructLit)
		if !is {
			return nil, fmt.Errorf("lens %s->%s mapper is not a struct literal", ll.from, ll.to)
		}
		existing.Lbrace = token.NoPos
		astutil.Apply(existing, ll.rewriteFromTo, nil)
		mapsrc = existing
	}

	var lacsrc ast.Expr = ast.NewList()
	if ll.lacunas.Exists() && ll.lacunas.Source() != nil {
		src := ll.lacunas.Source().(*ast.Field).Value.(*ast.ListLit)
		astutil.Apply(src, ll.rewriteFromTo, nil)
		lacsrc = src
	}
	return ast.NewStruct(
		"to", synvToAST(ll.to),
		"from", synvToAST(ll.from),
		"input", ast.NewIdent("_"),
		"result", mapsrc,
		"lacunas", lacsrc,
	), nil
}

func findSeqsVal(v cue.Value) *cue.Value {
	if isStructLiteral(v) {
		return &v
	}

	if op, dvals := v.Expr(); op == cue.AndOp {
		for _, val := range dvals {
			// See if it contains a value at the seqs path at all
			if !val.LookupPath(cue.MakePath(cue.Str("seqs"))).Exists() {
				continue
			}
			if dv := findSeqsVal(val); dv != nil {
				return dv
			}
		}
	}

	return nil
}

func synvToAST(v thema.SyntacticVersion) ast.Expr {
	return ast.NewList(&ast.BasicLit{
		Kind:  token.INT,
		Value: fmt.Sprintf("%d", v[0]),
	}, &ast.BasicLit{
		Kind:  token.INT,
		Value: fmt.Sprintf("%d", v[1]),
	})
}

func isStructLiteral(v cue.Value) bool {
	var is bool
	switch x := v.Source().(type) {
	case *ast.StructLit:
		return true
	case *ast.Field:
		_, is = x.Value.(*ast.StructLit)
	}
	return is
}

func extractParts(raw cue.Value) ([]legacySchema, []legacyLens, error) {
	schlist := make([]legacySchema, 0)
	lenslist := make([]legacyLens, 0)

	seqiter, err := raw.LookupPath(cue.MakePath(cue.Str("seqs"))).List()
	if err != nil {
		return nil, nil, err
	}

	var majv, lastminv uint
	for seqiter.Next() {
		var minv uint
		seq := seqiter.Value()
		schemas := seq.LookupPath(cue.MakePath(cue.Str("schemas")))
		schiter, err := schemas.List()
		if err != nil {
			return nil, nil, err
		}

		if majv != uint(0) {
			forward := legacyLens{
				to:      thema.SV(majv-1, lastminv-1),
				from:    thema.SV(majv, 0),
				mapper:  seq.LookupPath(cue.ParsePath("lens.forward.rel")),
				lacunas: seq.LookupPath(cue.ParsePath("lens.forward.lacunas")),
			}
			reverse := legacyLens{
				to:      thema.SV(majv-1, lastminv-1),
				from:    thema.SV(majv, 0),
				mapper:  seq.LookupPath(cue.ParsePath("lens.reverse.rel")),
				lacunas: seq.LookupPath(cue.ParsePath("lens.reverse.lacunas")),
			}
			lenslist = append(lenslist, reverse, forward)
		}

		for schiter.Next() {
			schlist = append(schlist, legacySchema{
				synv: thema.SyntacticVersion{majv, minv},
				raw:  schiter.Value(),
			})
			if minv != uint(0) {
				lenslist = append(lenslist, legacyLens{
					to:   thema.SV(majv, minv-1),
					from: thema.SV(majv, minv),
				})
			}
			minv++
		}

		lastminv = minv
		majv++
	}

	return schlist, lenslist, nil
}
