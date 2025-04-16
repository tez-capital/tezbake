package cmd

import (
	"os"

	"github.com/tez-capital/tezbake/apps"

	"github.com/spf13/cobra"
)

var dalCmd = &cobra.Command{
	Use:                "dal",
	Short:              "Passes args through to dal node app.",
	Long:               `Passes args through to dal node app.`,
	DisableFlagParsing: true,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 && args[0] == "-" {
			args[0] = "dal-node"
		}
		exitCode, _ := apps.DalNode.Execute(args...)
		os.Exit(exitCode)
	},
}

func init() {
	dalCmd.Flags().SetInterspersed(false)

	RootCmd.AddCommand(dalCmd)
}
