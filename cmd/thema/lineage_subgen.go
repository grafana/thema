package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/pkg/encoding/yaml"
	"github.com/grafana/thema"
	"github.com/grafana/thema/encoding/gocode"
	"github.com/grafana/thema/encoding/jsonschema"
	"github.com/grafana/thema/encoding/openapi"
	"github.com/spf13/cobra"
)

type genCommand struct {
	group bool
	// include suffix in generated lineage factory
	suffix bool
	// lineage factory should be private
	private bool
	// don't generate the embed.FS
	noembed bool
	// don't generate the themaFSFunc impl
	nofsfunc bool

	// write to stdout instead of generator-specific file
	stdout bool

	quiet bool

	// input file format (yaml, json, etc.)
	format string

	lin thema.Lineage
	sch thema.Schema
	// go type to bind to
	bindtype string
	// go package name to target
	pkgname string
	// path for embedding
	epath string

	lla *lineageLoadArgs
}

func (gc *genCommand) setup(cmd *cobra.Command) {
	cmd.AddCommand(genLineageCmd)
	gc.lla = new(lineageLoadArgs)
	addLinPathVars(genLineageCmd, gc.lla)
	genLineageCmd.PersistentPreRunE = mergeCobraefuncs(gc.lla.validateLineageInput, gc.lla.validateVersionInputOptional)

	gop := genOapiLineageCmd
	genLineageCmd.AddCommand(gop)
	gop.Flags().StringVarP(&gc.lla.verstr, "version", "v", "", "schema syntactic version to generate. Defaults to latest")
	gop.Flags().StringVarP(&gc.format, "format", "f", "yaml", "output format. \"json\" or \"yaml\".")
	gop.Run = gc.run

	gj := genJschLineageCmd
	genLineageCmd.AddCommand(gj)
	gj.Flags().StringVarP(&gc.lla.verstr, "version", "v", "", "schema syntactic version to generate. Defaults to latest")
	gj.Flags().StringVarP(&gc.format, "format", "f", "json", "output format. \"json\" or \"yaml\".")
	gj.Run = gc.run

	ggt := genGoTypesLineageCmd
	genLineageCmd.AddCommand(ggt)
	ggt.Use = "gotypes -l <path> [-p <cue-path>] [-v <synver>] [--pkgname <name>] [--stdout]"
	ggt.Flags().StringVarP(&gc.lla.verstr, "version", "v", "", "schema syntactic version to generate. Defaults to latest")
	ggt.Flags().StringVar(&gc.pkgname, "pkgname", "", "Name for generated Go package. Defaults to lowercase lineage name")
	ggt.Flags().BoolVar(&gc.noembed, "stdout", false, "Write to stdout instead of '<lineage.name>_types_gen.go'")
	ggt.Flags().BoolVarP(&gc.quiet, "quiet", "q", false, "Do not print generated filename")
	ggt.Run = gc.run

	ggb := genGoBindingsLineageCmd
	genLineageCmd.AddCommand(ggb)
	ggb.Use = "gobindings -l <path> [-p <cue-path>] [--bindtype <name>] [--bindversion <synver>] [--suffix] [--private] [--no-embed]"
	ggb.Flags().StringVar(&gc.bindtype, "bindtype", "", "Generate a ConvergentLineage that binds a lineage's schema to this Go type")
	ggb.Flags().StringVarP(&gc.lla.verstr, "version", "v", "", "Only meaningful with --bindtype. Bind to this schema version. Defaults to latest")
	ggb.Flags().StringVar(&gc.pkgname, "pkgname", "", "Name for generated Go package. Defaults to lowercase lineage name")
	ggb.Flags().BoolVar(&gc.suffix, "suffix", false, "Generate the lineage factory as 'Lineage<TitleCaseName>()' instead of 'Lineage()'")
	ggb.Flags().BoolVar(&gc.private, "private", false, "Generate the lineage factory as an unexported (lowercase) func.")
	ggb.Flags().BoolVar(&gc.noembed, "no-embed", false, "Do not generate an embed.FS, allowing it to be handwritten")
	ggb.Flags().BoolVar(&gc.nofsfunc, "no-fs-func", false, "Do not generate the func that returns fs.FS for loading")
	ggb.Flags().BoolVar(&gc.stdout, "stdout", false, "Write to stdout instead of '<lineage.name>_types_gen.go'")
	ggb.Flags().BoolVarP(&gc.quiet, "quiet", "q", false, "Do not print generated filename")
	ggb.Run = gc.run

	// TODO
	// genLineageCmd.AddCommand(genTSTypesLineageCmd)
	// genTSTypesLineageCmd.Flags().StringVarP((*string)(&verstr), "version", "v", "", "schema syntactic version to generate. Defaults to latest")
}

