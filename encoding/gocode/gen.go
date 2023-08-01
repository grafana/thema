package gocode

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
	"text/template"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/pkg/encoding/yaml"
	"github.com/dave/dst"
	"github.com/dave/dst/dstutil"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/grafana/thema"
	"github.com/grafana/thema/encoding/openapi"
	"github.com/grafana/thema/internal/deepmap/oapi-codegen/pkg/codegen"
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

	// ApplyFuncs is a slice of AST manipulation funcs that will be executed against
	// the generated Go file prior to running it through goimports. For each slice
	// element, [dstutil.Apply] is called with the element as the "pre" parameter.
	ApplyFuncs []dstutil.ApplyFunc

	// IgnoreDiscoveredImports causes the generator not to fail with an error in the
	// event that goimports adds additional import statements. (The default behavior
	// is to fail because adding imports entails a search, which can slow down
	// codegen by multiple orders of magnitude. Succeeding silently but slowly is a bad
	// default behavior when the fix is usually quite easy.)
	IgnoreDiscoveredImports bool

	// NoOptionalPointers causes optional schema fields to be represented as normal
	// Go struct fields, rather than as pointers. For example, a `foo?: string`
	// field is usually converted to `Foo *string`, but is instead converted `Foo
	// string` when NoOptionalPointers is true.
	NoOptionalPointers bool

	// UseGoDeclInComments sets the name of the fields and structs at the beginning of each comment.
	UseGoDeclInComments bool

	// Config is passed through to the Thema OpenAPI encoder, [openapi.GenerateSchema].
	Config *openapi.Config
}

// GenerateTypesOpenAPI generates native Go code corresponding to the provided Schema.
func GenerateTypesOpenAPI(sch thema.Schema, cfg *TypeConfigOpenAPI) ([]byte, error) {
	if cfg == nil {
		cfg = new(TypeConfigOpenAPI)
	}

	depointer := depointerizer(false)
	if cfg.NoOptionalPointers {
		depointer = depointerizer(true)
	}

	applyFuncs := []dstutil.ApplyFunc{depointer, fixRawData(), fixUnderscoreInTypeName()}
	if !cfg.UseGoDeclInComments {
		applyFuncs = append(applyFuncs, fixTODOComments())
	}
	applyFuncs = append(applyFuncs, cfg.ApplyFuncs...)

	f, err := openapi.GenerateSchema(sch, cfg.Config)
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

	ccfg := codegen.Configuration{
		PackageName: cfg.PackageName,
		Compatibility: codegen.CompatibilityOptions{
			AlwaysPrefixEnumValues: true,
		},
		Generate: codegen.GenerateOptions{
			Models: true,
		},
		OutputOptions: codegen.OutputOptions{
			SkipPrune: true,
			// SkipFmt:   true, // we should be able to skip fmt, but dst's parser panics on nested structs when we don't
			UserTemplates: map[string]string{
				"imports.tmpl": importstmpl,
			},
		},
		ImportMapping:     nil,
		AdditionalImports: nil,
	}

	gostr, err := codegen.Generate(oT, ccfg)
	if err != nil {
		return nil, fmt.Errorf("openapi generation failed: %w", err)
	}

	return PostprocessGoFile(GenGoFile{
		Path:     fmt.Sprintf("%s_type_gen.go", sch.Lineage().Name()),
		Appliers: applyFuncs,
		In:       []byte(gostr),
		ErrIfAdd: !cfg.IgnoreDiscoveredImports,
	})
}

// Almost all of the below imports are eliminated by dst transformers and calls
// to goimports - but if they're not present in the template, then the internal
// call to goimports that oapi-codegen makes will trigger a search for them,
// which can slow down codegen by orders of magnitude.
var importstmpl = `package {{ .PackageName }}

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"os"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/deepmap/oapi-codegen/pkg/runtime"
	openapi_types "github.com/deepmap/oapi-codegen/pkg/types"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
	"github.com/labstack/echo/v4"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/mux"
)
`

// options
// - next to file, no cue.mod parent, therefore need one --- create one dynamically, load with dir "."
//   - embed - create it, *.cue
//   - fsfunc - yes, and wrapfs
//   - loaderfunc - yes, load with ""
//   - INPUTS: embed path, module name
// - next to file, cue.mod exists also next to file --- call func, load with dir "."
//   - embed - create it, *.cue +cue.mod
//   - fsfunc - yes, and wrapfs
//   - loaderfunc - yes, load with "."
//   - INPUTS: embed path, module name
// - next to file, cue.mod exists in parent dir --- call func, load with dir <specified>
//   - embed - yes, file.cue
//   - fsfunc - skip
//   - loaderfunc - yes, load with specified dir
//   - INPUTS: embed path, dir
// - not next to file (stdout) --- NFI, just call the func, load with dir <specified>
//   - embed - skip
//   - fsfunc - skip
//   - loaderfunc - skip

