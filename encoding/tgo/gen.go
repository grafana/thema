package tgo

import (
	"bytes"
	"embed"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/pkg/encoding/yaml"
	"github.com/deepmap/oapi-codegen/pkg/codegen"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/grafana/thema"
	"github.com/grafana/thema/encoding/openapi"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/imports"
)

// All the parsed templates in the tmpl subdirectory
var tmpls *template.Template

//go:embed *.tmpl
var tmplFS embed.FS

func init() {
	base := template.New("gogen").Funcs(template.FuncMap{
		"now": time.Now,
	})
	tmpls = template.Must(base.ParseFS(tmplFS, "*.tmpl"))
}

// TypeConfigOpenAPI governs the behavior of [GenerateTypesOpenAPI].
type TypeConfigOpenAPI struct {
	// PackageName determines the name of the generated Go package. If empty, the
	// lowercase version of the Lineage.Name() is used.
	PackageName string

	// Apply is an optional AST manipulation func that, if provided, will be run
	// against the generated Go file prior to running it through goimports.
	Apply astutil.ApplyFunc
}

// GenerateTypesOpenAPI generates native Go code corresponding to the provided Schema.
func GenerateTypesOpenAPI(sch thema.Schema, cfg *TypeConfigOpenAPI) ([]byte, error) {
	if cfg == nil {
		cfg = new(TypeConfigOpenAPI)
	}

	f, err := openapi.GenerateSchema(sch, nil)
	if err != nil {
		return nil, fmt.Errorf("thema openapi generation failed: %w", err)
	}

	str, err := yaml.Marshal(sch.Lineage().Runtime().Context().BuildFile(f))
	if err != nil {
		return nil, fmt.Errorf("cue-yaml marshaling failed: %w", err)
	}

	loader := openapi3.NewLoader()
	oT, err := loader.LoadFromData([]byte(str))
	if err != nil {
		return nil, fmt.Errorf("loading generated openapi failed: %w", err)
	}
	if cfg.PackageName == "" {
		cfg.PackageName = sch.Lineage().Name()
	}

	ccfg := codegen.Options{
		GenerateTypes: true,
		SkipFmt:       true,
		SkipPrune:     true,
		UserTemplates: map[string]string{
			"imports.tmpl": "package {{ .PackageName }}\n",
		},
	}

	gostr, err := codegen.Generate(oT, cfg.PackageName, ccfg)
	if err != nil {
		return nil, fmt.Errorf("openapi generation failed: %w", err)
	}

	return postprocessGoFile(genGoFile{
		path:  "type_gen.go",
		apply: cfg.Apply,
		in:    []byte(gostr),
	})
}

// BindingConfig governs the behavior of [GenerateLineageBinding].
type BindingConfig struct {
	// Lineage is the Thema lineage for which bindings are to be generated.
	Lineage thema.Lineage

	// EmbedPath determines the path to use in the generated go:embed variable
	// that is expected to contain the definition of the provided lineage.
	//
	// It is the responsibility of the caller to ensure that the file referenced
	// by EmbedPath contains the definition of the provided Lineage.
	EmbedPath string

	// CUEPath is the path to the lineage within the instance referred to by EmbedPath.
	// If non-empty, the generated binding will include a [cue.Value.LookupPath] call
	// prior to calling [thema.BindLineage].
	CUEPath cue.Path

	// FactoryNameSuffix determines whether the [thema.LineageFactory] or
	// [thema.ConvergentLineageFactory] implementation will be generated with
	// a title-cased lineage.name as a suffix.
	//
	// For example, for a lineage with lineage.name "foo", if this
	// property is false, the generated code will be:
	//  func Lineage(...) {...}
	// but if true, the following code will be generated:
	//  func LineageFoo(...) {...}
	FactoryNameSuffix bool

	// PrivateFactory determines whether the generated lineage factory will be
	// exported (`func Lineage` vs. `func doLineage`).
	//
	// A private factory may be preferable in cases where, for example, it is
	// desirable to ensure that certain [thema.BindOption] are always passed, or to
	// memoize the generated lineage factory function using the *thema.Runtime
	// parameter as a key.
	PrivateFactory bool

	// NoEmbed determines whether generation of an embed.FS containing the lineage's
	// declaring CUE files should be generated. If true, the embed.FS will NOT be
	// generated. Generated code will still reference the absent var, leaving
	// it to the developer to manually construct the var instead.
	NoEmbed bool

	// Assignee is an ast.Ident that determines the generic type parameter used
	// in the generated [thema.ConvergentLineageFactory]. If this parameter is nil,
	// a [thema.LineageFactory] is generated instead.
	Assignee *ast.Ident

	// TargetSchemaVersion determines the schema version that will be used in a call
	// to [thema.BindType], along with Assignee, in order to create a
	// thema.ConvergentLineage.
	//
	// This value is ignored if Assignee is nil.
	TargetSchemaVersion thema.SyntacticVersion

	// PackageName determines the name of the generated Go package. If empty, the
	// lowercase version of the Lineage.Name() is used.
	PackageName string
}

