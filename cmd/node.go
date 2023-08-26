package cmd

import (
	"os"

	"alis.is/bb-cli/ami"
	"alis.is/bb-cli/bb"

	"github.com/spf13/cobra"
)

var nodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Passes args through to node app.",
	Long:  `Passes args through to node app.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 && args[0] == "-" {
			args[0] = "node"
		}
		exitCode, _ := ami.Execute(bb.Node.GetPath(), args...)
		os.Exit(exitCode)
	},
}

func init() {
	nodeCmd.Flags().SetInterspersed(false)

	RootCmd.AddCommand(nodeCmd)
}
