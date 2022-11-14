package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"github.com/grafana/thema"
	"github.com/grafana/thema/vmux"
	"github.com/spf13/cobra"
)

type dataCommand struct {
	format  string
	quiet   bool
	inbytes []byte

	datval cue.Value

	lla *lineageLoadArgs

	inst *thema.Instance

	err error
}

func setupDataCommand(cmd *cobra.Command) {
	cmd.AddCommand(dataCmd)
	dc := new(dataCommand)
	dc.setup(dataCmd)
}

func (dc *dataCommand) setup(cmd *cobra.Command) {
	dc.lla = new(lineageLoadArgs)
	addLinPathVars(cmd, dc.lla)
	dataCmd.MarkPersistentFlagRequired("lineage")

	dataCmd.AddCommand(validateCmd)
	validateCmd.Flags().StringVarP(&dc.lla.verstr, "version", "v", "", "schema syntactic version to validate data against. defaults to latest")
	validateCmd.Flags().StringVarP(&dc.format, "format", "e", "", "input data format. Autodetected by default, but can be constrained to \"json\" or \"yaml\".")
	validateCmd.Flags().BoolVarP(&dc.quiet, "quiet", "q", false, "emit no output, exit status only")
	validateCmd.PersistentPreRunE = mergeCobraefuncs(dc.lla.validateLineageInput, dc.lla.validateVersionInputOptional, dc.validateDataInput)
	validateCmd.RunE = dc.runValidate

	dataCmd.AddCommand(validateAnyCmd)
	validateAnyCmd.Flags().StringVarP(&dc.lla.verstr, "version", "v", "", "schema syntactic version to validate data against")
	validateAnyCmd.Flags().StringVarP(&dc.format, "format", "e", "", "input data format. Autodetected by default, but can be constrained to \"json\" or \"yaml\".")
	validateAnyCmd.Flags().BoolVarP(&dc.quiet, "quiet", "q", false, "emit no output, exit status only")
	validateAnyCmd.PersistentPreRunE = mergeCobraefuncs(dc.lla.validateLineageInput, dc.lla.validateVersionInputOptional, dc.validateDataInput)
	validateAnyCmd.RunE = dc.runValidateAny

	dataCmd.AddCommand(translateCmd)
	translateCmd.Flags().StringVarP(&dc.lla.verstr, "to", "v", "", "schema version to translate input data to")
	translateCmd.MarkFlagRequired("to")
	translateCmd.Flags().StringVarP(&dc.format, "format", "e", "", "input data format. Autodetected by default, but can be constrained to \"json\" or \"yaml\".")
	translateCmd.PersistentPreRunE = mergeCobraefuncs(dc.lla.validateLineageInput, dc.lla.validateVersionInput, dc.validateDataInput)
	translateCmd.RunE = dc.runTranslate

	dataCmd.AddCommand(hydrateCmd)
	hydrateCmd.Flags().StringVarP(&dc.lla.verstr, "version", "v", "", "schema syntactic version to validate data against")
	hydrateCmd.Flags().StringVarP(&dc.format, "format", "e", "", "input data format. Autodetected by default, but can be constrained to \"json\" or \"yaml\".")
	hydrateCmd.PersistentPreRunE = mergeCobraefuncs(dc.lla.validateLineageInput, dc.lla.validateVersionInputOptional, dc.validateDataInput)
	hydrateCmd.RunE = dc.runHydrate

	dataCmd.AddCommand(dehydrateCmd)
	dehydrateCmd.Flags().StringVarP(&dc.lla.verstr, "version", "v", "", "schema syntactic version to validate data against")
	dehydrateCmd.Flags().StringVarP(&dc.format, "format", "e", "", "input data format. Autodetected by default, but can be constrained to \"json\" or \"yaml\".")
	dehydrateCmd.PersistentPreRunE = mergeCobraefuncs(dc.lla.validateLineageInput, dc.lla.validateVersionInputOptional, dc.validateDataInput)
	dehydrateCmd.RunE = dc.runDehydrate
}

func (dc *dataCommand) run(cmd *cobra.Command, args []string) {
	switch cmd.CalledAs() {
	case "validate":
	case "validate-any":
	case "translate":
	case "hydrate":
	case "dehydrate":
	}
}

var dataCmd = &cobra.Command{
	Use:   "data <command>",
	Short: "Perform Thema operations on some input data",
	Long: `Perform Thema operations on some input data.
`,
}

var dataReuseText = `
A filesystem path to a Thema lineage must be provided. It may be relative or
absolute. Lineages are necessarily validated prior to validation of the input
data. All data operations are performed in the context of the provided lineage.

Data may be provided on stdin, or by passing a single path to a file as an
argument. Stdin is ignored if a path is provided. JSON and YAML inputs are
supported; the correct format is inferred. Only one object instance may be
validated per command invocation.
`

var validateCmd = &cobra.Command{
	Use:   "validate -l <lineage-fs-path> -v <synver> [-p <cue-path>] [-q] [-e <format>] [<data-fs-path>]",
	Short: "Validate some input data against a particular Thema schema",
	Long: `Validate some input data against a particular Thema schema.
` + dataReuseText + `
Success outputs nothing and exits 0. Failure outputs the validation problem
(unless quieted) and exits 1.
`,
	Args: cobra.MaximumNArgs(1),
}

