package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/samber/lo"
	"github.com/tez-capital/tezbake/apps"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/system"
	"github.com/tez-capital/tezbake/util"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var updateDalProfilesCmd = &cobra.Command{
	Use:   "update-dal-profiles",
	Short: "Updates dal profiles.",
	Long:  "Updates dal profiles.",
	Args:  cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		util.AssertBE(apps.DalNode.IsInstalled(), "DAL node is not installed!", constants.ExitAppNotInstalled)
		util.AssertBE(apps.Node.IsInstalled(), "Octez node is not installed!", constants.ExitAppNotInstalled)

		system.RequireElevatedUser()

		autodetect := util.GetCommandBoolFlag(cmd, "auto")
		force := util.GetCommandBoolFlag(cmd, "force")

		args = util.RemoveCmdFlags(cmd, args)

		if len(args) == 0 && !autodetect {
			proceed := false
			survey.AskOne(&survey.Confirm{
				Message: "No keys provided. Do you want to autodetect?",
				Default: true,
			}, &proceed, nil)
			if !proceed {
				fmt.Println("No keys provided. Exiting.")
				os.Exit(constants.ExitOperationCanceled)
			}
			autodetect = true
		}

		keys := args

		if autodetect {
			output, exitCode, err := apps.Node.ExecuteGetOutput("list-bakers")
			util.AssertEE(err, "Failed to get baker key hash!", exitCode)
			util.AssertB(exitCode == 0, "Failed to get baker key hash!")

			foundKeys := strings.Split(strings.TrimSpace(string(output)), "\n")
			log.Info("Importing keys to dal node...", fmt.Sprintf("keys=%v", keys))

			keys = append(keys, foundKeys...)
		}

		profiles := make([]string, 0, len(keys))
		for _, key := range keys {
			profile := key
			if !force {
				var err error
				profile, err = util.ResolveAttestationProfile(key)
				util.AssertEE(err, "Failed to resolve attestation profile!", constants.ExitInternalError)
			}
			profiles = append(profiles, profile)
		}

		log.Info("Attester profiles resolved successfully, updating dal node...", fmt.Sprintf("profiles=%v", profiles))

		err := apps.DalNode.SetAttesterProfiles(lo.Uniq(profiles))
		util.AssertEE(err, "Failed to set attester profiles!", constants.ExitAppConfigurationLoadFailed)

		exitCode, err := apps.DalNode.Execute("setup", "--configure") // reconfigure to apply changes
		util.AssertEE(err, "Failed to setup dal node!", exitCode)
		util.AssertBE(exitCode == 0, "Failed to setup dal node!", exitCode)
		log.Info("Attester profiles updated successfully. ", fmt.Sprintf("profiles=%v", profiles))
	},
}

func init() {
	updateDalProfilesCmd.Flags().Bool("auto", false, "Autodetect attester profiles")
	updateDalProfilesCmd.Flags().Bool("force", false, "Force update attester profiles")

	RootCmd.AddCommand(updateDalProfilesCmd)
}
