package cue

import (
	"bytes"
	"fmt"
	"text/template"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/parser"
	"github.com/grafana/thema"
	"github.com/grafana/thema/internal/astutil"
	"github.com/grafana/thema/internal/compat"
)

// NewLineage constructs a CUE ast.File with a new lineage declaration in it,
// using the provided cue.Value as the 0.0 schema.
//
// The name parameter is used as the lineage name, and must be a non-empty
// string. pkgname is used as the returned file's package declaration. If
// pkgname is empty, the resulting file will have no package declaration.
func NewLineage(sch cue.Value, name, pkgname string) (*ast.File, error) {
	x := astutil.Format(sch)
	switch x.(type) {
	case *ast.File, ast.Expr:
		x = astutil.ToExpr(x)
	}
	b, err := astutil.FmtNode(x)
	if err != nil {

		return nil, fmt.Errorf("failed to convert input schema to string: %ww", err)
	}

	vars := linTplVars{
		PkgName: pkgname,
		Name:    name,
		Sch:     string(b),
	}

	var buf bytes.Buffer
	err = emptyLineage.Execute(&buf, vars)
	if err != nil {
		return nil, fmt.Errorf("template generation failed: %w", err)
	}

	f, err := parser.ParseFile(name+".cue", buf.Bytes(), parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("%s\nerror while parsing generated output: %w", buf.String(), err)
	}
	return f, nil
}

// InsertSchemaNodeAs inserts the provided schema ast.Expr into the provided
// lineage ast.Node at the position corresponding to the provided version. The
// provided schema will either replace an existing schema, or be appended to the
// end of an existing sequence.
//
// The provided lineage node is modified in place. Corresponding lenses are not
// generated. The result is not checked for Thema validity. Behavior is
// undefined if the provided lineage node is not well-formed.
func InsertSchemaNodeAs(lin ast.Node, sch ast.Expr, v thema.SyntacticVersion) error {
	seql := astutil.FindSeqs(lin)
	if seql == nil {
		return fmt.Errorf("could not find seqs list in input - invalid lineage ast?")
	}

	// Handle inserting new sequence path separately
	if v[0] == uint(len(seql.Elts)) {
		ast.AddComment(sch, versionComment(v))
		seql.Elts = append(seql.Elts, newSequenceNode(sch))
		return nil
	}

	seql, err := astutil.SchemaListFor(lin, v[0])
	if err != nil {
		return err
	}

	if v[1] > uint(len(seql.Elts)) {
		return fmt.Errorf("cannot insert version %s, previous version does not exist in lineage", v)
	}

	ast.AddComment(sch, versionComment(v))
	if v[1] == uint(len(seql.Elts)) {
		// append
		seql.Elts = append(seql.Elts, sch)
	} else {
		// replace
		seql.Elts[v[1]] = sch
	}

	return nil
}

type linTplVars struct {
	PkgName string
	Name    string
	Sch     string
}

// TODO replace with collection of templates
var emptyLineage = template.Must(template.New("newlin").Parse(`
{{- if ne .PkgName "" }}package {{ .PkgName }}
{{end}}import "github.com/grafana/thema"

thema.#Lineage
name: "{{ .Name }}"
seqs: [
    {
        schemas: [
            {{ .Sch }},
        ]
    },
]
`))

// Append adds the provided cue.Value as a new schema to the provided Lineage.
//
// If the provided schema is backwards compatible with the latest schema in the
// lineage, the new schema will be appended to the latest sequence (minor
// version bump). Otherwise, a new sequence will be created with the provided
// schema as its only element (major version bump).
func Append(lin thema.Lineage, sch cue.Value) (ast.Node, error) {
	linf := astutil.Format(lin.Underlying()).(*ast.File)
	schnode := astutil.ToExpr(astutil.Format(sch))

	lv := thema.LatestVersion(lin)
	lsch := thema.SchemaP(lin, lv)
	if err := compat.ThemaCompatible(lsch.Underlying(), sch); err == nil {
		// Is compatible, append to same sequence
		tgtv := thema.SyntacticVersion{lv[0], lv[1] + 1}
		ast.AddComment(schnode, versionComment(tgtv))

		schl, err := astutil.LatestSchemaList(linf)
		if err != nil {
			return nil, fmt.Errorf("could not get lineage's latest seq's schema list: %w", err)
		}

		schl.Elts = append(schl.Elts, schnode)
		// TODO add boilerplate lenses, etc.
	} else {
		// Not compatible, start a new sequence
		tgtv := thema.SyntacticVersion{lv[0] + 1, 0}
		ast.AddComment(schnode, versionComment(tgtv))

		seql := astutil.FindSeqs(linf)
		if seql == nil {
			return nil, fmt.Errorf("could not find seqs list in lineage input")
		}

		seql.Elts = append(seql.Elts, newSequenceNode(schnode))
		// TODO add boilerplate lenses, etc.
	}

	return linf, nil
}

func newSequenceNode(sch ast.Expr) *ast.StructLit {
	if sch == nil {
		sch = ast.NewStruct() // use empty struct
	}

	return ast.NewStruct(&ast.Field{
		Label: ast.NewString("schemas"),
		Value: ast.NewList(sch),
	})
}

func versionComment(v thema.SyntacticVersion) *ast.CommentGroup {
	return &ast.CommentGroup{
		Doc:  true,
		Line: true,
		List: []*ast.Comment{
			&ast.Comment{
				Text: fmt.Sprint("// v", v.String()),
			},
		},
	}
}