// BindingConfig governs the behavior of [GenerateLineageBinding].
type BindingConfig struct {
	// EmbedPath is the path that will appear in the generated go:embed variable
	// that is expected to contain the definition of the provided lineage.
	//
	// It is the responsibility of the caller to ensure that the file referenced
	// by EmbedPath contains the definition of the provided Lineage.
	//
	// If empty, no embed will be generated.
	//
	// EmbedPath is passed unmodified to the binding template, so multiple paths may be
	// provided, separated by spaces, per the //go:embed spec.
	EmbedPath string

	// LoadDir is the path that will be passed to when calling
	// [load.InstanceWithThema] within the generated loadInstanceFor$NAME func.
	//
	// If empty, the func will not be generated.
	LoadDir string

	// CueModName is the name of the CUE module that will be used when calling
	// [load.AsModFS] within the generated themaFSFor$NAME func.
	//
	// If empty, the func will not be generated.
	CueModName string

	// TitleName is the title-case name of the lineage. If empty, this will default
	// to the result of [strings.Title] called on lineage.name.
	TitleName string

	// CuePath is the path to the lineage within the instance referred to by EmbedPath.
	// If non-empty, the generated binding will include a [cue.Value.LookupPath] call
	// prior to calling [thema.BindLineage].
	CuePath cue.Path

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

	// Assignee is an dst.Ident that determines the generic type parameter used
	// in the generated [thema.ConvergentLineageFactory]. If this parameter is nil,
	// a [thema.LineageFactory] is generated instead.
	Assignee *dst.Ident

	// TargetSchemaVersion determines the schema version that will be used in a call
	// to [thema.BindType], along with Assignee, in order to create a
	// thema.ConvergentLineage.
	//
	// This value is ignored if Assignee is nil.
	TargetSchemaVersion thema.SyntacticVersion

	// PackageName determines the name of the generated Go package. If empty, the
	// lowercase version of the Lineage.Name() is used.
	PackageName string

	// ApplyFuncs is a slice of AST manipulation funcs that will be executed against
	// the generated Go file prior to running it through goimports. For each slice
	// element, [dstutil.Apply] is called with the element as the "pre" parameter.
	ApplyFuncs []dstutil.ApplyFunc

	// IgnoreDiscoveredImports causes the generator not to fail with an error in the
	// event that goimports adds additional import statements. (The default behavior
	// is to fail because adding imports entails a search, which can slow down
	// codegen by multiple orders of magnitude. Succeeding silently but slowly is a bad
	// default behavior when the fix is usually quite easy.)
	IgnoreDiscoveredImports bool
}

// generate scenarios:
// - output is stdout - non-optionally disable generating a cueFS embed var; generate a func instead
// - there is no cue.mod parent, in which case we dynamically construct one (using what module path?)
// - there is one, and it's in the output dir - include it directly in cueFS embed
// - there is one, and it's in the parent of output dir - construct a cueFS from a call to a prefixer

// GenerateLineageBinding generates Go code that makes a Thema lineage defined
// in a .cue file reliably available in Go via a [thema.LineageFactory] or
// [thema.ConvergentLineageFactory].
//
// The thema CLI implements the capabilities of this function via the `thema
// lineage gen go` subcommand. The CLI command should meet most use cases,
// though some may require the additional flexibility earned by writing a Go
// program that calls this function directly.
func GenerateLineageBinding(lin thema.Lineage, cfg *BindingConfig) ([]byte, error) {
	if cfg == nil {
		cfg = new(BindingConfig)
	}

	vars := bindingVars{
		Name:        lin.Name(),
		PackageName: cfg.PackageName,
		// certain optional generated elements are generated contingent on
		// config input strings being non-empty
		GenEmbed:            cfg.EmbedPath != "",
		GenFSFunc:           cfg.CueModName != "",
		GenLoaderFunc:       cfg.LoadDir != "",
		CueModName:          cfg.CueModName,
		LoadDir:             cfg.LoadDir,
		EmbedPath:           cfg.EmbedPath,
		CUEPath:             fmt.Sprintf("%q", cfg.CuePath.String()),
		BaseFactoryFuncName: "Lineage",
		FactoryFuncName:     "Lineage",
		IsConvergent:        cfg.Assignee != nil,
		TargetSchemaVersion: cfg.TargetSchemaVersion,
	}

	if vars.PackageName == "" {
		vars.PackageName = strings.ToLower(lin.Name())
	}

	if cfg.FactoryNameSuffix {
		if cfg.TitleName == "" {
			vars.BaseFactoryFuncName += strings.Title(vars.Name)
			vars.FactoryFuncName += strings.Title(vars.Name)
		} else {
			vars.BaseFactoryFuncName += cfg.TitleName
			vars.FactoryFuncName += cfg.TitleName
		}
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

	return PostprocessGoFile(GenGoFile{
		Path:     fmt.Sprintf("%s_binding_gen.go", strings.ToLower(lin.Name())),
		Appliers: cfg.ApplyFuncs,
		In:       buf.Bytes(),
		ErrIfAdd: !cfg.IgnoreDiscoveredImports,
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
	LoadDir string
	// Path within the cue file to look up to get lineage
	CUEPath string

	// name of the CUE module, used in generated call to load.AsModFS
	CueModName string

	// Name for the base factory func, which is always generated and does basic
	// lineage binding.
	BaseFactoryFuncName string

	// generate the embedfs
	GenEmbed bool
	// generate the fs func impl
	GenFSFunc bool
	// generate the build loader func impl
	GenLoaderFunc bool

	// Name of the factory func to generate. Must accommodate both FactoryNameSuffix
	// and PrivateFactory
	FactoryFuncName string

	// Whether we're generating a convergent lineage
	IsConvergent bool
	// The ident of the generic type parameter for a convergent lineage.
	Assignee *dst.Ident
	// The initializer for a Assignee
	AssigneeInit string

	TargetSchemaVersion thema.SyntacticVersion
}
