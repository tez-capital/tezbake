package cmd

import (
	"fmt"

	"alis.is/bb-cli/bb"
	"alis.is/bb-cli/system"
	"alis.is/bb-cli/util"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts BB.",
	Long:  "Starts services of BB instance.",
	Run: func(cmd *cobra.Command, args []string) {
		system.RequireElevatedUser()

		all := true
		installedModules := bb.GetInstalledModules()
		for _, v := range installedModules {
			shouldStart, _ := cmd.Flags().GetBool(v.GetId())
			if shouldStart {
				all = false
			}
		}

		for _, v := range installedModules {
			shouldStart, _ := cmd.Flags().GetBool(v.GetId())
			if all || shouldStart {
				exitCode, err := v.Start()
				util.AssertEE(err, fmt.Sprintf("Failed to starts %s's services!", v.GetId()), exitCode)
			}
		}

		log.Info("Requested services started succesfully")
	},
}

func init() {
	for _, v := range bb.Modules {
		startCmd.Flags().Bool(v.GetId(), false, fmt.Sprintf("Starts %s's services.", v.GetId()))
	}
	RootCmd.AddCommand(startCmd)
}
