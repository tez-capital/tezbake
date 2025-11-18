package cmd

import (
	"os"
	"slices"

	"github.com/tez-capital/tezbake/apps"
	"github.com/tez-capital/tezbake/util"

	"github.com/spf13/cobra"
)

var octezSignerCmd = &cobra.Command{
	Use:                "octez-signer",
	Short:              "Passes args through to signer app - octez-signer.",
	Long:               `Passes args through to signer app - octez-signer.`,
	DisableFlagParsing: true,
	Run: func(cmd *cobra.Command, _ []string) {
		args := util.GetCommandArgs(cmd)
		args = slices.Insert(args, 0, "signer")
		exitCode, _ := apps.Signer.Execute(args...)
		os.Exit(exitCode)
	},
}

func init() {
	octezSignerCmd.Flags().SetInterspersed(false)

	RootCmd.AddCommand(octezSignerCmd)
}
