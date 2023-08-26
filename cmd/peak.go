package cmd

import (
	"os"

	"github.com/tez-capital/tezbake/apps"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/util"

	"github.com/spf13/cobra"
)

var peakCmd = &cobra.Command{
	Use:   "peak",
	Short: "Passes args through to peak app.",
	Long:  `Passes args through to peak app.`,
	Run: func(cmd *cobra.Command, args []string) {
		util.AssertBE(apps.Peak.IsInstalled(), "Peak app is not installed!", constants.ExitAppNotInstalled)
		exitCode, _ := apps.Peak.Execute(args...)
		os.Exit(exitCode)
	},
}

func init() {
	peakCmd.Flags().SetInterspersed(false)
	RootCmd.AddCommand(peakCmd)
}
