package cmd

import (
	"fmt"
	"path"

	"alis.is/bb-cli/ami"
	"alis.is/bb-cli/bb"
	"alis.is/bb-cli/cli"
	"alis.is/bb-cli/system"
	"alis.is/bb-cli/util"

	bb_module_node "alis.is/bb-cli/bb/modules/node"
	bb_module_signer "alis.is/bb-cli/bb/modules/signer"

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

		if (shouldImportToSigner || !shouldImportToNode) && !cli.IsRemoteInstance && bb_module_signer.Module.IsInstalled() {
			log.Info("Importing key to the signer...")
			wasRunning, _ := bb.Signer.IsServiceStatus(ami.SignerService, "running")
			if wasRunning {
				exitCode, err := bb.Signer.Stop()
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

			exitCode, err := ami.Execute(bb.Signer.GetPath(), amiArgs...)
			util.AssertEE(err, "Failed to import key to signer!", exitCode)

			signerDef, _, err := bb.Signer.LoadAppDefinition()
			util.AssertEE(err, "Failed to load signer definition!", cli.ExitInvalidUser)
			signerUser, ok := signerDef["user"].(string)

			util.AssertBE(ok, "Failed to get username from signer!", cli.ExitInvalidUser)
			util.ChownR(signerUser, path.Join(bb.Signer.GetPath(), "data"))

			if wasRunning {
				bb.Signer.Start()
			}
		}
		if (shouldImportToNode || !shouldImportToSigner) && bb_module_node.Module.IsInstalled() {
			// import to node
			var wasSignerRunning bool
			if !cli.IsRemoteInstance {
				log.Info("Importing key to the node...")
				wasSignerRunning, _ = bb.Signer.IsServiceStatus(ami.SignerService, "running")
				exitCode, err := bb.Signer.Start()
				util.AssertEE(err, "Failed to start signer!", exitCode)

				isSignerRunning, _ := bb.Signer.IsServiceStatus(ami.SignerService, "running")
				util.AssertBE(isSignerRunning, "Signer is not running. Please start signer services.", cli.ExitSignerNotOperational)
			}

			bakerAddr, exitCode, err := bb.Signer.GetKeyHash(alias)
			util.AssertEE(err, "Failed to get baker key hash!", exitCode)
			ami.REMOTE_VARS[ami.BAKER_KEY_HASH_REMOTE_VAR] = bakerAddr
			amiArgs := []string{"import-key", bakerAddr}
			if force {
				amiArgs = append(amiArgs, "--force")
			}
			amiArgs = append(amiArgs, fmt.Sprintf("--alias=%s", alias))
			exitCode, err = ami.Execute(bb.Node.GetPath(), amiArgs...)
			util.AssertEE(err, "Failed to import key to node!", exitCode)

			if isRemote, _ := ami.IsRemoteApp(bb.Node.GetPath()); !isRemote {
				nodeDef, _, err := bb.Node.LoadAppDefinition()
				util.AssertEE(err, "Failed to load node definition!", cli.ExitAppConfigurationLoadFailed)
				nodeUser, ok := nodeDef["user"].(string)
				util.AssertBE(ok, "Failed to get username from node!", cli.ExitInvalidUser)
				util.ChownR(nodeUser, path.Join(bb.Node.GetPath(), "data"))
			}
			if !wasSignerRunning && !cli.IsRemoteInstance {
				bb.Signer.Stop()
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
