package cmd

import (
	"os"
	"slices"

	"github.com/tez-capital/tezbake/apps"
	"github.com/tez-capital/tezbake/util"

	"github.com/spf13/cobra"
)

var tezsignCmd = &cobra.Command{
	Use:                "tezsign",
	Short:              "Passes args through to signer app - tezsign.",
	Long:               `Passes args through to signer app - tezsign.`,
	DisableFlagParsing: true,
	Run: func(cmd *cobra.Command, _ []string) {
		args := util.GetCommandArgs(cmd)
		args = slices.Insert(args, 0, "tezsign")
		exitCode, _ := apps.Signer.Execute(args...)
		os.Exit(exitCode)
	},
}

func init() {
	tezsignCmd.Flags().SetInterspersed(false)

	RootCmd.AddCommand(tezsignCmd)
}
