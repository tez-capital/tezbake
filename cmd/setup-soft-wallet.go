package cmd

import (
	"fmt"
	"path"
	"strings"

	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps"
	"github.com/tez-capital/tezbake/cli"
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
		shouldOperateOnDal, _ := cmd.Flags().GetBool("dal")
		force, _ := cmd.Flags().GetBool("force")
		keyAlias, _ := cmd.Flags().GetString("key-alias")

		isAnySelected := shouldOperateOnSigner || shouldOperateOnNode || shouldOperateOnDal

		if (shouldOperateOnSigner || !isAnySelected) && !cli.IsRemoteInstance && apps.Signer.IsInstalled() {
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
			if !cli.IsRemoteInstance {
				log.Info("Importing key to the node...")
				wasSignerRunning, _ = apps.Signer.IsServiceStatus(constants.SignerAppServiceId, "running")
				exitCode, err := apps.Signer.Start()
				util.AssertEE(err, "Failed to start signer!", exitCode)

				isSignerRunning, _ := apps.Signer.IsServiceStatus(constants.SignerAppServiceId, "running")
				util.AssertBE(isSignerRunning, "Signer is not running. Please start signer services.", constants.ExitSignerNotOperational)
			}
			defer func() {
				if isRemote := apps.Node.IsRemoteApp(); !isRemote {
					nodeDef, _, err := apps.Node.LoadAppDefinition()
					util.AssertEE(err, "Failed to load node definition!", constants.ExitAppConfigurationLoadFailed)
					nodeUser, ok := nodeDef["user"].(string)
					util.AssertBE(ok, "Failed to get username from node!", constants.ExitInvalidUser)
					util.ChownR(nodeUser, path.Join(apps.Node.GetPath(), "data"))
				}
				if !wasSignerRunning && !cli.IsRemoteInstance {
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

		// TODO: merge with setup ledger and simplify
		if (shouldOperateOnDal || !isAnySelected) && apps.DalNode.IsInstalled() {
			util.AssertBE(apps.Node.IsInstalled(), "node is not installed - can not import keys to dal node", constants.ExitAppNotInstalled)

			var wasSignerRunning bool
			if !cli.IsRemoteInstance {
				log.Info("Importing key to the node...")
				wasSignerRunning, _ = apps.Signer.IsServiceStatus(constants.SignerAppServiceId, "running")
				exitCode, err := apps.Signer.Start()
				util.AssertEE(err, "Failed to start signer!", exitCode)

				isSignerRunning, _ := apps.Signer.IsServiceStatus(constants.SignerAppServiceId, "running")
				util.AssertBE(isSignerRunning, "Signer is not running. Please start signer services.", constants.ExitSignerNotOperational)
			}
			defer func() {
				if isRemote := apps.DalNode.IsRemoteApp(); !isRemote {
					dalDef, _, err := apps.DalNode.LoadAppDefinition()
					util.AssertEE(err, "Failed to load node definition!", constants.ExitAppConfigurationLoadFailed)
					dalUser, ok := dalDef["user"].(string)
					util.AssertBE(ok, "Failed to get username from node!", constants.ExitInvalidUser)
					util.ChownR(dalUser, path.Join(apps.DalNode.GetPath(), "data"))
				}
				if !wasSignerRunning && !cli.IsRemoteInstance {
					apps.Signer.Stop()
				}
			}()

			output, exitCode, err := apps.Node.ExecuteGetOutput("list-bakers")
			util.AssertEE(err, "Failed to get baker key hash!", exitCode)
			util.AssertB(exitCode == 0, "Failed to get baker key hash!")

			keys := strings.Split(strings.TrimSpace(string(output)), "\n")

			err = apps.DalNode.SetAttesterProfiles(keys)
			util.AssertEE(err, "Failed to set attester profiles!", constants.ExitAppConfigurationLoadFailed)

			exitCode, err = apps.DalNode.Execute("setup", "--configure") // reconfigure to apply changes
			util.AssertEE(err, "Failed to reconfigure dal node!", exitCode)
			util.AssertBE(exitCode == 0, "Failed to setup dal node!", exitCode)
			log.Info("Keys imported into dal node!")
		}
	},
}

func init() {
	setupSoftWalletCmd.Flags().Bool("node", false, "Import key to node (affects import-key only)")
	setupSoftWalletCmd.Flags().Bool("signer", false, "Import key to signer (affects import-key only)")
	setupSoftWalletCmd.Flags().Bool("dal", false, "Import key to dal node (affects import-key only)")

	setupSoftWalletCmd.Flags().String("import-key", "", "Import key")
	setupSoftWalletCmd.Flags().String("generate", "ed25519", "Generate key")
	setupSoftWalletCmd.Flags().String("key-alias", "baker", "Alias ofkey to be imported")

	setupSoftWalletCmd.Flags().BoolP("force", "f", false, "Force key import. (overwrites existing)")

	RootCmd.AddCommand(setupSoftWalletCmd)
}