func (dc *dataCommand) runValidate(cmd *cobra.Command, args []string) error {
	if !dc.datval.Exists() {
		panic("datval does not exist")
	}

	_, err := dc.lla.dl.sch.Validate(dc.datval)
	if err != nil {
		return err
	}
	return nil
}

var validateAnyCmd = &cobra.Command{
	Use:   "validate-any -l <lineage-fs-path> [-p <cue-path>] [-v <synver>] [-q] [-e <format>] [<data-fs-path>]",
	Short: "Search a lineage for a schema that validates some input data",
	Long: `Search a lineage for a schema that validates some input data.
` + dataReuseText + `
Success outputs the schema version that matched and exits 0. Failure exits 1 and
outputs nothing.

If --version is passed, that version is checked first. If validation fails
against all schemas in the lineage, the error against the --version schema will
be printed.
`,
	Args: cobra.MaximumNArgs(1),
}

func (dc *dataCommand) runValidateAny(cmd *cobra.Command, args []string) error {
	if !dc.datval.Exists() {
		panic("datval does not exist")
	}

	var reterr error
	if dc.lla.dl.sch != nil {
		_, reterr = dc.lla.dl.sch.Validate(dc.datval)
		if reterr == nil {
			fmt.Fprintf(cmd.OutOrStdout(), "%s\n", dc.lla.dl.sch.Version())
			return nil
		}
	}
	inst := dc.lla.dl.lin.ValidateAny(dc.datval)
	if inst != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", dc.inst.Schema().Version())
		return nil
	}

	if reterr != nil {
		return reterr
	}
	// Empty error should cause exit 1, but no output (or maybe just newline)
	return errors.New("")
}

var translateCmd = &cobra.Command{
	Use:   "translate -l <lineage-fs-path> [-p <cue-path>] [--to <synver>] [-e <format>] [<data-fs-path>]",
	Short: "Translate some valid input data from one schema to another",
	Long: `Translate some valid input data from one schema to another.
` + dataReuseText + `
Success outputs the translated object instance, the version the input validated
against, any emitted lacuna, and exits 0. Failure exits 1 with an informative
error.

Note that Thema's invariants (once finalized) guarantee that failures can only
arise during data input decoding or validation, never during translation.
`,
	Args: cobra.MaximumNArgs(1),
}

func (dc *dataCommand) runTranslate(cmd *cobra.Command, args []string) error {
	if !dc.datval.Exists() {
		panic("datval does not exist")
	}

	inst := dc.lla.dl.lin.ValidateAny(dc.datval)
	if inst == nil {
		return errors.New("input data is not valid for any schema in lineage")
	}

	// Prior validations checked that the schema version exists in the lineage
	tinst, lac := inst.Translate(dc.lla.dl.sch.Version())
	if err := dc.validateTranslationResult(tinst, lac); err != nil {
		return err
	}

	// TODO support non-JSON output
	r := translationResult{
		From:    inst.Schema().Version().String(),
		To:      tinst.Schema().Version().String(),
		Result:  tinst.Underlying(),
		Lacunas: lac,
	}

	byt, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling translation result to JSON: %w", err)
	}
	buf := bytes.NewBuffer(byt)
	_, err = io.Copy(cmd.OutOrStdout(), buf)

	return err
}

type translationResult struct {
	From    string                   `json:"from"`
	To      string                   `json:"to,omitempty"`
	Result  cue.Value                `json:"result"`
	Lacunas thema.TranslationLacunas `json:"lacunas"`
}

var hydrateCmd = &cobra.Command{
	Use:   "hydrate -l <lineage-fs-path> [-p <cue-path>] [-e <format>] [-v <synver>] [<data-fs-path>] ",
	Short: "Fill some valid input data with any schema-specified defaults",
	Long: `Fill some valid input data with any schema-specified defaults.
` + dataReuseText + `
Success outputs the input object, but fully hydrated with schema-specified
default values, if any. Input formatting (e.g. indent spaces) and/or object
key ordering are not maintained.

If a syntactic version is not provided (-v), the input data will be checked
for validity against all schemas in the lineage.
`,
	Args: cobra.MaximumNArgs(1),
}

func (dc *dataCommand) runHydrate(cmd *cobra.Command, args []string) error {
	if !dc.datval.Exists() {
		panic("datval does not exist")
	}

	inst := dc.lla.dl.lin.ValidateAny(dc.datval)
	if inst == nil {
		return errors.New("input data is not valid for any schema in lineage")
	}

	// TODO support non-JSON output
	byt, err := json.MarshalIndent(inst.Hydrate().Underlying(), "", "  ")
	if err != nil {
		// fmt.Printf("%+v %#v\n", inst.Hydrate().Underlying(), inst.Hydrate().Underlying())
		return fmt.Errorf("error marshaling hydrated object to JSON: %w", err)
	}
	buf := bytes.NewBuffer(byt)
	_, err = io.Copy(cmd.OutOrStdout(), buf)

	return err
}

