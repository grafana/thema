package main

import (
	"bytes"
	"fmt"
	"os"

	upcue "cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/ast/astutil"
	"cuelang.org/go/cue/build"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/cue/parser"
	"cuelang.org/go/encoding/json"
	"cuelang.org/go/encoding/jsonschema"
	"cuelang.org/go/encoding/openapi"
	"cuelang.org/go/encoding/yaml"
	"github.com/grafana/thema"
	"github.com/grafana/thema/encoding/cue"
	tastutil "github.com/grafana/thema/internal/astutil"
	"github.com/grafana/thema/internal/util"
	"github.com/spf13/cobra"
)

var linCmd = &cobra.Command{
	Use:   "lineage <command>",
	Short: "Perform operations directly on lineages and schema",
	Long: `Perform operations directly on lineages and schema.

Create, modify, and validate Thema lineages. Generate representations
of Thema lineages and schema in other schema languages.
`,
}

type initCommand struct {
	name    string
	cuepath string
	pkgname string
	nopkg   bool
	srcpath string
	input   []byte

	err error
}

func setupLineageCommand(cmd *cobra.Command) {
	cmd.AddCommand(linCmd)
	ic := &initCommand{}
	ic.setup(linCmd)

	ac := &bumpCommand{}
	ac.setup(linCmd)
}

func (ic *initCommand) setup(cmd *cobra.Command) {
	cmd.AddCommand(initLineageCmd)
	initLineageCmd.PersistentFlags().StringVarP(&ic.name, "name", "n", "", "String for the #Lineage.name field")
	initLineageCmd.PersistentFlags().StringVar(&ic.pkgname, "package-name", "", "Name for generated package. If omitted, --name value is used")
	initLineageCmd.PersistentFlags().BoolVar(&ic.nopkg, "no-package", false, "Generate lineage without a package directive")

	// TODO uncomment these and fix broken toSubpath logic to support generating at a subpath
	// initLineageCmd.MarkFlagRequired("name")
	// initLineageCmd.PersistentFlags().StringVarP(&ic.cuepath, "cue-path", "p", "", "CUE expression for subpath at which lineage should be generated")

	initLineageCmd.AddCommand(initLineageEmptyCmd)
	initLineageEmptyCmd.Run = ic.run
	initLineageEmptyCmd.PreRunE = ic.processPackageArgs

	initLineageCmd.AddCommand(initLineageOpenAPICmd)
	initLineageOpenAPICmd.Flags().StringVar(&ic.srcpath, "src-subpath", "", "Schema path within the OpenAPI document. Default: whole document")
	initLineageOpenAPICmd.Run = ic.run
	initLineageOpenAPICmd.PreRunE = ic.processInput

	initLineageCmd.AddCommand(initLineageJSONSchemaCmd)
	initLineageJSONSchemaCmd.Flags().StringVar(&ic.srcpath, "src-subpath", "", "Schema path within the JSON Schema document (e.g. #/...) Default: whole document")
	initLineageJSONSchemaCmd.Run = ic.run
	initLineageJSONSchemaCmd.PreRunE = ic.processInput
}

func toSubpath(subpath string, f *ast.File) (*ast.File, error) {
	if subpath == "" {
		return f, nil
	}

	p := upcue.ParsePath(subpath)
	if p.Err() != nil {
		return nil, fmt.Errorf("invalid path provided for --cue-path: %w", p.Err())
	}

	err := astutil.Sanitize(f)
	if err != nil {
		return nil, fmt.Errorf("error while sanitizing generated file: %w", err)
	}

	// in := ctx.BuildFile(f)
	// if in.Err() != nil {
	// 	return nil, fmt.Errorf("error when building value: %w", in.Err())
	// }

	out := empt().FillPath(p, empt())
	if out.Err() != nil {
		return nil, fmt.Errorf("error in lineage when placed at requested --cue-path: %w", out.Err())
	}

	var nf *ast.File
	switch x := tastutil.Format(out).(type) {
	case *ast.File:
		nf = x
	case ast.Expr:
		nf, err = astutil.ToFile(x)
		if err != nil {
			return nil, fmt.Errorf("error converting expr to file: %w", err)
		}
	}

	var done bool
	astutil.Apply(nf, func(c astutil.Cursor) bool {
		if x, ok := c.Node().(*ast.StructLit); ok {
			if len(x.Elts) == 0 {
				x.Elts = f.Decls
				done = true
			}
		}
		return !done
	}, nil)

	f.Decls = nf.Decls
	return f, nil
}

func (ic *initCommand) run(cmd *cobra.Command, args []string) {
	switch cmd.CalledAs() {
	case "empty":
		ic.runEmpty(cmd, args)
	case "jsonschema":
		ic.runJSONSchema(cmd, args)
	case "openapi":
		ic.runOpenAPI(cmd, args)
	default:
		panic(fmt.Sprint("unrecognized command ", cmd.CalledAs()))
	}

	if ic.err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", ic.err)
		os.Exit(1)
	}
}

