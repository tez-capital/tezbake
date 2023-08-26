//go:build server

package cmd

import (
	"alis.is/bb-cli/server"

	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:    "serve",
	Short:  "Run bb-cli server.",
	Long:   "Run bb-cli server",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		listenOn, _ := cmd.Flags().GetString("listen")
		useHttps, _ := cmd.Flags().GetBool("https")
		server.RunMonitoringServer(&listenOn, useHttps)
	},
}

func init() {
	serverCmd.Flags().String("listen", "127.0.0.1:9027", "Sets address for bb-cli to listen on.")
	serverCmd.Flags().Bool("https", true, "Whether to use https.")
	RootCmd.AddCommand(serverCmd)
}
