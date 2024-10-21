package cmd

import (
	"os"

	"github.com/tez-capital/tezbake/apps"

	"github.com/spf13/cobra"
)

var signerCmd = &cobra.Command{
	Use:                "signer",
	Short:              "Passes args through to signer app.",
	Long:               `Passes args through to signer app.`,
	DisableFlagParsing: true,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 && args[0] == "-" {
			args[0] = "signer"
		}
		exitCode, _ := apps.Signer.Execute(args...)
		os.Exit(exitCode)
	},
}

func init() {
	signerCmd.Flags().SetInterspersed(false)

	RootCmd.AddCommand(signerCmd)
}
