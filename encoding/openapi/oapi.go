package openapi

import (
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/encoding/openapi"
	"cuelang.org/go/pkg/strings"
	"github.com/grafana/thema"
	"github.com/grafana/thema/internal/astutil"
	"github.com/grafana/thema/internal/cuetil"
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

// Config controls OpenAPI derivation from a Thema schema.
type Config struct {
	*openapi.Config

	// Group indicates that the type is a grouped lineage - the root schema itself
	// does not represent an object that is ever expected to exist independently,
	// but each of its top-level fields do, including definitions and optional fields.
	//
	// NOTE - https://github.com/grafana/thema/issues/62 is the issue for formalizing
	// the group concept. Fixing that issue will obviate this field. Once fixed,
	// this field will be deprecated and ignored.
	Group bool

	// RootName specifies the name to use for the type representing the root of the
	// schema. If empty, this defaults to titlecasing of the lineage name.
	//
	// No-op if Group is true.
	RootName string
}

// GenerateSchema creates an OpenAPI document that represents the provided Thema
// Schema as an OpenAPI schema component.
//
// Returns the result as a CUE AST, which is suitable for direct manipulation and
// marshaling to either JSON or YAML.
func GenerateSchema(sch thema.Schema, cfg *Config) (*ast.File, error) {
	if cfg == nil {
		cfg = &Config{Config: &openapi.Config{}}
	}

	ctx := sch.Underlying().Context()
	bpath := sch.Underlying().Path()

	onf := cfg.NameFunc
	nf := func(val cue.Value, path, defpath cue.Path, name string) string {
		// tpath := cuetil.ReplacePathPrefix(path, bpath, cue.MakePath(cue.Str(name)))
		// tpath = cuetil.ReplacePathPrefix(tpath, defpath, cue.MakePath(cue.Str(name)))
		tpath := cuetil.TrimPathPrefix(cuetil.TrimPathPrefix(path, bpath), defpath)

		if tpath.String() == defpath.String() {
			return name
		}
		if cuetil.LastSelectorEq(tpath, cue.Str("joinSchema")) {
			return ""
		}
		if onf != nil {
			return onf(val, tpath)
		}
		return strings.Trim(tpath.String(), "?#")
	}

	// TODO always do this, then cannibalize the result if Group=true
	if !cfg.Group {
		name := util.SanitizeLabelString(sch.Lineage().Name())
		if cfg.RootName != "" {
			name = util.SanitizeLabelString(sch.Lineage().Name())
		}

		v := ctx.CompileString(fmt.Sprintf("#%s: _", name))
		defpath := cue.MakePath(cue.Def(name))
		defsch := v.FillPath(defpath, sch.Underlying())

		cfg.NameFunc = func(val cue.Value, path cue.Path) string {
			return nf(val, path, defpath, name)
		}

		return doGenerate(defsch, sch.Version(), defpath, cfg.Config)
	}
	iter, err := sch.Underlying().Fields(cue.Definitions(true), cue.Optional(true))
	if err != nil {
		panic(fmt.Errorf("unreachable - should always be able to get iter for struct kinds: %w", err))
	}
	f := &ast.File{}

	for iter.Next() {
		val, sel := iter.Value(), iter.Selector()
		// TODO test this against various inputs
		name := strings.Trim(sel.String(), "?#")

		v := ctx.CompileString(fmt.Sprintf("#%s: _", name))
		defpath := cue.MakePath(cue.Def(name))
		defsch := v.FillPath(defpath, val)

		cfgi := *cfg.Config
		cfgi.NameFunc = func(val cue.Value, path cue.Path) string {
			return nf(val, path, defpath, name)
		}

		part, err := doGenerate(defsch, sch.Version(), defpath, &cfgi)
		if err != nil {
			return nil, fmt.Errorf("failed generation for grouped field %s: %w", sel, err)
		}

		elem := orp(astutil.GetFieldByLabel(orp(astutil.GetFieldByLabel(part, "components")).
			Value, "schemas"))

		f.Decls = append(f.Decls, elem)
	}
	return f, nil
}

func orp[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}

func doGenerate(v cue.Value, ver thema.SyntacticVersion, p cue.Path, cfg *openapi.Config) (*ast.File, error) {
	cfg.SelfContained = true
	if cfg.Info == nil {
		cfg.Info = ast.NewStruct(
			"title", ast.NewString(strings.Trim(p.String(), "#?")),
			"version", ast.NewString(ver.String()),
		)
	}
	return openapi.Generate(v, cfg)
}
