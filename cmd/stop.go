package cmd

import (
	"fmt"

	"alis.is/bb-cli/bb"
	"alis.is/bb-cli/system"
	"alis.is/bb-cli/util"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stops BB.",
	Long:  "Stops services of BB instance.",
	Run: func(cmd *cobra.Command, args []string) {
		system.RequireElevatedUser()

		all := true

		installedModules := bb.GetInstalledModules()
		for _, v := range installedModules {
			shouldStop, _ := cmd.Flags().GetBool(v.GetId())
			if shouldStop {
				all = false
			}
		}

		for _, v := range installedModules {
			shouldStop, _ := cmd.Flags().GetBool(v.GetId())
			if all || shouldStop {
				exitCode, err := v.Stop()
				util.AssertEE(err, fmt.Sprintf("Failed to stop %s's services!", v.GetId()), exitCode)
			}
		}

		log.Info("Requested services stopped succesfully")
	},
}

func init() {
	for _, v := range bb.Modules {
		stopCmd.Flags().Bool(v.GetId(), false, fmt.Sprintf("Stops %s's services.", v.GetId()))
	}
	RootCmd.AddCommand(stopCmd)
}
