package openapi

import (
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/errors"
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

	// Group indicates that the [thema.Schema] is from a grouped lineage - the root
	// schema itself does not represent an object that is ever expected to exist
	// independently, but each of its top-level fields do, including definitions and
	// optional fields.
	//
	// NOTE - https://github.com/grafana/thema/issues/62 is the issue for formalizing
	// the group concept. Fixing that issue will obviate this field. Once fixed,
	// this field will be deprecated and ignored.
	Group bool

	// RootName specifies the name to use for the type representing the root of the
	// schema. If empty, this defaults to titlecasing of the lineage name.
	//
	// No-op if [Group] is true.
	RootName string

	// Subpath specifies a path within the provided [thema.Schema] that should be
	// translated, rather than the root schema.
	//
	// No-op if [Group] is true.
	Subpath cue.Path
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

	gen := &oapiGen{
		cfg: cfg,
		sch: sch,
		onf: cfg.NameFunc,
		// Underlying() gives us the _#schema ref to the definition. We need to
		// eliminate path elements leading into the actual user-specified definition.
		schdef: sch.Underlying().LookupPath(cue.MakePath(cue.Hid("_#schema", "github.com/grafana/thema"))),
		schraw: sch.Underlying().LookupPath(cue.MakePath(cue.Str("schema"))),
		join:   sch.Underlying().LookupPath(cue.MakePath(cue.Hid("_join", "github.com/grafana/thema"))),
		bpath:  sch.Underlying().Path(),
	}

	var decls []ast.Decl
	var err error
	if cfg.Group {
		decls, err = genGroup(gen)
	} else {
		decls, err = genSingle(gen)
	}
	if err != nil {
		return nil, err
	}

	// TODO recursively sort output to improve stability of output
	return &ast.File{
		Decls: []ast.Decl{
			ast.NewStruct(
				"openapi", ast.NewString("3.0.0"),
				"info", ast.NewStruct(
					"title", ast.NewString(gen.name),
					"version", ast.NewString(sch.Version().String()),
				),
				"paths", ast.NewStruct(),
				"components", ast.NewStruct(
					"schemas", &ast.StructLit{Elts: decls},
				),
			),
		},
	}, nil
}

type oapiGen struct {
	cfg *Config
	sch thema.Schema
	// the #SchemaDef._#schema
	schdef cue.Value
	// the #SchemaDef.schema
	schraw cue.Value
	// the #SchemaDef._join
	join cue.Value

	// overall name for the generated oapi doc
	name string

	// original NameFunc
	onf func(cue.Value, cue.Path) string

	// full prefix path that leads up to the #SchemaDef, e.g. lin._sortedSchemas[0]
	bpath cue.Path
}

func genGroup(gen *oapiGen) ([]ast.Decl, error) {
	ctx := gen.sch.Underlying().Context()
	iter, err := gen.schdef.Fields(cue.Definitions(true), cue.Optional(true))
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

		cfgi := *gen.cfg.Config
		cfgi.NameFunc = func(val cue.Value, path cue.Path) string {
			return gen.nfSingle(val, path, defpath, name)
		}
		cfgi.Info = newHeader(defpath.String(), gen.sch.Version())

		part, err := openapi.Generate(defsch, &cfgi)
		if err != nil {
			return nil, fmt.Errorf("failed generation for grouped field %s: %w", sel, err)
		}

		decls = append(decls, getSchemas(part)...)
	}

	gen.name = util.SanitizeLabelString(gen.sch.Lineage().Name())
	return decls, nil
}

