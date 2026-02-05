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
	"go.alis.is/common/log"

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

		removingAllInstalled := lo.EveryBy(apps.GetInstalledApps(cmd), func(installedApp base.BakeBuddyApp) bool {
			return slices.Contains(selectedApps, installedApp)
		})

		isUserConfirmed := skipConfirm
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
				prompt = fmt.Sprintf("Are you sure you want to remove all files related to tezbake instance - %s?", cli.BBdir)
			case shouldRemoveAll:
				prompt = fmt.Sprintf("Are you sure you want to remove all files related to %s (%s)?", appsToRemove, cli.BBdir)
			default:
				prompt = fmt.Sprintf("Are you sure you want to remove %s data (%s)?", appsToRemove, cli.BBdir)
			}
			isUserConfirmed = util.Confirm(prompt, false, "Failed to confirm removal!")
			if isUserConfirmed {
				isUserConfirmed = false
				abort := false
				fmt.Println("")
				prompt = "This operation is irreversible. Do you want to abort?"
				abort = util.ConfirmWithCancelValue(prompt, false, true, "Failed to confirm removal abort!")
				isUserConfirmed = !abort
			}
		}
		if !isUserConfirmed {
			log.Info("Aborting removal.")
			os.Exit(constants.ExitOperationCanceled)
		}
		removeArgs := []string{}
		if force {
			removeArgs = append(removeArgs, "--force")
		}

		for _, app := range selectedApps {
			serviceInfo, err := app.GetServiceInfo()
			if err == nil && !force {
				for serviceName, service := range serviceInfo {
					util.AssertBE(service.Status != "running", fmt.Sprintf("%s service %s is running. Please stop the application first or use --force to override", app.GetId(), serviceName), constants.ExitUserInvalidInput)
				}
			}

			exitCode, err := app.Remove(shouldRemoveAll, removeArgs...)
			util.AssertEE(err, fmt.Sprintf("Failed to remove %s!", app.GetId()), exitCode)
		}

		if removingAllInstalled && shouldRemoveAll {
			os.RemoveAll(cli.BBdir)
		}
		log.Info("tezbake removal successful")
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
