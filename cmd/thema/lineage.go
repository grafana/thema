package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var genCmd = &cobra.Command{
	Use:   "gen <command>",
	Short: "Generate code from and for Thema lineages",
	Long: `Generate code from and for Thema lineages.
`,
}

func setupGenCmd(cmd *cobra.Command) {
	cmd.AddCommand(genCmd)

	genCmd.AddCommand(initLineageCmd)
}

var initLineageCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate an empty lineage, ready to fill in with schema.",
	Long: `Generate an empty lineage, ready to fill in with schema.

The name for the new lineage must be provided as a single argument. The empty
lineage is printed to stdout.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("must exactly one argument, the name of the lineage")
		}

		fmt.Printf(`package %s

import "github.com/grafana/thema"

thema.#Lineage
name: "%s"
seqs: [
    {
        schemas: [
            { // 0.0
				// First schema (v0.0) goes here! (delete me)
            },
        ]
    },
]
`, args[0], args[0])

		return nil
	},
}