func genSingle(gen *oapiGen) ([]ast.Decl, error) {
	hasSubpath := len(gen.cfg.Subpath.Selectors()) > 0

	if hasSubpath {
		for i, sel := range gen.cfg.Subpath.Selectors() {
			if !gen.schdef.Allows(sel) {
				return nil, errors.Newf(cuetil.FirstNonThemaPos(gen.schraw), "subpath %q not present in schema", cue.MakePath(gen.cfg.Subpath.Selectors()[:i+1]...))
			}
		}
		if err := pathAllowed(gen.schdef, gen.cfg.Subpath); err != nil {
			return nil, err
		}
		gen.schdef = gen.schdef.LookupPath(gen.cfg.Subpath)
	}

	name := util.SanitizeLabelString(gen.sch.Lineage().Name())
	if gen.cfg.RootName != "" {
		name = gen.cfg.RootName
	} else if hasSubpath {
		sel := gen.cfg.Subpath.Selectors()
		name = sel[len(sel)-1].String()
	}

	v := gen.sch.Underlying().Context().CompileString(fmt.Sprintf("#%s: _", name))
	defpath := cue.MakePath(cue.Def(name))
	// v, defpath := newEmptyDef(gen.sch.Underlying().Context(), name)
	defsch := v.FillPath(defpath, gen.schdef)

	gen.cfg.NameFunc = func(val cue.Value, path cue.Path) string {
		// fmt.Println("NF===", path)
		// if gen.sch.Lineage().Name() == "maps" {
		// 	// fmt.Println("NF===", path, p(val))
		// }
		return gen.nfSingle(val, path, defpath, name)
	}
	gen.cfg.Info = newHeader(name, gen.sch.Version())

	// fmt.Println(gen.sch.Lineage().Name(), defsch.Eval())
	// fmt.Println(gen.sch.Lineage().Name(), defsch)
	f, err := openapi.Generate(defsch.Eval(), gen.cfg.Config)
	if err != nil {
		return nil, err
	}

	gen.name = name
	return getSchemas(f), nil
}

// For generating a single, our NameFunc must:
// - Eliminate any path prefixes on the element, both internal lineage and wrapping
// - Replace the name "_#schema" with the desired name
// - Call the user-provided NameFunc, if any
// - Remove CUE markers like #, !, ?
func (gen *oapiGen) nfSingle(val cue.Value, path, defpath cue.Path, name string) string {
	tpath := cuetil.TrimPathPrefix(trimThemaPathPrefix(path, gen.bpath), defpath)

	if path.String() == "" || tpath.String() == defpath.String() {
		return name
	}
	switch val {
	case gen.schraw, gen.join:
		return ""
	case gen.schdef:
		return name
	}
	if gen.onf != nil {
		return gen.onf(val, tpath)
	}
	return strings.Trim(tpath.String(), "?#")
}

func getSchemas(f *ast.File) []ast.Decl {
	compos := orp(astutil.GetFieldByLabel(f, "components"))
	schemas := orp(astutil.GetFieldByLabel(compos.Value, "schemas"))
	return schemas.Value.(*ast.StructLit).Elts
}

func pathAllowed(v cue.Value, path cue.Path) error {
	for i, sel := range path.Selectors() {
		if !v.Allows(sel) {
			return errors.Newf(v.Pos(), "subpath %q not present in schema", cue.MakePath(path.Selectors()[:i+1]...))
		}
	}

	return nil
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

func trimThemaPathPrefix(p, base cue.Path) cue.Path {
	if !cuetil.PathHasPrefix(p, base) {
		return p
	}

	rest := p.Selectors()[len(base.Selectors()):]
	if len(rest) == 0 {
		return cue.Path{}
	}
	switch rest[0].String() {
	case "schema", "_#schema", "_join", "joinSchema":
		return cue.MakePath(rest[1:]...)
	default:
		return cue.MakePath(rest...)
	}
}

// func p(val cue.Value) string {
// 	return string(astutil.FmtNodeP(astutil.Format(val)))
// }
//
// func newEmptyDef(ctx *cue.Context, name string) (cue.Value, cue.Path) {
// 	rnd := randSeq(20)
// 	v := ctx.CompileString(fmt.Sprintf("%s: #%s: _", rnd, name))
// 	return v.LookupPath(cue.MakePath(cue.Str(rnd))), cue.MakePath(cue.Def(name))
// }

// var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
//
// // randSeq produces random (basic, not crypto) letters of a given length.
// func randSeq(n int) string {
// 	b := make([]rune, n)
// 	for i := range b {
// 		b[i] = letters[rand.Intn(len(letters))]
// 	}
// 	return string(b)
// }
