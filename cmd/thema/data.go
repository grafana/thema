package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"github.com/grafana/thema"
	"github.com/grafana/thema/kernel"
	"github.com/spf13/cobra"
)

func setupDataCommand(cmd *cobra.Command) {
	cmd.AddCommand(dataCmd)
	addLinPathVars(dataCmd)

	dataCmd.AddCommand(validateCmd)
	validateCmd.Flags().StringVarP((*string)(&verstr), "version", "v", "", "schema syntactic version to validate data against")
	validateCmd.MarkFlagRequired("version")
	validateCmd.Flags().StringVarP(&encoding, "encoding", "e", "", "input data encoding. Autodetected by default, but can be constrained to \"json\" or \"yaml\".")
	validateCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "emit no output, exit status only")

	dataCmd.AddCommand(validateAnyCmd)
	validateAnyCmd.Flags().StringVarP((*string)(&verstr), "version", "v", "", "schema syntactic version to validate data against")
	validateAnyCmd.Flags().StringVarP(&encoding, "encoding", "e", "", "input data encoding. Autodetected by default, but can be constrained to \"json\" or \"yaml\".")
	validateAnyCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "emit no output, exit status only")

	dataCmd.AddCommand(translateCmd)
	translateCmd.Flags().StringVarP((*string)(&verstr), "to", "v", "", "schema version to translate input data to")
	translateCmd.MarkFlagRequired("to")
	translateCmd.Flags().StringVarP(&encoding, "encoding", "e", "", "input data encoding. Autodetected by default, but can be constrained to \"json\" or \"yaml\".")

	dataCmd.AddCommand(hydrateCmd)
	hydrateCmd.Flags().StringVarP((*string)(&verstr), "version", "v", "", "schema syntactic version to validate data against")
	hydrateCmd.Flags().StringVarP(&encoding, "encoding", "e", "", "input data encoding. Autodetected by default, but can be constrained to \"json\" or \"yaml\".")

	dataCmd.AddCommand(dehydrateCmd)
	dehydrateCmd.Flags().StringVarP((*string)(&verstr), "version", "v", "", "schema syntactic version to validate data against")
	dehydrateCmd.Flags().StringVarP(&encoding, "encoding", "e", "", "input data encoding. Autodetected by default, but can be constrained to \"json\" or \"yaml\".")
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
supported; the correct encoding is inferred. Only one object instance may be
validated per command invocation.
`

var validateCmd = &cobra.Command{
	Use:   "validate -l <lineage-fs-path> -v <synver> [-p <cue-path>] [-q] [-e <encoding>] [<data-fs-path>]",
	Short: "Validate some input data against a particular Thema schema",
	Long: `Validate some input data against a particular Thema schema.
` + dataReuseText + `
Success outputs nothing and exits 0. Failure outputs the validation problem
(unless quieted) and exits 1.
`,
	Args:              cobra.MaximumNArgs(1),
	PersistentPreRunE: mergeCobraefuncs(validateLineageInput, validateVersionInput, validateDataInput),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !datval.Exists() {
			panic("datval does not exist")
		}

		_, err := sch.Validate(datval)
		if err != nil {
			return err
		}
		return nil
	},
}

var validateAnyCmd = &cobra.Command{
	Use:   "validate-any -l <lineage-fs-path> [-p <cue-path>] [-v <synver>] [-q] [-e <encoding>] [<data-fs-path>]",
	Short: "Search a lineage for a schema that validates some input data",
	Long: `Search a lineage for a schema that validates some input data.
` + dataReuseText + `
Success outputs the schema version that matched and exits 0. Failure exits 1 and
outputs nothing.

If --version is passed, that version is checked first. If validation fails
against all schemas in the lineage, the error against the --version schema will
be printed.
`,
	PersistentPreRunE: mergeCobraefuncs(validateLineageInput, validateVersionInputOptional, validateDataInput),
	Args:              cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !datval.Exists() {
			panic("datval does not exist")
		}

		var reterr error
		if sch != nil {
			_, reterr = sch.Validate(datval)
			if reterr == nil {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\n", sch.Version())
				return nil
			}
		}
		inst := lin.ValidateAny(datval)
		if inst != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "%s\n", inst.Schema().Version())
			return nil
		}

		if reterr != nil {
			return reterr
		}
		// Empty error should cause exit 1, but no output (or maybe just newline)
		return errors.New("")
	},
}

var translateCmd = &cobra.Command{
	Use:   "translate -l <lineage-fs-path> [-p <cue-path>] [--to <synver>] [-e <encoding>] [<data-fs-path>]",
	Short: "Translate some valid input data from one schema to another",
	Long: `Translate some valid input data from one schema to another.
` + dataReuseText + `
Success outputs the translated object instance, the version the input validated
against, any emitted lacuna, and exits 0. Failure exits 1 with an informative
error.

Note that Thema's invariants (once finalized) guarantee that failures can only
arise during data input decoding or validation, never during translation.
`,
	PersistentPreRunE: mergeCobraefuncs(validateLineageInput, validateVersionInput, validateDataInput),
	Args:              cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !datval.Exists() {
			panic("datval does not exist")
		}

		inst := lin.ValidateAny(datval)
		if inst == nil {
			return errors.New("input data is not valid for any schema in lineage")
		}

		// Prior validations checked that the schema version exists in the lineage
		tinst, lac := inst.Translate(sch.Version())
		if err := validateTranslationResult(tinst, lac); err != nil {
			return err
		}

		// TODO support non-JSON output
		r := translationResult{
			From:    inst.Schema().Version().String(),
			To:      tinst.Schema().Version().String(),
			Result:  tinst.UnwrapCUE(),
			Lacunas: lac,
		}

		byt, err := json.MarshalIndent(r, "", "  ")
		if err != nil {
			return fmt.Errorf("error marshaling translation result to JSON: %w", err)
		}
		buf := bytes.NewBuffer(byt)
		_, err = io.Copy(cmd.OutOrStdout(), buf)

		return err
	},
}

type translationResult struct {
	From    string                   `json:"from"`
	To      string                   `json:"to,omitempty"`
	Result  cue.Value                `json:"result"`
	Lacunas thema.TranslationLacunas `json:"lacunas"`
}

var hydrateCmd = &cobra.Command{
	Use:   "hydrate -l <lineage-fs-path> [-p <cue-path>] [-e <encoding>] [-v <synver>] [<data-fs-path>] ",
	Short: "Fill some valid input data with any schema-specified defaults",
	Long: `Fill some valid input data with any schema-specified defaults.