func (gc *genCommand) run(cmd *cobra.Command, args []string) {
	// TODO encapsulate these properly
	gc.lin = gc.lla.dl.lin
	gc.sch = gc.lla.dl.sch
	gc.epath = gc.lla.inputLinFilePath

	if fi, err := os.Stat(gc.lla.inputLinFilePath); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", err)
		os.Exit(1)
	} else if fi.IsDir() {
		if gc.epath == "." {
			gc.epath = "*.cue"
		} else {
			gc.epath += "/*.cue"
		}
	}

	var err error
	switch cmd.CalledAs() {
	case "jsonschema":
		err = gc.runJSONSchema(cmd, args)
	case "openapi":
		err = gc.runOpenAPI(cmd, args)
	case "gotypes":
		err = gc.runGoTypes(cmd, args)
	case "gobindings":
		err = gc.runGoBindings(cmd, args)
	case "tstypes":
		err = gc.runTSTypes(cmd, args)
	default:
		panic(fmt.Sprint("unrecognized command ", cmd.CalledAs()))
	}

	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", err)
		os.Exit(1)
	}
}

var genLineageCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate code from a lineage",
	Long: `Generate code from a lineage and its schemas.

Each subcommand supports generating code for a different language target.

Note that the controls offered by each subcommand are intentionally simplified.
But, each subcommand is implemented as a thin layer atop the packages in
github.com/grafana/thema/format/*. If the CLI lacks the fine-grained control
you require, it is recommended to write your own code generator using those packages.
`,
}

var genOapiLineageCmd = &cobra.Command{
	Use:   "openapi",
	Short: "Generate OpenAPI from a lineage",
	Long: `Generate OpenAPI from a lineage.

Generate an OpenAPI document containing a OpenAPI schema components representing a
single schema in a lineage.

Only OpenAPI 3.0 is currently supported.
`,
}

func (gc *genCommand) runOpenAPI(cmd *cobra.Command, args []string) error {
	f, err := openapi.GenerateSchema(gc.sch, nil)
	if err != nil {
		return err
	}

	var str string
	switch gc.format {
	case "json":
		var b []byte
		b, err = rt.Context().BuildFile(f).MarshalJSON()
		if b != nil {
			nb := new(bytes.Buffer)
			err = json.Indent(nb, b, "", "  ")
			str = nb.String()
		}
	case "yaml", "yml":
		str, err = yaml.Marshal(rt.Context().BuildFile(f))
	default:
		fmt.Fprintf(cmd.ErrOrStderr(), `unrecognized output format %q - must choose "yaml" or "json"`, gc.format)
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), str)
	return nil
}

var genJschLineageCmd = &cobra.Command{
	Use:   "jsonschema",
	Short: "Generate JSON Schema from a lineage",
	Long: `Generate JSON Schema from a lineage.

Generate a JSON Schema (Draft 4) document representing a single schema in a lineage,
and print it to stdout.
`,
}

func (gc *genCommand) runJSONSchema(cmd *cobra.Command, args []string) error {
	f, err := jsonschema.GenerateSchema(gc.sch)
	if err != nil {
		return err
	}

	var str string
	switch gc.format {
	case "json":
		var b []byte
		b, err = rt.Context().BuildFile(f).MarshalJSON()
		if b != nil {
			nb := new(bytes.Buffer)
			err = json.Indent(nb, b, "", "  ")
			str = nb.String()
		}
	case "yaml", "yml":
		str, err = yaml.Marshal(rt.Context().BuildFile(f))
	default:
		fmt.Fprintf(cmd.ErrOrStderr(), `unrecognized output format %q - must choose "yaml" or "json"`, gc.format)
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), str)
	return nil
}

var genGoTypesLineageCmd = &cobra.Command{
	Short: "Generate Go types from a lineage",
	Long: `Generate Go types from a lineage.

Generate Go types that correspond to a single schema in a lineage.

By default, the generated types are written to the same directory that contains the lineage,
in a file named $NAME_types_gen.go, where $NAME is the lowercase string value of
the lineage's name. Pass --stdout to send generated code to stdout instead.

This command internally generates OpenAPI, then uses github.com/deepmap/oapi-codegen
to produce Go types. Future parameters may expose different implementations, but this
form of output will be preserved by default.
`,
}

