package cmd

import (
	"fmt"

	"alis.is/bb-cli/ami"
	"alis.is/bb-cli/bb"
	bb_module "alis.is/bb-cli/bb/modules"
	"alis.is/bb-cli/cli"
	"alis.is/bb-cli/system"
	"alis.is/bb-cli/util"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrades BB.",
	Long:  "Upgrades BB instance.",
	Run: func(cmd *cobra.Command, args []string) {
		system.RequireElevatedUser()

		upgradeContext := &bb_module.UpgradeContext{
			UpgradeStorage: util.GetCommandBoolFlagS(cmd, UpgradeStorage),
		}

		if util.GetCommandBoolFlagS(cmd, SetupAmi) || cli.IsRemoteInstance {
			// install ami by default in case of remote instance
			exitCode, err := ami.Install()
			util.AssertEE(err, "Failed to install ami and eli!", exitCode)
		}

		exitCode, err := ami.EraseCache()
		util.AssertEE(err, "Failed to erase ami cache!", exitCode)

		someModuleSelected := false
		installedModules := bb.GetInstalledModules()
		for _, v := range installedModules {
			if checked, _ := cmd.Flags().GetBool(v.GetId()); checked {
				someModuleSelected = true
			}
			if someModuleSelected {
				break
			}
		}

		for _, v := range installedModules {
			shouldUpgrade, _ := cmd.Flags().GetBool(v.GetId())
			if !someModuleSelected || shouldUpgrade {
				exitCode, err := v.Upgrade(upgradeContext)
				util.AssertEE(err, fmt.Sprintf("Failed to upgrade '%s'!", v.GetId()), exitCode)
			}
		}
		log.Info("Upgrade succesful.")
	},
}

func init() {
	for _, v := range bb.Modules {
		upgradeCmd.Flags().Bool(v.GetId(), false, fmt.Sprintf("Upgrade %s.", v.GetId()))
	}

	upgradeCmd.Flags().BoolP(UpgradeStorage, "s", false, "Upgrade storage during the upgrade.")
	upgradeCmd.Flags().BoolP(SetupAmi, "a", false, "Install latest ami during the BB upgrade.")
	RootCmd.AddCommand(upgradeCmd)
}
