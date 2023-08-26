package cmd

import (
	"fmt"
	"os"

	"github.com/tez-capital/tezbake/apps"
	"github.com/tez-capital/tezbake/cli"
	"github.com/tez-capital/tezbake/system"
	"github.com/tez-capital/tezbake/util"

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

		selectedApps := GetAppsBySelectionCriteria(cmd, AppSelectionCriteria{
			InitialSelection:  InstalledApps,
			FallbackSelection: AllFallback,
		})
		removingAllInstalled := len(selectedApps) == len(apps.GetInstalledApps())

		for _, v := range selectedApps {
			exitCode, err := v.Remove(shouldRemoveAll)
			util.AssertEE(err, fmt.Sprintf("Failed to remove %s!", v.GetId()), exitCode)
		}

		if removingAllInstalled && shouldRemoveAll {
			os.RemoveAll(cli.BBdir)
		}
		log.Info("BB removal succesfull")
	},
}

func init() {
	for _, v := range apps.All {
		removeCmd.Flags().Bool(v.GetId(), false, fmt.Sprintf("Removes %s.", v.GetId()))
	}
	removeCmd.Flags().BoolP("all", "a", false, "Removes all files related to BB instance.")
	RootCmd.AddCommand(removeCmd)
}
