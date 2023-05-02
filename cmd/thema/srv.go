package main

import (
	"github.com/grafana/thema/server"
	"github.com/spf13/cobra"
)

type srvCommand struct {
	port string
}

func setupSrvCommand(cmd *cobra.Command) {
	cmd.AddCommand(srvCmd)
	sc := new(srvCommand)
	sc.setup(srvCmd)
}

func (sc *srvCommand) setup(cmd *cobra.Command) {
	srvCmd.AddCommand(httpCmd)
	httpCmd.Flags().StringVarP(&sc.port, "port", "p", "8080", "port of the HTTP server")
	httpCmd.RunE = sc.runHTTPServer
}

func (sc *srvCommand) runHTTPServer(cmd *cobra.Command, args []string) error {
	if err := server.Init(sc.port); err != nil {
		return err
	}

	return nil
}

var srvCmd = &cobra.Command{
	Use:   "srv <command>",
	Short: "Run a server that offers Thema operations over the network",
	Long: `Run a server that offers Thema operations over the network.

	TODO not yet implemented
`,
}

var httpCmd = &cobra.Command{
	Use:   "http [-p port]",
	Short: "Start an HTTP(S) server",
	Long: `Start an HTTP(S) server.
`,
}
