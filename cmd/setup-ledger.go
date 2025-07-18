package cmd

import (
	"fmt"
	"path"
	"time"

	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/system"
	"github.com/tez-capital/tezbake/util"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	platform  *BoolStringCombinedFlag
	importKey *BoolStringCombinedFlag
)

var setupLedgerCmd = &cobra.Command{
	Use:   "setup-ledger",
	Short: "Setup ledger for baking.",
	Long:  "Setups ledger for baking.",
	Run: func(cmd *cobra.Command, args []string) {
		system.RequireElevatedUser()

		shouldOperateOnSigner, _ := cmd.Flags().GetBool("signer")
		shouldOperateOnNode, _ := cmd.Flags().GetBool("node")
		force, _ := cmd.Flags().GetBool("force")
		keyAlias, _ := cmd.Flags().GetString("key-alias")
		protocol, _ := cmd.Flags().GetString("protocol")

		isAnySelected := shouldOperateOnSigner || shouldOperateOnNode

		if (shouldOperateOnSigner || !isAnySelected) && apps.Signer.IsInstalled() {
			log.Info("setting up ledger for signer...")
			wasRunning, _ := apps.Signer.IsServiceStatus(constants.SignerAppServiceId, "running")
			if wasRunning {
				exitCode, err := apps.Signer.Stop()
				util.AssertEE(err, "Failed to stop signer!", exitCode)
			}

			amiArgs := []string{"setup-ledger"}
			if platform.HasValue() {
				amiArgs = append(amiArgs, "--platform="+platform.String())
			} else if platform.IsTrue() {
				amiArgs = append(amiArgs, "--platform")
			}

			noUdev, _ := cmd.Flags().GetString("no-udev")
			if noUdev != "" {
				amiArgs = append(amiArgs, "--no-udev")
			}

			if protocol != "" {
				amiArgs = append(amiArgs, fmt.Sprintf("--protocol=%s", protocol))
			}

			if importKey.HasValue() {
				amiArgs = append(amiArgs, "--import-key="+importKey.String())
			} else if importKey.IsTrue() {
				amiArgs = append(amiArgs, "--import-key")
			}

			ledgerId, _ := cmd.Flags().GetString("ledger-id")
			if ledgerId != "" {
				amiArgs = append(amiArgs, "--ledger-id="+ledgerId)
			}

			amiArgs = append(amiArgs, fmt.Sprintf("--key-alias=%s", keyAlias))

			authorize, _ := cmd.Flags().GetBool("authorize")
			if authorize {
				amiArgs = append(amiArgs, "--authorize")
			}
			chainId, _ := cmd.Flags().GetString("chain-id")
			if chainId != "" {
				amiArgs = append(amiArgs, "--chain-id="+chainId)
			}

			hwm, _ := cmd.Flags().GetString("hwm")
			if hwm != "" {
				amiArgs = append(amiArgs, "--hwm="+hwm)
			}

			if force {
				amiArgs = append(amiArgs, "--force")
			}

			exitCode, err := apps.Signer.Execute(amiArgs...)
			util.AssertEE(err, "Failed to import key to signer!", exitCode)

			signerDef, _, err := apps.Signer.LoadAppDefinition()
			util.AssertEE(err, "Failed to load signer definition!", constants.ExitInvalidUser)
			signerUser, ok := signerDef["user"].(string)

			util.AssertBE(ok, "Failed to get username from signer!", constants.ExitInvalidUser)
			util.ChownR(signerUser, path.Join(apps.Signer.GetPath(), "data"))

			if wasRunning {
				apps.Signer.Start()
			}
		}

		if (shouldOperateOnNode || !isAnySelected) && apps.Node.IsInstalled() {
			if importKey.IsTrue() { // node only imports key
				var wasSignerRunning bool

				log.Info("Importing key to the node...")
				wasSignerRunning, _ = apps.Signer.IsServiceStatus(constants.SignerAppServiceId, "running")
				if !wasSignerRunning {
					exitCode, err := apps.Signer.Start()
					util.AssertEE(err, "Failed to start signer!", exitCode)

					// Sleep 2 seconds to allow the signer service to start up
					time.Sleep(3 * time.Second)
				}

				isSignerRunning, _ := apps.Signer.IsServiceStatus(constants.SignerAppServiceId, "running")
				util.AssertBE(isSignerRunning, "Signer is not running. Please start signer services.", constants.ExitSignerNotOperational)
				defer func() {
					if isRemote := apps.Node.IsRemoteApp(); !isRemote {
						nodeDef, _, err := apps.Node.LoadAppDefinition()
						util.AssertEE(err, "Failed to load node definition!", constants.ExitAppConfigurationLoadFailed)
						nodeUser, ok := nodeDef["user"].(string)
						util.AssertBE(ok, "Failed to get username from node!", constants.ExitInvalidUser)
						util.ChownR(nodeUser, path.Join(apps.Node.GetPath(), "data"))
					}
					if !wasSignerRunning {
						apps.Signer.Stop()
					}
				}()

				bakerAddr, exitCode, err := apps.Signer.GetKeyHash(keyAlias)
				util.AssertEE(err, "Failed to get baker key hash!", exitCode)
				ami.REMOTE_VARS[ami.BAKER_KEY_HASH_REMOTE_VAR] = bakerAddr
				amiArgs := []string{"import-key", bakerAddr}
				if force {
					amiArgs = append(amiArgs, "--force")
				}
				amiArgs = append(amiArgs, fmt.Sprintf("--alias=%s", keyAlias))
				exitCode, err = apps.Node.Execute(amiArgs...)
				util.AssertEE(err, "Failed to import key to node!", exitCode)
			}
		}
	},
}

func init() {
	setupLedgerCmd.Flags().Bool("node", false, "Import key to node (affects import-key only)")
	setupLedgerCmd.Flags().Bool("signer", false, "Import key to signer (affects import-key only)")

	importKey = addCombinedFlag(setupLedgerCmd, "import-key", "", "Import key from ledger (optionally specify derivation path)")
	setupLedgerCmd.Flags().String("ledger-id", "", "Ledger id to import key from (affects import-key only)")
	setupLedgerCmd.Flags().String("key-alias", "baker", "Alias ofkey to be imported")

	setupLedgerCmd.Flags().Bool("authorize", false, "Authorize ledger for baking.")
	setupLedgerCmd.Flags().String("chain-id", "", "Id of chain to be used for baking.")
	setupLedgerCmd.Flags().String("hwm", "", "High watermark to be used during baking.")

	setupLedgerCmd.Flags().String("protocol", "", "Protocol hash to be used during setup-ledger.")

	platform = addCombinedFlag(setupLedgerCmd, "platform", "", "Prepare platform for ledger (optionally specify platform to override)")
	setupLedgerCmd.Flags().String("no-udev", "", "Skip udev rules installation. (linux only)")

	setupLedgerCmd.Flags().BoolP("force", "f", false, "Force key import. (overwrites existing)")

	RootCmd.AddCommand(setupLedgerCmd)
}