` + dataReuseText + `
Success outputs the input object, but fully hydrated with schema-specified
default values, if any. Input formatting (e.g. indent spaces) and/or object
key ordering are not maintained.

If a syntactic version is not provided (-v), the input data will be checked
for validity against all schemas in the lineage.
`,
	PersistentPreRunE: mergeCobraefuncs(validateLineageInput, validateVersionInputOptional, validateDataInput),
	Args:              cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !datval.Exists() {
			panic("datval does not exist")
		}

		inst := lin.ValidateAny(datval)
		if inst == nil {
			return errors.New("input data is not valid for any schema in lineage")
		}

		// TODO support non-JSON output
		byt, err := json.MarshalIndent(inst.Hydrate().UnwrapCUE(), "", "  ")
		if err != nil {
			fmt.Printf("%+v %#v\n", inst.Hydrate().UnwrapCUE(), inst.Hydrate().UnwrapCUE())
			return fmt.Errorf("error marshaling hydrated object to JSON: %w", err)
		}
		buf := bytes.NewBuffer(byt)
		_, err = io.Copy(cmd.OutOrStdout(), buf)

		return err
	},
}

var dehydrateCmd = &cobra.Command{
	Use:   "dehydrate -l <lineage-fs-path> [-p <cue-path>] [-e <encoding>] [-v <synver>] [<data-fs-path>] ",
	Short: "Remove all schema-specified defaults from some valid input data",
	Long: `Remove all schema-specified defaults from some valid input data.
` + dataReuseText + `
Success outputs the input data object, but fully dehydrated, with all of its values
that are implied by defaults specified in its validating schema removed. Input
formatting (e.g. indent spaces) and/or object key ordering are not maintained.

If a syntactic version is not provided (-v), the input data will be checked
for validity against all schemas in the lineage.
`,
	PersistentPreRunE: mergeCobraefuncs(validateLineageInput, validateVersionInputOptional, validateDataInput),
	Args:              cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !datval.Exists() {
			panic("datval does not exist")
		}

		var inst *thema.Instance
		var err error
		if sch != nil {
			inst, err = sch.Validate(datval)
			if err != nil {
				return err
			}
		} else {
			inst = lin.ValidateAny(datval)
			if inst == nil {
				return errors.New("input data is not valid for any schema in lineage")
			}
		}

		// TODO support non-JSON output
		byt, err := json.MarshalIndent(inst.Dehydrate().UnwrapCUE(), "", "  ")
		if err != nil {
			return fmt.Errorf("error marshaling dehydrated object to JSON: %w", err)
		}
		buf := bytes.NewBuffer(byt)
		_, err = io.Copy(cmd.OutOrStdout(), buf)

		return err
	},
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

		byt, err = ioutil.ReadAll(os.Stdin)
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

		byt, err = ioutil.ReadAll(f)
		if err != nil {
			return nil, fmt.Errorf("error reading from input file %q: %w", args[0], err)
		}

	default:
		return nil, errors.New("too many args: either provide path to input or pass input on stdin")
	}

	return byt, nil
}

func validateDataInput(cmd *cobra.Command, args []string) error {
	var ext string

	byt, err := pathOrStdin(args)
	if err != nil {
		return err
	}
	if len(byt) > 0 && len(inbytes) == 0 {
		inbytes = byt
	}

	if len(args) == 1 {
		switch filepath.Ext(args[0]) {
		case ".json", ".ldjson":
			ext = "json"
		case ".yaml", ".yml":
			ext = "yaml"
		}
	}

	jd := kernel.NewJSONDecoder("stdin")
	yd := kernel.NewYAMLDecoder("stdin")

	switch encoding {
	case "":
		// Figure it out; try JSON first
		datval, err = jd(lib.UnwrapCUE().Context(), inbytes)
		if err != nil {
			encoding = "json"
			break
		}
		// Nope, try yaml
		datval, err = yd(lib.UnwrapCUE().Context(), inbytes)
		if err != nil {
			encoding = "yaml"
			break
		}
		// Double nope
		return errors.New("unrecognized encoding of input data")

	case "json":
		if ext != "json" {
			return fmt.Errorf("JSON input encoding specified, but file extension is %s", ext)
		}

		datval, err = jd(lib.UnwrapCUE().Context(), inbytes)
		if err != nil {
			return err
		}
	case "yaml":
		if ext != "yaml" {
			return fmt.Errorf("YAML input encoding specified, but file extension is %s", ext)
		}

		datval, err = yd(lib.UnwrapCUE().Context(), inbytes)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown input encoding %q requested", encoding)
	}

	return nil
}

// Everything here should become unnecessary once Thema's key invariants are in
// place
func validateTranslationResult(tinst *thema.Instance, lac thema.TranslationLacunas) error {
	if tinst == nil {
		panic("unreachable, thema.Translate() should never return a nil instance")
	}

	raw := tinst.UnwrapCUE()
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
