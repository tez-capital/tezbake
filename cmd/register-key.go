package cmd

import (
	"alis.is/bb-cli/ami"
	"alis.is/bb-cli/bb"
	"alis.is/bb-cli/util"

	"github.com/spf13/cobra"
)

var registerKeyCmd = &cobra.Command{
	Use:   "register-key",
	Short: "Register key for baking.",
	Long:  "Registers key for baking.",
	Run: func(cmd *cobra.Command, args []string) {
		exitCode, err := ami.Execute(bb.Signer.GetPath(), "register-key")
		util.AssertEE(err, "Failed to import key!", exitCode)
	},
}

func init() {
	RootCmd.AddCommand(registerKeyCmd)
}
