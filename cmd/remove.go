package cmd

import (
	"fmt"
	"os"
	"slices"
	"strings"

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

		shouldRemoveAll := util.GetCommandBoolFlagS(cmd, "all")
		force := util.GetCommandBoolFlagS(cmd, "force")
		skipConfirm := util.GetCommandBoolFlagS(cmd, "confirm")

		selectedApps := GetAppsBySelectionCriteria(cmd, AppSelectionCriteria{
			InitialSelection:  AllApps,
			FallbackSelection: AllFallback,
		})

		removingAllInstalled := lo.EveryBy(apps.GetInstalledApps(), func(installedApp base.BakeBuddyApp) bool {
			return slices.Contains(selectedApps, installedApp)
		})

		proceed := skipConfirm
		if system.IsTty() && !skipConfirm {
			appsToRemove := strings.Join(lo.Map(selectedApps, func(app base.BakeBuddyApp, _ int) string {
				return strings.ToUpper(app.GetId())
			}), ", ")
			var prompt string
			fmt.Println("")
			fmt.Println("!!!!!! WARNING !!!!!!")
			fmt.Println("")
			switch {
			case removingAllInstalled && shouldRemoveAll:
				prompt = "Are you sure you want to remove all files related to tezbake instance?"
			case shouldRemoveAll:
				prompt = fmt.Sprintf("Are you sure you want to remove all files related to %s?", appsToRemove)
			default:
				prompt = fmt.Sprintf("Are you sure you want to remove %s data?", appsToRemove)
			}
			var err error
			proceed, err = util.PromptConfirm(prompt, false)
			if err != nil {
				proceed = false
			}
			if proceed {
				proceed = false
				abort := false
				fmt.Println("")
				prompt = "This operation is irreversible. Do you want to abort?"
				abort, err = util.PromptConfirm(prompt, false)
				if err != nil {
					abort = false
				}
				proceed = !abort
			}
		}
		if !proceed {
			log.Info("Aborting removal.")
			os.Exit(constants.ExitOperationCanceled)
		}
		removeArgs := []string{}
		if force {
			removeArgs = append(removeArgs, "--force")
		}

		for _, v := range selectedApps {
			exitCode, err := v.Remove(shouldRemoveAll, removeArgs...)
			util.AssertEE(err, fmt.Sprintf("Failed to remove %s!", v.GetId()), exitCode)
		}

		if removingAllInstalled && shouldRemoveAll {
			os.RemoveAll(cli.BBdir)
		}
		log.Info("BB removal successful")
	},
}

func init() {
	for _, v := range apps.All {
		removeCmd.Flags().Bool(v.GetId(), false, fmt.Sprintf("Removes %s.", v.GetId()))
	}
	removeCmd.Flags().BoolP("all", "a", false, "Removes all files related to BB instance.")
	removeCmd.Flags().Bool("force", false, "Forces removal even when there are no package specific removal routines.")
	removeCmd.Flags().Bool("confirm", false, "Skips confirmation prompts.")
	RootCmd.AddCommand(removeCmd)
}
