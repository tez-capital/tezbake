package cmd

import (
	"alis.is/bb-cli/ami"
	"alis.is/bb-cli/bb"
	"alis.is/bb-cli/util"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var setupLedgerCmd = &cobra.Command{
	Use:   "setup-ledger",
	Short: "Setup ledger for baking.",
	Long:  "Setups ledger for baking.",
	Run: func(cmd *cobra.Command, args []string) {
		amiArgs := make([]string, 0)
		amiArgs = append(amiArgs, "setup-ledger")

		mainChainId, _ := cmd.Flags().GetString("main-chain-id")
		if mainChainId != "" {
			amiArgs = append(amiArgs, "--main-chain-id="+mainChainId)
		}

		mainHwm, _ := cmd.Flags().GetString("main-hwm")
		if mainHwm != "" {
			amiArgs = append(amiArgs, "--main-hwm="+mainHwm)
		}

		log.Info("To complete setup process you have to confirm operation on your ledger...")
		exitCode, err := ami.Execute(bb.Signer.GetPath(), amiArgs...)
		util.AssertEE(err, "Failed to import key!", exitCode)
	},
}

func init() {
	setupLedgerCmd.Flags().String("main-chain-id", "", "Id of chain to be used for baking.")
	setupLedgerCmd.Flags().String("main-hwm", "", "High watermark to be used during baking.")
	RootCmd.AddCommand(setupLedgerCmd)
}
