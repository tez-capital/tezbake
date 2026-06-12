package cmd

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
	"path"

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
	Long:  "Builds a merged archive containing all installed tezbake apps.",
	Run: func(cmd *cobra.Command, _ []string) {
		system.RequireElevatedUser()

		mode, err := validatePackMode(util.GetCommandStringFlagSD(cmd, PackModeFlag, base.PackModeFull))
		util.AssertEE(err, "Invalid pack mode!", constants.ExitInvalidArgs)

		output := util.GetCommandStringFlagSD(cmd, PackOutputFlag, "tezbake-pack.zip")
		err = packInstalledApps(output, mode)
		util.AssertEE(err, "Failed to pack tezbake apps!", constants.ExitExternalError)
	},
}

func packInstalledApps(output string, mode string) (resultErr error) {
	appsToPack := getInstalledAppsForPack()
	if len(appsToPack) == 0 {
		return fmt.Errorf("no installed apps found in %s", cli.BBdir)
	}

	tempDir, err := os.MkdirTemp("", "tezbake-pack-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	outputFile, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer outputFile.Close()
	defer func() {
		if resultErr != nil {
			_ = os.Remove(output)
		}
	}()

	writer := zip.NewWriter(outputFile)
	defer func() {
		closeErr := writer.Close()
		if resultErr == nil {
			resultErr = closeErr
		}
	}()

	metadata := tezbakePackMetadata{
		Version: tezbakePackFormatVersion,
		Mode:    mode,
		Apps:    make([]tezbakePackedApp, 0, len(appsToPack)),
	}

	packCtx := &base.PackContext{Mode: mode}

	for _, app := range appsToPack {
		appArchive, err := os.CreateTemp(tempDir, app.GetId()+"-*.zip")
		if err != nil {
			return err
		}
		appArchivePath := appArchive.Name()
		if err := appArchive.Close(); err != nil {
			return err
		}
		if err := os.Remove(appArchivePath); err != nil {
			return err
		}

		exitCode, err := app.Pack(packCtx, appArchivePath)
		if err != nil {
			return fmt.Errorf("failed to pack app %s: %w", app.GetId(), err)
		}
		if exitCode != 0 {
			return fmt.Errorf("failed to pack app %s: ami pack exited with %d", app.GetId(), exitCode)
		}

		if app.IsRemoteApp() {
			bundle, err := base.LoadRemoteBundle(app)
			if err != nil {
				return fmt.Errorf("failed to load remote bundle for %s: %w", app.GetId(), err)
			}
			if err := addRemoteBundleToArchive(writer, app.GetId(), bundle); err != nil {
				return fmt.Errorf("failed to add remote bundle for %s: %w", app.GetId(), err)
			}
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

	log.Info("Pack successful", "output", output, "mode", mode, "apps", len(metadata.Apps))
	return nil
}

func init() {
	packCmd.Flags().String(PackOutputFlag, "tezbake-pack.zip", "Output path for the merged archive.")
	packCmd.Flags().String(PackModeFlag, base.PackModeFull, "Pack mode to use for every app (full/light).")
	hideFlagsInHelp(packCmd, OUTPUT_FORMAT_FLAG)

	RootCmd.AddCommand(packCmd)
}
