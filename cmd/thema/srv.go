package main

import "github.com/spf13/cobra"

func setupSrvCommand(cmd *cobra.Command) {
	cmd.AddCommand(srvCmd)

}

var srvCmd = &cobra.Command{
	Use:   "srv <command>",
	Short: "Run a server that offers Thema operations over the network",
	Long: `Run a server that offers Thema operations over the network.

	TODO not yet implemented
`,
}

var httpCmd = &cobra.Command{
	Use:    "http",
	Hidden: true,
	Short:  "Start an HTTP(S) server",
	Long: `Start an HTTP(S) server.
`,
}
