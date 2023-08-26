package cmd

import (
	"alis.is/bb-cli/cli"
	"alis.is/bb-cli/system"
	"alis.is/bb-cli/util"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:    "install",
	Hidden: true,
	Short:  "Copies self to /usr/sbin/bb-cli.",
	Run: func(cmd *cobra.Command, args []string) {
		username := util.GetCommandStringFlag(cmd, User)
		if !system.IsElevated() {
			system.RequireElevatedUser("--user=" + username)
		}

		err := system.CopySelfToSystem(username)
		util.AssertEE(err, "Failed to copy self to system!", cli.ExitIOError)
	},
}

func init() {
	nodeCmd.Flags().SetInterspersed(false)

	RootCmd.AddCommand(installCmd)
}