func (gc *genCommand) runGoTypes(cmd *cobra.Command, args []string) error {
	buf := new(bytes.Buffer)
	if gc.stdout {
		fmt.Fprint(buf, goheader)
	} else {
		fmt.Fprintf(buf, fmt.Sprintf(goheaderp, gc.epath))
	}
	b, err := gocode.GenerateTypesOpenAPI(gc.sch, &gocode.TypeConfigOpenAPI{
		PackageName: gc.pkgname,
	})
	if err != nil {
		return err
	}
	buf.Write(b)
	if gc.stdout {
		fmt.Fprint(cmd.OutOrStdout(), buf.String())
		return nil
	}

	path := gc.lla.absInput
	if !gc.lla.pathIsDir {
		path = filepath.Dir(path)
	}
	path = filepath.Join(path, fmt.Sprintf("%s_types_gen.go", strings.ToLower(gc.lin.Name())))
	return os.WriteFile(path, buf.Bytes(), 0644)
}

var genGoBindingsLineageCmd = &cobra.Command{
	Short: "Generate Go bindings for a lineage",
	Long: `Generate Go bindings for a lineage.

Generate Go bindings to a Thema lineage. These bindings provide access to the
key Thema operations (see "thema help data") in a running Go program, for that
particular lineage.

If --bindtype is omitted, a basic LineageFactory is generated.
If --bindtype is provided, a ConvergentLineageFactory is also generated, layered
on top of the basic factory. The type itself is not generated by this command. For
that, run "thema lineage gen gotypes".

Output is written to the same directory that contains the lineage, in a file named
$NAME_bindings_gen.go, where $NAME is the lowercase string value of the lineage's
name.

The correctness of the generated embed.FS and CUE loading behavior is
sensitive to location of the generated file relative to a cue.mod directory,
if any, and in any parent directory. As such, if --stdout is passed, the
command cannot offer any correctness guarantees. The generator no longer
produces a themaFSFor$NAME() implementation, but still calls the function,
passing responsibility to the user to hand-write their own implementation.

LineageFactory: https://pkg.go.dev/github.com/grafana/thema#LineageFactory
ConvergentLineageFactory: https://pkg.go.dev/github.com/grafana/thema#ConvergentLineageFactory
`,
}

func (gc *genCommand) runGoBindings(cmd *cobra.Command, args []string) error {
	epath := gc.epath
	if gc.stdout || gc.noembed {
		epath = ""
	}

	cfg := &gocode.BindingConfig{
		Lineage: gc.lin,
		// TODO figure out what to put here if a dir was provided
		EmbedPath:           epath,
		NoThemaFSImpl:       gc.nofsfunc,
		FactoryNameSuffix:   gc.suffix,
		PrivateFactory:      gc.private,
		TargetSchemaVersion: gc.sch.Version(),
		PackageName:         gc.pkgname,
	}
	if gc.bindtype != "" {
		cfg.Assignee = ast.NewIdent(gc.bindtype)
	}

	f, err := gocode.GenerateLineageBinding(cfg)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	if gc.stdout {
		fmt.Fprint(buf, goheader)
		buf.Write(f)
		fmt.Fprint(cmd.OutOrStdout(), buf.String())
		return nil
	}

	fmt.Fprintf(buf, fmt.Sprintf(goheaderp, gc.epath))
	buf.Write(f)

	path := gc.lla.absInput
	if !gc.lla.pathIsDir {
		path = filepath.Dir(path)
	}
	path = filepath.Join(path, fmt.Sprintf("%s_bindings_gen.go", strings.ToLower(gc.lin.Name())))
	return os.WriteFile(path, buf.Bytes(), 0644)
}

var genTSTypesLineageCmd = &cobra.Command{
	Use:   "tstypes",
	Short: "Generate TypeScript types from a lineage",
	Long: `Generate TypeScript types from a lineage.

Generate a JSON Schema document representing a single schema in a lineage.
`,
}

func (gc *genCommand) runTSTypes(cmd *cobra.Command, args []string) error {
	panic("TODO")
}

var goheader = `// THIS FILE IS GENERATED. EDITING IS FUTILE.
//
// Generated by "thema lineage gen"

`

var goheaderp = `// THIS FILE IS GENERATED. EDITING IS FUTILE.
//
// Generated by "thema lineage gen" from lineage defined in %s

`
