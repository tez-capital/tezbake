package cmd

import (
	"os"

	"alis.is/bb-cli/ami"
	"alis.is/bb-cli/bb"

	"github.com/spf13/cobra"
)

var signerCmd = &cobra.Command{
	Use:   "signer",
	Short: "Passes args through to signer app.",
	Long:  `Passes args through to signer app.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 && args[0] == "-" {
			args[0] = "signer"
		}
		exitCode, _ := ami.Execute(bb.Signer.GetPath(), args...)
		os.Exit(exitCode)
	},
}

func init() {
	signerCmd.Flags().SetInterspersed(false)
	RootCmd.AddCommand(signerCmd)
}
