package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"os"

	"cuelang.org/go/pkg/encoding/yaml"
	"github.com/grafana/thema"
	"github.com/grafana/thema/encoding/jsonschema"
	"github.com/grafana/thema/encoding/openapi"
	"github.com/grafana/thema/encoding/tgo"
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
	lin     thema.Lineage
	sch     thema.Schema
	// go type to bind to
	bindtype string
	// go package name to target
	pkgname string
	// path for embedding
	epath string
}

func (gc *genCommand) setup(cmd *cobra.Command) {
	cmd.AddCommand(genLineageCmd)
	addLinPathVars(genLineageCmd)

	// genLineageCmd.PersistentFlags().BoolVar((*bool)(&gc.group), "group", false, "whether the schema is a 'group', and therefore only child items should be generated")

	genLineageCmd.AddCommand(genOapiLineageCmd)
	genOapiLineageCmd.Flags().StringVarP((*string)(&verstr), "version", "v", "", "schema syntactic version to generate. Defaults to latest")
	genOapiLineageCmd.Flags().StringVarP(&encoding, "format", "f", "yaml", "output format. \"json\" or \"yaml\".")
	genOapiLineageCmd.Run = gc.run

	genLineageCmd.AddCommand(genJschLineageCmd)
	genJschLineageCmd.Flags().StringVarP((*string)(&verstr), "version", "v", "", "schema syntactic version to generate. Defaults to latest")
	genJschLineageCmd.Flags().StringVarP(&encoding, "format", "f", "json", "output format. \"json\" or \"yaml\".")
	genJschLineageCmd.Run = gc.run

	genLineageCmd.AddCommand(genGoTypesLineageCmd)
	genGoTypesLineageCmd.Flags().StringVarP((*string)(&verstr), "version", "v", "", "schema syntactic version to generate. Defaults to latest")
	genGoTypesLineageCmd.Flags().StringVar(&gc.pkgname, "pkgname", "", "Name for generated Go package. Defaults to lowercase lineage name")
	genGoTypesLineageCmd.Run = gc.run

	genLineageCmd.AddCommand(genGoBindingsLineageCmd)
	genGoBindingsLineageCmd.Use = "gobindings -l <path> [-p <cue-path>] [--bindtype <name>] [--suffix] [--private] [--bindversion <synver>]"
	genGoBindingsLineageCmd.Flags().StringVar(&gc.bindtype, "bindtype", "", "Generate a ConvergentLineage that binds a lineage's schema to this Go type")
	genGoBindingsLineageCmd.Flags().StringVarP((*string)(&verstr), "version", "v", "", "Only meaningful with --bindtype. Bind to this schema version. Defaults to latest")
	genGoBindingsLineageCmd.Flags().StringVar(&gc.pkgname, "pkgname", "", "Name for generated Go package. Defaults to lowercase lineage name")
	genGoBindingsLineageCmd.Flags().BoolVar(&gc.suffix, "suffix", false, "Generate the lineage factory as 'Lineage<TitleName>()' instead of 'Lineage()'")
	genGoBindingsLineageCmd.Flags().BoolVar(&gc.private, "private", false, "Generate the lineage factory as an unexported (lowercase) func.")
	genGoBindingsLineageCmd.Flags().BoolVar(&gc.noembed, "no-embed", false, "Do not generate an embed.FS, allowing it to be handwritten")
	genGoBindingsLineageCmd.Run = gc.run

	// TODO
	// genLineageCmd.AddCommand(genTSTypesLineageCmd)
	// genTSTypesLineageCmd.Flags().StringVarP((*string)(&verstr), "version", "v", "", "schema syntactic version to generate. Defaults to latest")
}

func (gc *genCommand) run(cmd *cobra.Command, args []string) {
	// TODO encapsulate these properly
	gc.lin = lin
	gc.sch = sch
	if gc.sch == nil {
		gc.sch = thema.SchemaP(gc.lin, thema.LatestVersion(gc.lin))
	}
	gc.epath = linfilepath
	if fi, err := os.Stat(linfilepath); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", err)
		os.Exit(1)
	} else if fi.IsDir() {
		gc.epath += "/*.cue"
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
github.com/grafana/thema/encoding/*. If the CLI lacks the fine-grained control
you require, it is recommended to write your own code generator using those packages.
`,
	PersistentPreRunE: mergeCobraefuncs(validateLineageInput, validateVersionInputOptional),
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
	switch encoding {
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
		fmt.Fprintf(cmd.ErrOrStderr(), `unrecognized output format %q - must choose "yaml" or "json"`, encoding)
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

Generate a JSON Schema (Draft 4) document representing a single schema in a lineage.
`,
}

func (gc *genCommand) runJSONSchema(cmd *cobra.Command, args []string) error {
	f, err := jsonschema.GenerateSchema(gc.sch)
	if err != nil {
		return err
	}

	var str string
	switch encoding {
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
		fmt.Fprintf(cmd.ErrOrStderr(), `unrecognized output format %q - must choose "yaml" or "json"`, encoding)
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), str)
	return nil
}

var genGoTypesLineageCmd = &cobra.Command{
	Use:   "gotypes",
	Short: "Generate Go types from a lineage",
	Long: `Generate Go types from a lineage.

Generate Go types that correspond to a single schema in a lineage.
`,
}

func (gc *genCommand) runGoTypes(cmd *cobra.Command, args []string) error {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, fmt.Sprintf(goheader, gc.epath))
	b, err := tgo.GenerateTypesOpenAPI(gc.sch, &tgo.TypeConfigOpenAPI{
		PackageName: gc.pkgname,
	})
	if err != nil {
		return err
	}
	buf.Write(b)

	fmt.Fprint(cmd.OutOrStdout(), buf.String())
	return nil
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

LineageFactory: https://pkg.go.dev/github.com/grafana/thema#LineageFactory
ConvergentLineageFactory: https://pkg.go.dev/github.com/grafana/thema#ConvergentLineageFactory
`,
}

func (gc *genCommand) runGoBindings(cmd *cobra.Command, args []string) error {
	cfg := &tgo.BindingConfig{
		Lineage: gc.lin,
		// TODO figure out what to put here if a dir was provided
		EmbedPath:           gc.epath,
		NoEmbed:             gc.noembed,
		FactoryNameSuffix:   gc.suffix,
		PrivateFactory:      gc.private,
		TargetSchemaVersion: gc.sch.Version(),
		PackageName:         gc.pkgname,
	}
	if gc.bindtype != "" {
		cfg.Assignee = ast.NewIdent(gc.bindtype)
	}

	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, fmt.Sprintf(goheader, gc.epath))
	f, err := tgo.GenerateLineageBinding(cfg)
	if err != nil {
		return err
	}
	buf.Write(f)

	fmt.Fprint(cmd.OutOrStdout(), buf.String())
	return nil
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

var goheader = `// This file is autogenerated. DO NOT EDIT.
//
// Generated by "thema lineage gen" from lineage defined in %s

`
