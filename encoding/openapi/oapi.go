package openapi

import (
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/encoding/openapi"
	"github.com/grafana/thema"
	"github.com/grafana/thema/internal/util"
)

// newLineage creates a new Lineage containing one schema from some some valid
// OpenAPI input bytes. The schemaPath provided must identify a single schema
// component to turn into a Thema schema in the new Lineage.
func newLineage(b []byte, schemaPath string) (thema.Lineage, error) {
	panic("not yet implemented")
}

// appendSchemaToLineage takes an OpenAPI input document and a path to a schema within
// that document, and appends it to the provided lineage. The version for the appended
// schema is chosen automatically: it is either appended to the tail of the latest sequence
// if backwards compatible, or a new sequence is created if not.
//
// Appending a schema to a lineage without also defining the corresponding
// necessary lenses is guaranteed to produce an invalid lineage. The only useful
// thing to do with the return value is write it to disk so an author can do
// that work. Thus, the return value is a []byte.
func appendSchemaToLineage(b []byte, schemaPath string, lin thema.Lineage) ([]byte, error) {
	panic("not yet implemented")
}

// GenerateSchema creates an OpenAPI document that represents the provided Thema
// Schema as an OpenAPI schema component.
func GenerateSchema(sch thema.Schema, cfg *openapi.Config) (*ast.File, error) {
	// Need it to make an instance
	rt := (*cue.Runtime)(sch.UnwrapCUE().Context())

	base := util.RandSeq(10)
	p := cue.MakePath(cue.Str(base), cue.Def(sch.Lineage().Name()))
	pp := cue.MakePath(cue.Str(base))

	v := sch.UnwrapCUE()
	dumpv := sch.Lineage().Library().UnwrapCUE().FillPath(p, v).LookupPath(pp)

	// TODO just Eval()'ing without giving the user choices is not great
	syn := dumpv.Eval().Syntax(
		cue.Definitions(true),
		cue.Hidden(true),
		cue.Optional(true),
		cue.Attributes(true),
		cue.Docs(true),
	)

	// Normalize to ast.File
	f := &ast.File{}
	switch syns := syn.(type) {
	case *ast.StructLit:
		f.Filename = fmt.Sprintf("%s@v%s.cue", sch.Lineage().Name(), sch.Version())
		f.Decls = syns.Elts
	case *ast.File:
		f = syns
	default:
		return nil, fmt.Errorf("schema cue.Value converted to unexpected ast type %T", syn)
	}

	inst, err := rt.CompileFile(f)
	if err != nil {
		return nil, err
	}

	if cfg == nil {
		cfg = &openapi.Config{}
	}
	if cfg.Info == nil {
		cfg.Info = ast.NewStruct(
			"title", ast.NewString(sch.Lineage().Name()),
			"version", ast.NewString(sch.Version().String()),
		)
	}
	return openapi.Generate(inst, cfg)
}
