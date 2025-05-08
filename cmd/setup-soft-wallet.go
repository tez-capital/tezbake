package cmd

import (
	"fmt"
	"path"

	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/system"
	"github.com/tez-capital/tezbake/util"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var setupSoftWalletCmd = &cobra.Command{
	Use:   "setup-soft-wallet",
	Short: "Setup soft wallet for baking.",
	Long:  "Setups soft wallet for baking.",
	Run: func(cmd *cobra.Command, args []string) {
		system.RequireElevatedUser()

		shouldOperateOnSigner, _ := cmd.Flags().GetBool("signer")
		shouldOperateOnNode, _ := cmd.Flags().GetBool("node")
		force, _ := cmd.Flags().GetBool("force")
		keyAlias, _ := cmd.Flags().GetString("key-alias")

		isAnySelected := shouldOperateOnSigner || shouldOperateOnNode

		if (shouldOperateOnSigner || !isAnySelected) && apps.Signer.IsInstalled() {
			log.Info("setting up ledger for signer...")
			wasRunning, _ := apps.Signer.IsServiceStatus(constants.SignerAppServiceId, "running")
			if wasRunning {
				exitCode, err := apps.Signer.Stop()
				util.AssertEE(err, "Failed to stop signer!", exitCode)
			}

			importKey, _ := cmd.Flags().GetString("import-key")
			generate, _ := cmd.Flags().GetString("generate")

			amiArgs := []string{"setup-soft-wallet"}
			if importKey != "" {
				amiArgs = append(amiArgs, "--import-key="+importKey)
			}
			if generate != "" {
				amiArgs = append(amiArgs, "--generate="+generate)
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
			var wasSignerRunning bool

			log.Info("Importing key to the node...")
			wasSignerRunning, _ = apps.Signer.IsServiceStatus(constants.SignerAppServiceId, "running")
			exitCode, err := apps.Signer.Start()
			util.AssertEE(err, "Failed to start signer!", exitCode)

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
	},
}

func init() {
	setupSoftWalletCmd.Flags().Bool("node", false, "Import key to node (affects import-key only)")
	setupSoftWalletCmd.Flags().Bool("signer", false, "Import key to signer (affects import-key only)")

	setupSoftWalletCmd.Flags().String("import-key", "", "Import key")
	setupSoftWalletCmd.Flags().String("generate", "ed25519", "Generate key")
	setupSoftWalletCmd.Flags().String("key-alias", "baker", "Alias ofkey to be imported")

	setupSoftWalletCmd.Flags().BoolP("force", "f", false, "Force key import. (overwrites existing)")

	RootCmd.AddCommand(setupSoftWalletCmd)
}
