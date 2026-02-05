package cmd

import (
	"fmt"
	"os"

	"github.com/alis-is/go-common/log"
	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/system"
	"github.com/tez-capital/tezbake/util"

	"github.com/spf13/cobra"
)

var (
	tezsignPlatformFlag  *BoolStringCombinedFlag
	tezsignImportKeyFlag *BoolStringCombinedFlag
)

var setupTezsignCmd = &cobra.Command{
	Use:   "setup-tezsign",
	Short: "Setup tezsign for baking.",
	Long:  "Setups tezsign for baking.",
	Run: func(cmd *cobra.Command, args []string) {
		shouldOperateOnSigner, _ := cmd.Flags().GetBool("signer")
		shouldOperateOnNode, _ := cmd.Flags().GetBool("node")
		init, _ := cmd.Flags().GetBool("init")
		force, _ := cmd.Flags().GetBool("force")
		keyAlias, _ := cmd.Flags().GetString("key-alias")
		password, _ := cmd.Flags().GetBool("password")

		isAnySelected := shouldOperateOnSigner || shouldOperateOnNode

		if tezsignImportKeyFlag.IsTrue() && (tezsignPlatformFlag.IsTrue() || init) {
			log.Error("Cannot use --import-key together with --platform or --init. Please run setup-tezsign in two steps.")
			os.Exit(constants.ExitInvalidArgs)
			return
		}

		if tezsignPlatformFlag.IsTrue() || init || password {
			system.RequireElevatedUser() // platform setup, init and password require elevated permissions
		}

		if tezsignImportKeyFlag.IsTrue() { // tezsign import requires signer to be running
			log.Info("ensuring signer is running for tezsign key import...")
			wasRunning, _ := apps.Signer.IsServiceStatus(constants.TezpayAppServiceId, "running")
			if !wasRunning {
				system.RequireElevatedUser() // starting signer service requires elevated permissions
				exitCode, err := apps.Signer.Start()
				util.AssertEE(err, "Failed to start signer!", exitCode)
				defer apps.Signer.Stop()
			}
		}

		if (shouldOperateOnSigner || !isAnySelected) && apps.Signer.IsInstalled() {
			log.Info("setting up tezsign for signer...")

			amiArgs := []string{"setup-tezsign"}

			if init {
				amiArgs = append(amiArgs, "--init")
			}
			if tezsignPlatformFlag.HasValue() {
				amiArgs = append(amiArgs, "--platform="+tezsignPlatformFlag.String())
			} else if tezsignPlatformFlag.IsTrue() {
				amiArgs = append(amiArgs, "--platform")
			}

			noUdev, _ := cmd.Flags().GetString("no-udev")
			if noUdev != "" {
				amiArgs = append(amiArgs, "--no-udev")
			}

			if password {
				amiArgs = append(amiArgs, "--password")
			}

			if tezsignImportKeyFlag.HasValue() {
				amiArgs = append(amiArgs, "--import-key="+tezsignImportKeyFlag.String())
			} else if tezsignImportKeyFlag.IsTrue() {
				amiArgs = append(amiArgs, "--import-key")
			}

			amiArgs = append(amiArgs, fmt.Sprintf("--key-alias=%s", keyAlias))

			if force {
				amiArgs = append(amiArgs, "--force")
			}

			exitCode, err := apps.Signer.Execute(amiArgs...)
			util.AssertEE(err, "Failed to import key to signer!", exitCode)
		}

		if (shouldOperateOnNode || !isAnySelected) && apps.Node.IsInstalled() && tezsignImportKeyFlag.IsTrue() { // node only imports key
			log.Info("Importing key to the node...")

			isSignerRunning, _ := apps.Signer.IsServiceStatus(constants.SignerAppServiceId, "running")
			util.AssertBE(isSignerRunning, "Signer is not running. Please start signer services.", constants.ExitSignerNotOperational)

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
	setupTezsignCmd.Flags().Bool("node", false, "Import key to node (affects import-key only)")
	setupTezsignCmd.Flags().Bool("signer", false, "Import key to signer (affects import-key only)")
	setupTezsignCmd.Flags().Bool("init", false, "Initialize tezsign configuration.")
	setupTezsignCmd.Flags().Bool("password", false, "Setup tezsign unlock password.")

	tezsignImportKeyFlag = addCombinedFlag(setupTezsignCmd, "import-key", "", "Import key from tezsign (optionally specify derivation path)")
	setupTezsignCmd.Flags().String("key-alias", "baker", "Alias ofkey to be imported")

	tezsignPlatformFlag = addCombinedFlag(setupTezsignCmd, "platform", "", "Prepare platform for tezsign (optionally specify platform to override)")
	setupTezsignCmd.Flags().String("no-udev", "", "Skip udev rules installation. (linux only)")

	setupTezsignCmd.Flags().BoolP("force", "f", false, "Force key import. (overwrites existing)")

	RootCmd.AddCommand(setupTezsignCmd)
}
