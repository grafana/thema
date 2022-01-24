package main

import (
	"fmt"
	"os"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/grafana/thema"
	"github.com/spf13/cobra"
)

var ctx = cuecontext.New()
var lib = thema.NewLibrary(ctx)

// linfilepath is the filesystem path to the file (or directory) containing
// the lineage
var linfilepath string
var lincuepath string

var lin thema.Lineage

// String argument of a version - "to" with translate and "version" with
var verstr synvstring

type synvstring string

func (s synvstring) synv() (thema.SyntacticVersion, error) {
	return thema.ParseSyntacticVersion(string(s))
}

// encoding of the input data.
var encoding string

// bytes read from stdin
var inbytes []byte

// input data, CUEified
var datval cue.Value

// quiet mode
var quiet bool

// schema to use
var sch thema.Schema

func main() {
	rootCmd.PersistentFlags().StringVarP(&linfilepath, "lineage", "l", ".", "path to .cue file or directory containing lineage")
	rootCmd.MarkFlagRequired("lineage")
	rootCmd.PersistentFlags().StringVarP(&lincuepath, "path", "p", "", "CUE expression for path to the lineage object within file, if not root")

	setupDataCommand(rootCmd)

	// srv commands
	// TODO
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "thema-cli",
	Short: "A tool for putting Thema lineages to work",
	Long: `A tool for putting Thema lineages to work.

This program offers several kinds of behavior for working with Thema:

* Validating and inspecting of written lineages.
* Given a valid lineage, provides basic Thema operations (validate, translate,
  [de]hydrate) on some input data.
* Run an HTTP server that exposes those basic Thema operations to the network.
* Provides scaffolding for writing lineages, lenses, and schema. (TODO)
`,
}

var linCmd = &cobra.Command{
	Use:   "lineage",
	Short: "Inspect lineages declared in .cue files",
	Long: `Inspect lineages declared in .cue files.
`,
}

var srvCmd = &cobra.Command{
	Use:   "srv",
	Short: "Run a server that offers Thema operations over the network",
	Long: `Run a server that offers Thema operations over the network.
`,
}

var httpCmd = &cobra.Command{
	Use:   "http",
	Short: "Start an HTTP(S) server",
	Long: `Start an HTTP(S) server.
`,
}

type cobraefunc func(cmd *cobra.Command, args []string) error

func mergeCobraefuncs(f ...cobraefunc) cobraefunc {
	return func(cmd *cobra.Command, args []string) error {
		for _, fun := range f {
			if err := fun(cmd, args); err != nil {
				return err
			}
		}

		return nil
	}
}

func validateLineageInput(cmd *cobra.Command, args []string) error {
	var err error
	lin, err = lineageFromPaths(lib, linfilepath, lincuepath)
	if err != nil {
		return err
	}
	return nil
}

func validateVersionInput(cmd *cobra.Command, args []string) error {
	if lin == nil {
		err := validateLineageInput(cmd, args)
		if err != nil {
			return err
		}
	}
	synv, err := verstr.synv()
	if err != nil {
		return err
	}

	_, err = lin.Schema(synv)
	if err != nil {
		return fmt.Errorf("lineage %q does not contain a schema with version %s", lin.Name(), synv)
	}
	return nil
}