func (ic *initCommand) processPackageArgs(cmd *cobra.Command, args []string) error {
	if ic.name == "" {
		return fmt.Errorf("must provide a name for lineage via --name")
	}
	if ic.nopkg {
		if ic.pkgname != "" {
			return fmt.Errorf("cannot pass both --no-package and --package-name")
		}
	} else {
		if ic.pkgname == "" {
			ic.pkgname = ic.name
		}

		if !ast.IsValidIdent(ic.pkgname) {
			return fmt.Errorf("%q is not a valid package name", ic.pkgname)
		}
	}
	return nil
}

// process both openapi and json schema, abstract over stdin
func (ic *initCommand) processInput(cmd *cobra.Command, args []string) error {
	byt, err := pathOrStdin(args)
	if err != nil {
		return err
	}

	ic.input = byt
	return nil
}

var initLineageEmptyCmd = &cobra.Command{
	Use:   "empty",
	Args:  cobra.MaximumNArgs(0),
	Short: "Initialize with an empty schema",
	Long: `Initialize the lineage with an empty schema.

The name for the new lineage must be provided as a single argument.

The generated lineage is printed to stdout.
`,
}

func empt() upcue.Value {
	str := `
{
	// TODO (delete me - first schema goes here!)
} 
`
	expr, _ := parser.ParseExpr("empty", str, parser.ParseComments)
	return ctx.BuildExpr(expr)
}

func (ic *initCommand) runEmpty(cmd *cobra.Command, args []string) {
	str := `
{
	// TODO (delete me - first schema goes here!)
} 
`
	expr, _ := parser.ParseExpr("empty", str, parser.ParseComments)
	v := ctx.BuildExpr(expr)
	linf, err := cue.NewLineage(v, ic.name, ic.pkgname)
	if err != nil {
		ic.err = err
		return
	}

	// Have to re-insert because comments get lost somehow by NewLineage()
	err = cue.InsertSchemaNodeAs(linf, expr, thema.SV(0, 0))
	if err != nil {
		ic.err = err
		return
	}

	linf, err = toSubpath(ic.cuepath, linf)
	if err != nil {
		ic.err = err
		return
	}

	b, err := tastutil.FmtNode(linf)
	if err != nil {
		ic.err = err
		return
	}

	fmt.Fprint(cmd.OutOrStdout(), string(b))
}

var initLineageJSONSchemaCmd = &cobra.Command{
	Use:   "jsonschema <path> ",
	Args:  cobra.MaximumNArgs(1),
	Short: "Initialize with a JSON Schema",
	Long: `Initialize the lineage with one schema, derived from a JSON Schema document.

A JSON Schema document to be converted for the initial lineage schema must be given as an argument.

The generated lineage is printed to stdout.
`,
}

func (ic *initCommand) runJSONSchema(cmd *cobra.Command, args []string) {
	v := ctx.CompileBytes(ic.input)
	if v.Err() != nil {
		ic.err = v.Err()
		return
	}

	jcfg := &jsonschema.Config{
		Root: ic.srcpath,
	}

	f, err := jsonschema.Extract(v, jcfg)
	if err != nil {
		ic.err = err
		return
	}

	sch := ctx.BuildFile(f)
	// Remove attributes field
	astutil.Apply(f, func(c astutil.Cursor) bool {
		if _, ok := c.Node().(*ast.Attribute); ok {
			astutil.CopyComments(c.Node(), c.Parent().Node())
			c.Delete()
		}

		// Only descend into the file/top-level, not within fields
		_, is := c.Node().(*ast.File)
		return is
	}, nil)

	linf, err := cue.NewLineage(sch.Eval(), ic.name, ic.pkgname)
	if err != nil {
		ic.err = err
		return
	}

	linf, err = toSubpath(ic.cuepath, linf)
	if err != nil {
		ic.err = err
		return
	}

	b, err := tastutil.FmtNode(linf)
	if err != nil {
		ic.err = err
		return
	}

	fmt.Fprint(cmd.OutOrStdout(), string(b))
}

var initLineageOpenAPICmd = &cobra.Command{
	Use:   "openapi <path> ",
	Args:  cobra.MaximumNArgs(1),
	Short: "Initialize with an OpenAPI v3 schema",
	Long: `Initialize the lineage with one schema, derived from an OpenAPI v3 document.

An OpenAPI document to be converted for the initial lineage schema must be given as an argument.

The generated lineage is printed to stdout.
`,
}

// expects something else to have already gotten the input from either a file
// or stdin (as we do with pathOrStdin) and passed it as input param
func inputToFile(input []byte, args []string) (*ast.File, error) {
	if len(args) == 0 {
		args = append(args, "-")
	}

	cfg := &load.Config{
		Stdin: bytes.NewBuffer(input),
	}
	binsts := load.Instances(args, cfg)
	bf := binsts[0].OrphanedFiles[0]
	if bf == nil {
		return nil, fmt.Errorf("could not load input file")
	}

	var f *ast.File
	var err error
	switch bf.Encoding {
	case build.YAML:
		f, err = yaml.Extract("input", input)
	case build.JSON:
		expr, err := json.Extract("input", input)
		if err == nil {
			f = &ast.File{
				Decls: []ast.Decl{expr},
			}
		}
	default:
		err = fmt.Errorf("unsupported encoding: %s", bf.Encoding)
	}

	if err != nil {
		return nil, err
	}

	return f, nil
}