var dehydrateCmd = &cobra.Command{
	Use:   "dehydrate -l <lineage-fs-path> [-p <cue-path>] [-e <format>] [-v <synver>] [<data-fs-path>] ",
	Short: "Remove all schema-specified defaults from some valid input data",
	Long: `Remove all schema-specified defaults from some valid input data.
` + dataReuseText + `
Success outputs the input data object, but fully dehydrated, with all of its values
that are implied by defaults specified in its validating schema removed. Input
formatting (e.g. indent spaces) and/or object key ordering are not maintained.

If a syntactic version is not provided (-v), the input data will be checked
for validity against all schemas in the lineage.
`,
	Args: cobra.MaximumNArgs(1),
}

func (dc *dataCommand) runDehydrate(cmd *cobra.Command, args []string) error {
	if !dc.datval.Exists() {
		panic("datval does not exist")
	}

	inst := dc.lla.dl.lin.ValidateAny(dc.datval)
	if inst == nil {
		return errors.New("input data is not valid for any schema in lineage")
	}

	// TODO support non-JSON output
	byt, err := json.MarshalIndent(dc.inst.Hydrate().Underlying(), "", "  ")
	if err != nil {
		fmt.Printf("%+v %#v\n", dc.inst.Hydrate().Underlying(), dc.inst.Hydrate().Underlying())
		return fmt.Errorf("error marshaling hydrated object to JSON: %w", err)
	}
	buf := bytes.NewBuffer(byt)
	_, err = io.Copy(cmd.OutOrStdout(), buf)

	return err
}

func pathOrStdin(args []string) ([]byte, error) {
	var byt []byte
	switch len(args) {
	case 0:
		fi, err := os.Stdin.Stat()
		if err != nil {
			panic(err)
		}
		if fi.Mode()&os.ModeNamedPipe == 0 {
			return nil, errors.New("no path provided and nothing on stdin")
		}

		byt, err = io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("error reading from stdin: %w", err)
		}
		return byt, nil
	case 1:
		fi, err := os.Stat(args[0])
		if err != nil {
			return nil, fmt.Errorf("failed to stat path %q: %w", args[0], err)
		}
		if fi.IsDir() {
			return nil, fmt.Errorf("%s is a directory", args[0])
		}

		f, err := os.Open(args[0])
		if err != nil {
			return nil, fmt.Errorf("could not open provided path: %w", err)
		}
		defer f.Close() // nolint: errcheck

		byt, err = io.ReadAll(f)
		if err != nil {
			return nil, fmt.Errorf("error reading from input file %q: %w", args[0], err)
		}

	default:
		return nil, errors.New("too many args: either provide path to input or pass input on stdin")
	}

	return byt, nil
}

func (dc *dataCommand) validateDataInput(cmd *cobra.Command, args []string) error {
	var ext string

	byt, err := pathOrStdin(args)
	if err != nil {
		return err
	}
	if len(byt) > 0 && len(dc.inbytes) == 0 {
		dc.inbytes = byt
	}

	if len(args) == 1 {
		switch filepath.Ext(args[0]) {
		case ".json", ".ldjson":
			ext = "json"
		case ".yaml", ".yml":
			ext = "yaml"
		}
	}

	jd := vmux.NewJSONCodec("stdin")
	yd := vmux.NewYAMLCodec("stdin")

	switch dc.format {
	case "":
		// Figure it out; try JSON first
		dc.datval, err = jd.Decode(rt.Underlying().Context(), dc.inbytes)
		if err == nil {
			dc.format = "json"
			break
		}
		// Nope, try yaml
		dc.datval, err = yd.Decode(rt.Underlying().Context(), dc.inbytes)
		if err == nil {
			dc.format = "yaml"
			break
		}
		// Double nope
		return errors.New("unrecognized format of input data")

	case "json":
		if ext != "json" {
			return fmt.Errorf("JSON input format specified, but file extension is %s", ext)
		}

		dc.datval, err = jd.Decode(rt.Underlying().Context(), dc.inbytes)
		if err != nil {
			return err
		}
	case "yaml":
		if ext != "yaml" {
			return fmt.Errorf("YAML input format specified, but file extension is %s", ext)
		}

		dc.datval, err = yd.Decode(rt.Underlying().Context(), dc.inbytes)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown input format %q requested", dc.format)
	}

	return nil
}

// Everything here should become unnecessary once Thema's key invariants are in
// place
func (dc *dataCommand) validateTranslationResult(tinst *thema.Instance, lac thema.TranslationLacunas) error {
	if tinst == nil {
		panic("unreachable, thema.Translate() should never return a nil instance")
	}

	raw := tinst.Underlying()
	if !raw.Exists() {
		return errors.New("should be unreachable - result should at least always exist")
	}

	if raw.Err() != nil {
		return fmt.Errorf("translated value has errors, should be unreachable: %w", raw.Err())
	}

	if !raw.IsConcrete() {
		return fmt.Errorf("translated value is not concrete (TODO print non-concrete fields)")
	}

	return nil
}
