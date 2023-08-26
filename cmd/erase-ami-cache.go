package cmd

import (
	"alis.is/bb-cli/ami"
	"alis.is/bb-cli/system"
	"alis.is/bb-cli/util"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var eraseAmiCacheCmd = &cobra.Command{
	Use:   "erase-ami-cache",
	Short: "Erases ami cache.",
	Long:  "Erases ami package cache.",
	Run: func(cmd *cobra.Command, args []string) {
		system.RequireElevatedUser()

		exitCode, err := ami.EraseCache()
		util.AssertEE(err, "Failed to erase ami cache!", exitCode)

		log.Info("'ami' package cache erased.")
	},
}

func init() {
	RootCmd.AddCommand(eraseAmiCacheCmd)
}
