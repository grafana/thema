package main

import (
	"fmt"
	"os"

	"cuelang.org/go/cue"
	cuenc "github.com/grafana/thema/encoding/cue"
	tastutil "github.com/grafana/thema/internal/astutil"
	"github.com/spf13/cobra"
)

var lineageFixCmd = &cobra.Command{
	Use:   "fix",
	Args:  cobra.MaximumNArgs(0),
	Short: "Rewrite legacy lineage definitions to new standards",
	Long: `Rewrite legacy lineage definitions to current standards.
`,
}

type fixCommand struct {
	lla *lineageLoadArgs
}

func (fc *fixCommand) setup(cmd *cobra.Command) {
	cmd.AddCommand(lineageFixCmd)
	fc.lla = new(lineageLoadArgs)

	lineageFixCmd.PersistentFlags().StringVarP(&fc.lla.inputLinFilePath, "lineage", "l", ".", "path to .cue file or package containing legacy lineage to rewrite")
	lineageFixCmd.MarkFlagRequired("lineage")
	lineageFixCmd.PersistentFlags().StringVarP(&fc.lla.lincuepath, "path", "p", "", "CUE expression for path to the lineage object within file, if not root")
	fc.lla.skipBindLineage = true

	lineageFixCmd.PreRunE = fc.lla.validateLineageInput
	lineageFixCmd.Run = fc.run
}

func (fc *fixCommand) run(cmd *cobra.Command, args []string) {
	if err := fc.do(cmd, args); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", err)
		os.Exit(1)
	}
}

func (fc *fixCommand) do(cmd *cobra.Command, args []string) error {
	f, err := cuenc.RewriteLegacyLineage(ctx.BuildInstance(fc.lla.dl.binst), cue.ParsePath(fc.lla.lincuepath))
	if err != nil {
		return err
	}

	b, err := tastutil.FmtNode(f)
	if err != nil {
		return err
	}

	return os.WriteFile(f.Filename, b, 0666)
}
