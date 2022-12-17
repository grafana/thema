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
		cfg = &Config{}
	}
	if cfg.Config == nil {
		cfg.Config = &openapi.Config{}
	}

	ctx := sch.Underlying().Context()
	bpath := sch.Underlying().Path()

	onf := cfg.NameFunc
	nf := func(val cue.Value, path, defpath cue.Path, name string) string {
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
		cfg.Info = newHeader(defpath.String(), sch.Version())

		return openapi.Generate(defsch, cfg.Config)
	}

	iter, err := sch.Underlying().Fields(cue.Definitions(true), cue.Optional(true))
	if err != nil {
		panic(fmt.Errorf("unreachable - should always be able to get iter for struct kinds: %w", err))
	}

	var decls []ast.Decl
	for iter.Next() {
		val, sel := iter.Value(), iter.Selector()
		name := strings.Trim(sel.String(), "?#")

		v := ctx.CompileString(fmt.Sprintf("#%s: _", name))
		defpath := cue.MakePath(cue.Def(name))
		defsch := v.FillPath(defpath, val)

		cfgi := *cfg.Config
		cfgi.NameFunc = func(val cue.Value, path cue.Path) string {
			return nf(val, path, defpath, name)
		}
		cfgi.Info = newHeader(defpath.String(), sch.Version())

		part, err := openapi.Generate(defsch, &cfgi)
		if err != nil {
			return nil, fmt.Errorf("failed generation for grouped field %s: %w", sel, err)
		}


		elems := orp(astutil.GetFieldByLabel(
			orp(astutil.GetFieldByLabel(part, "components")).Value, "schemas")).
				Value.(*ast.StructLit).Elts
		decls = append(decls, elems...)
	}

	return &ast.File{
		Decls: []ast.Decl{
			ast.NewStruct(
				"openapi", ast.NewString("3.0.0"),
				"info", ast.NewStruct(
					"title", ast.NewString(util.SanitizeLabelString(sch.Lineage().Name())),
					"version", ast.NewString(sch.Version().String()),
				),
				"path", ast.NewStruct(),
				"components", ast.NewStruct(
				"schemas", &ast.StructLit{ Elts: decls },
				),
			),
		},
	}, nil
}

func orp[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}

func newHeader(title string, v thema.SyntacticVersion) *ast.StructLit {
	return ast.NewStruct(
		"title", ast.NewString(strings.Trim(title, "#?")),
		"version", ast.NewString(v.String()),
	)
}