// GenerateLineageBinding generates Go code that makes a Thema lineage defined
// in a .cue file reliably available in Go via a [thema.LineageFactory] or
// [thema.ConvergentLineageFactory].
//
// The thema CLI implements the capabilities of this function via the `thema
// lineage gen go` subcommand. The CLI command should meet most use cases,
// though some may require the additional flexibility earned by writing a Go
// program that calls this function directly.
func GenerateLineageBinding(cfg *BindingConfig) ([]byte, error) {
	if cfg == nil || cfg.Lineage == nil || cfg.EmbedPath == "" {
		return nil, fmt.Errorf("cfg.Lineage and cfg.EmbedPath are required")
	}

	vars := bindingVars{
		Name:                cfg.Lineage.Name(),
		PackageName:         cfg.PackageName,
		GenEmbed:            !cfg.NoEmbed,
		EmbedPath:           cfg.EmbedPath,
		CUEPath:             cfg.CUEPath.String(),
		BaseFactoryFuncName: "Lineage",
		FactoryFuncName:     "Lineage",
		IsConvergent:        cfg.Assignee != nil,
		TargetSchemaVersion: cfg.TargetSchemaVersion,
	}

	if vars.PackageName == "" {
		vars.PackageName = strings.ToLower(cfg.Lineage.Name())
	}

	if cfg.FactoryNameSuffix {
		vars.BaseFactoryFuncName += strings.Title(vars.Name)
		vars.FactoryFuncName += strings.Title(vars.Name)
	}
	if vars.IsConvergent {
		vars.BaseFactoryFuncName = "base" + vars.BaseFactoryFuncName
		vars.Assignee = cfg.Assignee
		if strings.HasPrefix(cfg.Assignee.String(), "*") {
			vars.AssigneeInit = fmt.Sprintf("new(%s)", cfg.Assignee.String()[1:])
		} else {
			vars.AssigneeInit = fmt.Sprintf("%s{}", cfg.Assignee)
		}

		if cfg.PrivateFactory {
			vars.FactoryFuncName = "do" + vars.FactoryFuncName
		}
	} else if cfg.PrivateFactory {
		vars.BaseFactoryFuncName = "do" + vars.BaseFactoryFuncName
	}

	buf := new(bytes.Buffer)
	err := tmpls.Lookup("binding.tmpl").Execute(buf, vars)
	if err != nil {
		return nil, fmt.Errorf("error executing binding template: %w", err)
	}

	return postprocessGoFile(genGoFile{
		path: "binding_gen.go",
		in:   buf.Bytes(),
	})
}

type bindingVars struct {
	// Name of the lineage
	Name string
	// name to be used for the generated package
	PackageName string
	// Path to use in the go:embed directive
	EmbedPath string
	// Path to use as dir param to load.InstancesWithThema
	LoadPath string
	// Path within the cue file to look up to get lineage
	CUEPath string
	// Name for the base factory func, which is always generated and does basic
	// lineage binding.
	BaseFactoryFuncName string

	// generate the embedfs
	GenEmbed bool

	// Name of the factory func to generate. Must accommodate both FactoryNameSuffix
	// and PrivateFactory
	FactoryFuncName string

	// Whether we're generating a convergent lineage
	IsConvergent bool
	// The ident of the generic type parameter for a convergent lineage.
	Assignee *ast.Ident
	// The initializer for a Assignee
	AssigneeInit string

	TargetSchemaVersion thema.SyntacticVersion
}

type genGoFile struct {
	path  string
	apply astutil.ApplyFunc
	in    []byte
}

// func postprocessGoFile(cfg genGoFile) (*ast.File, error) {
func postprocessGoFile(cfg genGoFile) ([]byte, error) {
	fname := filepath.Base(cfg.path)
	buf := new(bytes.Buffer)
	fset := token.NewFileSet()
	gf, err := parser.ParseFile(fset, fname, string(cfg.in), parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("error parsing generated file: %w", err)
	}

	if cfg.apply != nil {
		astutil.Apply(gf, cfg.apply, nil)

		err = format.Node(buf, fset, gf)
		if err != nil {
			return nil, fmt.Errorf("error formatting Go AST: %w", err)
		}
	} else {
		buf = bytes.NewBuffer(cfg.in)
	}

	byt, err := imports.Process(fname, buf.Bytes(), nil)
	if err != nil {
		return nil, fmt.Errorf("goimports processing failed: %w", err)
	}
	return byt, nil
}
