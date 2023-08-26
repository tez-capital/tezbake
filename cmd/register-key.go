package cmd

import (
	"github.com/tez-capital/tezbake/apps"
	"github.com/tez-capital/tezbake/util"

	"github.com/spf13/cobra"
)

var registerKeyCmd = &cobra.Command{
	Use:   "register-key",
	Short: "Register key for baking.",
	Long:  "Registers key for baking.",
	Run: func(cmd *cobra.Command, args []string) {
		exitCode, err := apps.Signer.Execute("register-key")
		util.AssertEE(err, "Failed to import key!", exitCode)
	},
}

func init() {
	RootCmd.AddCommand(registerKeyCmd)
}
