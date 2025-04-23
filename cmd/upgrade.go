package cmd

import (
	"fmt"

	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps"
	"github.com/tez-capital/tezbake/system"
	"github.com/tez-capital/tezbake/util"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrades BB.",
	Long:  "Upgrades BB instance.",
	Run: func(cmd *cobra.Command, args []string) {
		system.RequireElevatedUser()

		upgradeContext := &apps.UpgradeContext{
			UpgradeStorage: util.GetCommandBoolFlagS(cmd, UpgradeStorage),
		}

		if util.GetCommandBoolFlagS(cmd, SetupAmi) {
			// install ami by default in case of remote instance
			exitCode, err := ami.Install()
			util.AssertEE(err, "Failed to install ami and eli!", exitCode)
		}

		exitCode, err := ami.EraseCache()
		util.AssertEE(err, "Failed to erase ami cache!", exitCode)

		for _, v := range GetAppsBySelectionCriteria(cmd, AppSelectionCriteria{
			InitialSelection:  InstalledApps,
			FallbackSelection: ImplicitApps,
		}) {
			exitCode, err := v.Upgrade(upgradeContext)
			util.AssertEE(err, fmt.Sprintf("Failed to upgrade '%s'!", v.GetId()), exitCode)
		}
		log.Info("Upgrade succesful.")
	},
}

func init() {
	for _, v := range apps.All {
		upgradeCmd.Flags().Bool(v.GetId(), false, fmt.Sprintf("Upgrade %s.", v.GetId()))
	}

	upgradeCmd.Flags().BoolP(UpgradeStorage, "s", false, "Upgrade storage during the upgrade.")
	upgradeCmd.Flags().BoolP(SetupAmi, "a", false, "Install latest ami during the BB upgrade.")
	RootCmd.AddCommand(upgradeCmd)
}
