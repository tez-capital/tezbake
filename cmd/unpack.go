package cmd

import (
	"archive/zip"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps"
	"github.com/tez-capital/tezbake/apps/base"
	"github.com/tez-capital/tezbake/cli"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/system"
	"github.com/tez-capital/tezbake/util"
	"go.alis.is/common/log"
)

var unpackCmd = &cobra.Command{
	Use:   "unpack",
	Short: "Unpack a merged tezbake archive",
	Long:  "Overlays an archive created by 'tezbake pack' onto an existing setup, like an upgrade. Every app in the archive must already be installed: run 'tezbake setup' first, locally or with --node-remote, then unpack. Data (chain data, keys, configuration) comes from the archive; data not in the archive (e.g. chain data excluded from a pack) is kept. The installation keeps its identity and wiring: app id, user, remote connection and the signer node connection. Running apps are stopped during the restore and started again after it succeeds; apps that were not running stay stopped until 'tezbake start'. Before changing anything, unpack asks you to confirm that every other instance of these apps is stopped and removed, because two instances baking with the same keys double bake; --confirm skips the prompts.",
	Run: func(cmd *cobra.Command, _ []string) {
		system.RequireElevatedUser()

		source := util.GetCommandStringFlagSD(cmd, UnpackSrcFlag, "tezbake-pack.zip")
		err := unpackInstalledApps(cmd, source)
		if errors.Is(err, errUnpackCanceled) {
			log.Info("Aborting unpack.")
			os.Exit(constants.ExitOperationCanceled)
		}
		util.AssertEE(err, "Failed to unpack tezbake apps!", constants.ExitExternalError)
	},
}

// isAppInstalled is a variable so tests do not need AMI.
var isAppInstalled = func(app base.BakeBuddyApp) (bool, error) {
	return ami.IsAppInstalledS(app.GetLocalPath())
}

var errUnpackCanceled = errors.New("unpack canceled")

// isInteractive is a variable so tests do not depend on how they are run.
var isInteractive = system.IsTty

// confirmUnpack makes the user confirm, twice like remove, that no other instance runs the restored apps:
// a backup restored next to its original bakes with the same keys, and double baking is slashed.
func confirmUnpack(cmd *cobra.Command, targets []string) bool {
	if util.GetCommandBoolFlagS(cmd, UnpackConfirmFlag) {
		return true
	}
	if !isInteractive() {
		log.Error("Unpack needs confirmation. Pass --confirm to unpack without a terminal.")
		return false
	}
	fmt.Println("")
	fmt.Println("!!!!!! WARNING !!!!!!")
	fmt.Println("")
	fmt.Printf("Unpack restores %s into %s.\n", strings.Join(targets, ", "), cli.BBdir)
	fmt.Println("Stop and remove every other instance of these apps first, including the one this backup was made from.")
	fmt.Println("Two instances baking with the same keys double bake, and double baking is slashed.")
	fmt.Println("")
	if !util.Confirm("Have you stopped and removed all other instances of these apps?", false, "Failed to confirm unpack!") {
		return false
	}
	fmt.Println("")
	abort := util.ConfirmWithCancelValue("Baking from two instances gets your stake slashed. Do you want to abort?", false, true, "Failed to confirm unpack abort!")
	return !abort
}

func unpackInstalledApps(cmd *cobra.Command, source string) error {
	archive, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer archive.Close()

	metadata, err := loadTezbakePackMetadata(&archive.Reader)
	if err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp("", "tezbake-unpack-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	type restoreApp struct {
		app    base.BakeBuddyApp
		source string
	}
	plans := make([]restoreApp, 0, len(metadata.Apps))
	targets := make([]string, 0, len(metadata.Apps))

	// Validate every app before any restore changes an existing installation.
	for _, packedApp := range metadata.Apps {
		app, ok := apps.FromId(packedApp.Id)
		if !ok {
			return fmt.Errorf("unsupported app id %q in archive", packedApp.Id)
		}

		appArchivePath := filepath.Join(tempDir, packedApp.Id+".zip")
		if err := extractArchiveFile(&archive.Reader, path.Join("apps", packedApp.Id+".zip"), appArchivePath, 0600); err != nil {
			return fmt.Errorf("failed to extract archive for %s: %w", packedApp.Id, err)
		}
		if err := base.ValidateAppArchive(appArchivePath, packedApp.Id); err != nil {
			return fmt.Errorf("invalid archive for %s: %w", packedApp.Id, err)
		}
		installed, err := isAppInstalled(app)
		if err != nil {
			return fmt.Errorf("failed to check installation of %s: %w", packedApp.Id, err)
		}
		if !installed {
			return fmt.Errorf("%s is not installed; run 'tezbake setup' first", packedApp.Id)
		}

		plans = append(plans, restoreApp{app, appArchivePath})
		target := strings.ToUpper(packedApp.Id)
		if isRemote, locator := ami.IsRemoteApp(app.GetLocalPath()); isRemote {
			target += fmt.Sprintf(" on %s@%s:%s", locator.Username, locator.Host, locator.Port)
		}
		targets = append(targets, target)
	}
	if !confirmUnpack(cmd, targets) {
		return errUnpackCanceled
	}

	for _, plan := range plans {
		exitCode, err := plan.app.Unpack(plan.source)
		if err != nil {
			return fmt.Errorf("failed to unpack app %s: %w", plan.app.GetId(), err)
		}
		if exitCode != 0 {
			return fmt.Errorf("failed to unpack app %s: ami unpack exited with %d", plan.app.GetId(), exitCode)
		}
	}

	log.Info("Unpack successful", "source", source, "apps", len(metadata.Apps))
	return nil
}

func init() {
	unpackCmd.Flags().String(UnpackSrcFlag, "tezbake-pack.zip", "Path to the merged archive.")
	unpackCmd.Flags().Bool(UnpackConfirmFlag, false, "Skips the confirmation that no other instance runs the restored apps.")
	hideFlagsInHelp(unpackCmd, OUTPUT_FORMAT_FLAG)

	RootCmd.AddCommand(unpackCmd)
}