func (ic *initCommand) runOpenAPI(cmd *cobra.Command, args []string) {
	f, err := inputToFile(ic.input, args)
	if err != nil {
		ic.err = err
		return
	}

	rt := (*upcue.Runtime)(ctx)
	inst, err := rt.CompileFile(f)
	if err != nil {
		ic.err = err
		return
	}
	fo, err := openapi.Extract(inst, &openapi.Config{})
	if err != nil {
		ic.err = err
		return
	}
	// Remove info field
	var done bool
	astutil.Apply(fo, func(c astutil.Cursor) bool {
		if x, ok := c.Node().(*ast.Field); ok {
			n, _, _ := ast.LabelName(x.Label)
			if n == "info" {
				c.Delete()
				done = true
			}
		}

		if done {
			return false
		}
		// Only descend into the file/top-level, not within fields
		_, is := c.Node().(*ast.File)
		return is
	}, nil)

	sch := ctx.BuildFile(fo)
	if ic.srcpath != "" {
		p := upcue.ParsePath(ic.srcpath)
		if p.Err() != nil {
			ic.err = fmt.Errorf("value for --src-subpath is not a valid cue path expression: %w", p.Err())
			return
		}
		// Eval will do dereferencing for us as needed, but may have other unintended
		// side effects.
		sch = sch.LookupPath(p).Eval()
		if !sch.Exists() {
			ic.err = fmt.Errorf("path %q does not exist in converted schema", p.String())
			return
		}
	}

	linf, err := cue.NewLineage(sch, ic.name, ic.pkgname)
	if err != nil {
		ic.err = err
		return
	}

	linf, err = toSubpath(ic.cuepath, linf)
	if err != nil {
		ic.err = err
		return
	}

	b, err := tastutil.FmtNode(linf)
	if err != nil {
		ic.err = err
		return
	}

	fmt.Fprint(cmd.OutOrStdout(), string(b))
}

var initLineageCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a new lineage",
	Long: `Create a new lineage.

Each subcommand supports initializing the lineage from a different kind of input source.
`,
}

var lineageBumpCmd = &cobra.Command{
	Use:     "bump",
	PreRunE: validateLineageInput,
	Args:    cobra.MaximumNArgs(0),
	Short:   "Add a new schema to an existing lineage",
	Long: `Add a new schema to an existing lineage.

Generate the necessary stubs to "bump" the latest schema version in an existing lineage by adding a new schema to it.
`,
}

type bumpCommand struct {
	maj      bool
	skipfill bool
}

func (bc *bumpCommand) setup(cmd *cobra.Command) {
	cmd.AddCommand(lineageBumpCmd)
	cmd.PersistentFlags().StringVarP(&linfilepath, "lineage", "l", ".", "path to .cue file or directory containing lineage")
	cmd.MarkFlagRequired("lineage")
	// TODO use this once we support bumping subpaths
	// addLinPathVars(lineageBumpCmd)

	lineageBumpCmd.Flags().BoolVar(&bc.maj, "major", false, "Bump the major version (breaking change) instead of the minor version")
	lineageBumpCmd.Flags().BoolVar(&bc.maj, "no-fill", false, "Do not pre-fill the new schema with the prior schema")
	lineageBumpCmd.Run = bc.run
}

func (bc *bumpCommand) run(cmd *cobra.Command, args []string) {
	if err := bc.do(cmd, args); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", err)
		os.Exit(1)
	}
}

func (bc *bumpCommand) do(cmd *cobra.Command, args []string) error {
	lv := thema.LatestVersion(lin)
	lsch := thema.SchemaP(lin, lv)
	// TODO UGH EVAL
	schv := lsch.UnwrapCUE().Eval()

	var err error
	var nlin ast.Node
	if bc.maj {
		p := "rand" + util.RandSeq(10)
		mschv := schv.FillPath(upcue.MakePath(upcue.Str(p)), empt())
		nlin, err = cue.Append(lin, mschv)
		if err != nil {
			return err
		}

		var done bool
		astutil.Apply(nlin, func(c astutil.Cursor) bool {
			if x, ok := c.Node().(*ast.Field); ok {
				n, _, _ := ast.LabelName(x.Label)
				if n == p {
					c.Delete()
					done = true
				}
			}

			return !done
		}, nil)
	} else {
		nlin, err = cue.Append(lin, schv)
		if err != nil {
			return err
		}
	}

	b, err := tastutil.FmtNode(nlin)
	if err != nil {
		return err
	}

	// TODO support writing back to stdout, when input came from stdin
	// FIXME don't rely on a brittle hack to write back. Also, respect perms
	return os.WriteFile(linbinst.BuildFiles[0].Filename, b, 0644)
}
