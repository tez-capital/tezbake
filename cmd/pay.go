package cmd

import (
	"os"
	"strings"

	"github.com/samber/lo"
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
		nonOptionArgsCount := lo.CountBy(args, func(s string) bool { return !strings.HasPrefix(s, "-") })
		hasHelpOption := lo.ContainsBy(args, func(s string) bool { return s == "-h" || s == "--help" })

		if nonOptionArgsCount == 0 && !hasHelpOption {
			args = append([]string{"pay"}, args...) // default to "pay" subcommand
		}
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
