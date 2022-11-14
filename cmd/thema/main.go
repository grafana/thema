package main

import (
	"os"

	"cuelang.org/go/cue/cuecontext"
	"github.com/grafana/thema"
	"github.com/spf13/cobra"
)

var ctx = cuecontext.New()
var rt = thema.NewRuntime(ctx)

func main() {
	setupDataCommand(rootCmd)
	setupLineageCommand(rootCmd)

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

func addLinPathVars(cmd *cobra.Command, lla *lineageLoadArgs) {
	cmd.PersistentFlags().StringVarP(&lla.inputLinFilePath, "lineage", "l", ".", "path to .cue file or directory containing lineage")
	cmd.MarkFlagRequired("lineage")
	cmd.PersistentFlags().StringVarP(&lla.lincuepath, "path", "p", "", "CUE expression for path to the lineage object within file, if not root")
}

// List of all commands, for batching stuff
var allCmds = []*cobra.Command{
	rootCmd,
	srvCmd,
	httpCmd,
	dataCmd,
	translateCmd,
	validateCmd,
	validateAnyCmd,
	linCmd,
	initLineageCmd,
	initLineageEmptyCmd,
	initLineageOpenAPICmd,
	initLineageJSONSchemaCmd,
	lineageBumpCmd,
	genLineageCmd,
	genTSTypesLineageCmd,
	genGoBindingsLineageCmd,
	genGoTypesLineageCmd,
	genOapiLineageCmd,
	genJschLineageCmd,
}

var rootCmd = &cobra.Command{
	Use:   "thema <command>",
	Short: "A tool for putting Thema lineages to work",
	Long: `A tool for putting Thema lineages to work.

This program offers several kinds of behavior for working with Thema:

* Validating and inspecting of written lineages.
* Given a valid lineage, provides basic Thema operations (validate, translate,
  [de]hydrate) on some input data.
* Run an HTTP server that exposes basic Thema operations to the network. (TODO)
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
