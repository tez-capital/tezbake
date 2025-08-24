package cmd

import (
	"os"

	"github.com/tez-capital/tezbake/apps"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/util"

	"github.com/spf13/cobra"
)

var payCmd = &cobra.Command{
	Use:                "pay",
	Short:              "Passes args through to tezpay app.",
	Long:               `Passes args through to tezpay app.`,
	DisableFlagParsing: true,
	Run: func(cmd *cobra.Command, _ []string) {
		args := util.GetCommandArgs(cmd)
		util.AssertBE(apps.Pay.IsInstalled(), "Pay app is not installed!", constants.ExitAppNotInstalled)
		if len(args) > 0 && args[0] == "-" {
			args[0] = "pay"
		}
		exitCode, _ := apps.Pay.Execute(args...)
		os.Exit(exitCode)
	},
}

func init() {
	payCmd.Flags().SetInterspersed(false)

	RootCmd.AddCommand(payCmd)
}
