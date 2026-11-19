package cmd

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tez-capital/tezbake/apps/base"
	"github.com/tez-capital/tezbake/cli"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/system"
	"github.com/tez-capital/tezbake/util"
	"go.alis.is/common/log"
)

var packCmd = &cobra.Command{
	Use:   "pack",
	Short: "Pack installed apps into one archive",
	Long:  "Builds a merged archive containing all installed tezbake apps. Running apps are stopped while they are packed and started again afterwards.",
	Run: func(cmd *cobra.Command, _ []string) {
		system.RequireElevatedUser()

		mode := util.GetCommandStringFlagSD(cmd, PackModeFlag, base.PackModeFull)
		util.AssertEE(base.ValidatePackMode(mode), "Invalid pack mode!", constants.ExitInvalidArgs)

		output := util.GetCommandStringFlagSD(cmd, PackOutputFlag, "tezbake-pack.zip")
		err := packInstalledApps(output, mode)
		util.AssertEE(err, "Failed to pack tezbake apps!", constants.ExitExternalError)
	},
}

func packInstalledApps(output string, mode string) error {
	appsToPack, err := getInstalledAppsForPack()
	if err != nil {
		return err
	}
	return packApps(output, mode, appsToPack)
}

func packApps(output string, mode string, appsToPack []base.BakeBuddyApp) error {
	if err := base.ValidatePackMode(mode); err != nil {
		return err
	}
	if len(appsToPack) == 0 {
		return fmt.Errorf("no installed apps found in %s", cli.BBdir)
	}

	output, err := validatePackOutput(output, appsToPack)
	if err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp("", "tezbake-pack-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	outputFile, err := os.CreateTemp(filepath.Dir(output), ".tezbake-pack-*.zip")
	if err != nil {
		return err
	}
	defer outputFile.Close()
	defer os.Remove(outputFile.Name())

	writer := zip.NewWriter(outputFile)
	defer writer.Close()

	metadata := tezbakePackMetadata{
		Version: tezbakePackFormatVersion,
		Mode:    mode,
		Apps:    make([]tezbakePackedApp, 0, len(appsToPack)),
	}

	packCtx := &base.PackContext{Mode: mode}

	for _, app := range appsToPack {
		appArchivePath := filepath.Join(tempDir, app.GetId()+".zip")

		exitCode, err := app.Pack(packCtx, appArchivePath)
		if err != nil {
			return fmt.Errorf("failed to pack app %s: %w", app.GetId(), err)
		}
		if exitCode != 0 {
			return fmt.Errorf("failed to pack app %s: ami pack exited with %d", app.GetId(), exitCode)
		}
		// the same check unpack runs, so a backup that cannot be restored fails now, not at restore time
		if err := base.ValidateAppArchive(appArchivePath, app.GetId()); err != nil {
			return fmt.Errorf("packed archive for %s cannot be restored: %w", app.GetId(), err)
		}

		if err := addFileToArchive(writer, path.Join("apps", app.GetId()+".zip"), appArchivePath); err != nil {
			return err
		}

		metadata.Apps = append(metadata.Apps, tezbakePackedApp{
			Id:     app.GetId(),
			Remote: app.IsRemoteApp(),
		})
	}

	metadataRaw, err := json.MarshalIndent(metadata, "", "\t")
	if err != nil {
		return err
	}
	if err := addBytesToArchive(writer, tezbakePackMetadataFile, metadataRaw, 0644); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	if err := outputFile.Sync(); err != nil {
		return err
	}
	if err := outputFile.Close(); err != nil {
		return err
	}
	if err := os.Rename(outputFile.Name(), output); err != nil {
		return err
	}

	log.Info("Pack successful", "output", output, "mode", mode, "apps", len(metadata.Apps))
	return nil
}

func validatePackOutput(output string, appsToPack []base.BakeBuddyApp) (string, error) {
	output, err := filepath.Abs(output)
	if err != nil {
		return "", err
	}
	outputDir, err := filepath.EvalSymlinks(filepath.Dir(output))
	if err != nil {
		return "", err
	}
	output = filepath.Join(outputDir, filepath.Base(output))
	for _, app := range appsToPack {
		appPath, err := filepath.EvalSymlinks(app.GetLocalPath())
		if err != nil {
			return "", err
		}
		appPath, err = filepath.Abs(appPath)
		if err != nil {
			return "", err
		}
		relative, err := filepath.Rel(appPath, output)
		if err != nil {
			return "", err
		}
		if filepath.IsLocal(relative) {
			return "", fmt.Errorf("pack output must be outside app directory %s", appPath)
		}
	}
	return output, nil
}

func init() {
	packCmd.Flags().String(PackOutputFlag, "tezbake-pack.zip", "Output path for the merged archive.")
	packCmd.Flags().String(PackModeFlag, base.PackModeFull, "Pack mode to use for every app (full/light).")
	hideFlagsInHelp(packCmd, OUTPUT_FORMAT_FLAG)

	RootCmd.AddCommand(packCmd)
}
