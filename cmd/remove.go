package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/samber/lo"
	"github.com/tez-capital/tezbake/apps"
	"github.com/tez-capital/tezbake/apps/base"
	"github.com/tez-capital/tezbake/cli"
	"github.com/tez-capital/tezbake/constants"
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

		if system.IsTty() {
			proceed := false
			appsToRemove := strings.Join(lo.Map(selectedApps, func(app base.BakeBuddyApp, _ int) string {
				return app.GetId()
			}), ", ")
			var prompt string
			switch {
			case removingAllInstalled && shouldRemoveAll:
				prompt = "Are you sure you want to remove all files related to tezbake instance? (y/n)"
			case shouldRemoveAll:
				prompt = fmt.Sprintf("Are you sure you want to remove all files related to %s? (y/n) ", appsToRemove)
			default:
				prompt = fmt.Sprintf("Are you sure you want to remove %s data? (y/n) ", appsToRemove)
			}
			survey.AskOne(&survey.Confirm{Message: prompt}, &proceed)
			if !proceed {
				log.Info("Aborting removal.")
				os.Exit(constants.ExitOperationCanceled)
			}
		}

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
