package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/grafana/thema"
	terrors "github.com/grafana/thema/errors"
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
	setupGenCmd(rootCmd)

	// Stop cobra from being so "helpful"
	for _, cmd := range allCmds {
		cmd.DisableFlagsInUseLine = true
		cmd.SilenceUsage = true
		cmd.CompletionOptions = cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		}
	}

	// srv commands
	// TODO
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// List of all commands, for batching stuff
var allCmds = []*cobra.Command{rootCmd, genCmd, srvCmd, httpCmd, dataCmd, translateCmd, validateCmd, validateAnyCmd}

var rootCmd = &cobra.Command{
	Use:   "thema <command>",
	Short: "A tool for putting Thema lineages to work",
	Long: `A tool for putting Thema lineages to work.

This program offers several kinds of behavior for working with Thema:

* Validating and inspecting of written lineages.
* Given a valid lineage, provides basic Thema operations (validate, translate,
  [de]hydrate) on some input data.
* Run an HTTP server that exposes those basic Thema operations to the network. (TODO)
* Provides scaffolding for writing lineages, lenses, and schema. (TODO)
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
		if errors.Is(err, terrors.ErrValueNotALineage) && strings.Contains(err.Error(), "instance root") {
			return fmt.Errorf("%w\nDid you forget to pass a CUE path with -p?", err)
		}
		return err
	}
	return nil
}

func validateVersionInput(cmd *cobra.Command, args []string) error {
	return dovinput(cmd, args, false)
}

func validateVersionInputOptional(cmd *cobra.Command, args []string) error {
	return dovinput(cmd, args, true)
}

func dovinput(cmd *cobra.Command, args []string, opt bool) error {
	if lin == nil {
		err := validateLineageInput(cmd, args)
		if err != nil {
			return err
		}
	}
	if verstr == "" {
		if opt {
			return nil
		}
		return errors.New("must pass a schema version with -v")
	}

	synv, err := verstr.synv()
	if err != nil {
		return err
	}

	sch, err = lin.Schema(synv)
	return err
}
