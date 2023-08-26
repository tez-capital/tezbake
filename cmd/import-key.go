package cmd

import (
	"fmt"
	"path"

	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps"
	"github.com/tez-capital/tezbake/cli"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/system"
	"github.com/tez-capital/tezbake/util"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var importKeyCmd = &cobra.Command{
	Use:   "import-key",
	Short: "Imports key from ledger.",
	Long:  "Imports key from selected ledger.",
	Run: func(cmd *cobra.Command, args []string) {
		system.RequireElevatedUser()

		shouldImportToSigner, _ := cmd.Flags().GetBool("signer")
		shouldImportToNode, _ := cmd.Flags().GetBool("node")
		force, _ := cmd.Flags().GetBool("force")
		alias, _ := cmd.Flags().GetString("alias")

		if (shouldImportToSigner || !shouldImportToNode) && !cli.IsRemoteInstance && apps.Signer.IsInstalled() {
			log.Info("Importing key to the signer...")
			wasRunning, _ := apps.Signer.IsServiceStatus(constants.SignerAppServiceId, "running")
			if wasRunning {
				exitCode, err := apps.Signer.Stop()
				util.AssertEE(err, "Failed to stop signer!", exitCode)
			}

			amiArgs := make([]string, 0)
			amiArgs = append(amiArgs, "import-key")

			derivPath, _ := cmd.Flags().GetString("derivation-path")
			if derivPath != "" {
				amiArgs = append(amiArgs, "--derivation-path="+derivPath)
			}

			ledgerId, _ := cmd.Flags().GetString("ledger-id")
			if ledgerId != "" {
				amiArgs = append(amiArgs, "--ledger-id="+ledgerId)
			}

			if force {
				amiArgs = append(amiArgs, "--force")
			}
			amiArgs = append(amiArgs, fmt.Sprintf("--alias=%s", alias))
			amiArgs = append(amiArgs, args...)

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
		if (shouldImportToNode || !shouldImportToSigner) && apps.Node.IsInstalled() {
			// import to node
			var wasSignerRunning bool
			if !cli.IsRemoteInstance {
				log.Info("Importing key to the node...")
				wasSignerRunning, _ = apps.Signer.IsServiceStatus(constants.SignerAppServiceId, "running")
				exitCode, err := apps.Signer.Start()
				util.AssertEE(err, "Failed to start signer!", exitCode)

				isSignerRunning, _ := apps.Signer.IsServiceStatus(constants.SignerAppServiceId, "running")
				util.AssertBE(isSignerRunning, "Signer is not running. Please start signer services.", constants.ExitSignerNotOperational)
			}

			bakerAddr, exitCode, err := apps.Signer.GetKeyHash(alias)
			util.AssertEE(err, "Failed to get baker key hash!", exitCode)
			ami.REMOTE_VARS[ami.BAKER_KEY_HASH_REMOTE_VAR] = bakerAddr
			amiArgs := []string{"import-key", bakerAddr}
			if force {
				amiArgs = append(amiArgs, "--force")
			}
			amiArgs = append(amiArgs, fmt.Sprintf("--alias=%s", alias))
			exitCode, err = apps.Node.Execute(amiArgs...)
			util.AssertEE(err, "Failed to import key to node!", exitCode)

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
		}
	},
}

func init() {
	importKeyCmd.Flags().BoolP("force", "f", false, "Force key import. (overwrites existing)")
	importKeyCmd.Flags().String("alias", "baker", "alias ofkey to be imported")
	importKeyCmd.Flags().BoolP("node", "n", false, "import key to node")
	importKeyCmd.Flags().BoolP("signer", "s", false, "import key to signer")
	importKeyCmd.Flags().StringP("ledger-id", "i", "", "Ledger id to import key from.")
	importKeyCmd.Flags().StringP("derivation-path", "d", "", "Derivation path you want to use.")
	RootCmd.AddCommand(importKeyCmd)
}
