package cmd

import (
	"fmt"
	"os"

	"alis.is/bb-cli/bb"
	"alis.is/bb-cli/cli"
	"alis.is/bb-cli/system"
	"alis.is/bb-cli/util"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Removes BB.",
	Long:  "Removes BB instance.",
	Run: func(cmd *cobra.Command, args []string) {
		system.RequireElevatedUser()

		shouldRemoveAll, _ := cmd.Flags().GetBool("all")
		all := true
		for _, v := range bb.Modules {
			shouldRemove, _ := cmd.Flags().GetBool(v.GetId())
			if shouldRemove {
				all = false
			}
		}

		for _, v := range bb.Modules {
			shouldRemove, _ := cmd.Flags().GetBool(v.GetId())
			if all || shouldRemove {
				exitCode, err := v.Remove(all)
				util.AssertEE(err, fmt.Sprintf("Failed to remove %s!", v.GetId()), exitCode)
			}
		}

		if all && shouldRemoveAll {
			os.RemoveAll(cli.BBdir)
		}
		log.Info("BB removal succesfull")
	},
}

func init() {
	for _, v := range bb.Modules {
		removeCmd.Flags().Bool(v.GetId(), false, fmt.Sprintf("Removes %s.", v.GetId()))
	}
	removeCmd.Flags().BoolP("all", "a", false, "Removes all files related to BB instance.")
	RootCmd.AddCommand(removeCmd)
}
