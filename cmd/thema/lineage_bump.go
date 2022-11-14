package main

import (
	"fmt"
	"os"

	"cuelang.org/go/cue/ast"
	"github.com/grafana/thema"
	"github.com/grafana/thema/encoding/cue"
	tastutil "github.com/grafana/thema/internal/astutil"
	"github.com/spf13/cobra"
)

var lineageBumpCmd = &cobra.Command{
	Use:   "bump",
	Args:  cobra.MaximumNArgs(0),
	Short: "Add a new schema to an existing lineage",
	Long: `Add a new schema to an existing lineage.

Generate the necessary stubs to "bump" the latest schema version in an existing lineage by adding a new schema to it.
`,
}

type bumpCommand struct {
	maj      bool
	skipfill bool

	lla *lineageLoadArgs
}

func (bc *bumpCommand) setup(cmd *cobra.Command) {
	cmd.AddCommand(lineageBumpCmd)
	bc.lla = new(lineageLoadArgs)
	addLinPathVars(lineageBumpCmd, bc.lla)

	lineageBumpCmd.Flags().BoolVar(&bc.maj, "major", false, "Bump the major version (breaking change) instead of the minor version")
	lineageBumpCmd.Flags().BoolVar(&bc.maj, "no-fill", false, "Do not pre-fill the new schema with the prior schema")
	lineageBumpCmd.PreRunE = bc.lla.validateLineageInput
	lineageBumpCmd.Run = bc.run
}

func (bc *bumpCommand) run(cmd *cobra.Command, args []string) {
	if err := bc.do(cmd, args); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", err)
		os.Exit(1)
	}
}

func (bc *bumpCommand) do(cmd *cobra.Command, args []string) error {
	lv := thema.LatestVersion(bc.lla.dl.lin)
	lsch := thema.SchemaP(bc.lla.dl.lin, lv)
	// TODO UGH EVAL
	schlit := tastutil.Format(lsch.Underlying().Eval())

	var err error
	var nlin ast.Node
	if bc.maj {
		nlin = bc.lla.dl.lin.Underlying().Source()
		err = cue.InsertSchemaNodeAs(nlin, tastutil.ToExpr(schlit), thema.SV(lv[0]+1, 0))
		if err != nil {
			return err
		}
	} else {
		nlin, err = cue.Append(bc.lla.dl.lin, lsch.Underlying())
		if err != nil {
			return err
		}
	}

	b, err := tastutil.FmtNode(tastutil.ToExpr(nlin))
	if err != nil {
		return err
	}

	// TODO write back to subpath

	fmt.Fprint(cmd.OutOrStdout(), string(b))
	return nil
}
