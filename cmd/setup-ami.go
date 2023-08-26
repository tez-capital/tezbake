package cmd

import (
	"os"

	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/system"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var setupAmiCmd = &cobra.Command{
	Use:   "setup-ami",
	Short: "Install ami and eli.",
	Long:  "Install latest ami and eli.",
	Run: func(cmd *cobra.Command, args []string) {
		system.RequireElevatedUser()

		exitCode, err := ami.Install()
		if err != nil {
			log.WithFields(log.Fields{"error": err}).Error("Failed to install ami and eli!")
			os.Exit(exitCode)
		}
	},
}

func init() {
	setupAmiCmd.Flags().String("remote-node", "", "(Not available).")
	RootCmd.AddCommand(setupAmiCmd)
}
