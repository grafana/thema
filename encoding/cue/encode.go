package cue

import (
	"bytes"
	"fmt"
	"text/template"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/format"
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
	syn := sch.Syntax(
		cue.Definitions(true),
		cue.Hidden(true),
		cue.Optional(true),
		cue.Attributes(true),
		cue.Docs(true),
	)

	b, err := format.Node(syn)
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

	f, err := parser.ParseFile(name+".cue", buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("%s\nerror while parsing generated output: %w\n", buf.String(), err)
	}
	return f, nil
}

type linTplVars struct {
	PkgName string
	Name    string
	Sch     string
}

// TODO replace with collection of templates
var emptyLineage = template.Must(template.New("newlin").Parse(`
{{- if ne .PkgName "" }}package {{ .PkgName }}
{{end}}
import "github.com/grafana/thema"

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
	linf := tonode(lin.UnwrapCUE()).(*ast.File)
	schnode := tonode(sch).(*ast.StructLit)

	lv := thema.LatestVersion(lin)
	lsch := thema.SchemaP(lin, lv)
	if err := compat.ThemaCompatible(lsch.UnwrapCUE(), sch); err == nil {
		// Is compatible, append to same sequence
		tgtv := thema.SyntacticVersion{lv[0], lv[1] + 1}
		schnode.AddComment(versionComment(tgtv))

		schl, err := astutil.LatestSchemaList(linf)
		if err != nil {
			return nil, fmt.Errorf("could not get lineage's latest seq's schema list: %w", err)
		}

		schl.Elts = append(schl.Elts, schnode)
		// TODO add boilerplate lenses, etc.
	} else {
		// Not compatible, start a new sequence
		tgtv := thema.SyntacticVersion{lv[0] + 1, 0}
		schnode.AddComment(versionComment(tgtv))

		seql := astutil.FindSeqs(linf)
		if seql == nil {
			return nil, fmt.Errorf("could not find seqs list in lineage input")
		}

		seql.Elts = append(seql.Elts, newSequenceNode(schnode))
		// TODO add boilerplate lenses, etc.
	}

	return linf, nil
}

func newSequenceNode(sch *ast.StructLit) *ast.StructLit {
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
				Text: fmt.Sprint("// ", v.String()),
			},
		},
	}
}

// Into prints a
func Into(lin thema.Lineage, v cue.Value, p cue.Path) (ast.Node, error) {
	panic("TODO")
}

func fmtn(n ast.Node) []byte {
	b, err := format.Node(n)
	if err != nil {
		panic(fmt.Errorf("failed to convert input schema to string: %w", err))
	}
	b, err = format.Source(b, format.TabIndent(true), format.Simplify())
	if err != nil {
		panic(fmt.Errorf("could not reformat to canonical source form: %w", err))
	}
	return b
}

func tonode(v cue.Value) ast.Node {
	n := v.Syntax(
		cue.Raw(),
		cue.Definitions(true),
		cue.Hidden(true),
		cue.Optional(true),
		cue.Attributes(true),
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
