package cmd

import (
	"archive/zip"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps"
	"github.com/tez-capital/tezbake/apps/base"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/system"
	"github.com/tez-capital/tezbake/util"
	"go.alis.is/common/log"
)

var unpackCmd = &cobra.Command{
	Use:   "unpack",
	Short: "Unpack a merged tezbake archive",
	Long:  "Restores all tezbake apps from a merged archive created by 'tezbake pack'.",
	Run: func(cmd *cobra.Command, _ []string) {
		system.RequireElevatedUser()

		source := util.GetCommandStringFlagSD(cmd, UnpackSrcFlag, "tezbake-pack.zip")
		err := unpackInstalledApps(cmd, source)
		util.AssertEE(err, "Failed to unpack tezbake apps!", constants.ExitExternalError)
	},
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

	for _, packedApp := range metadata.Apps {
		app, ok := apps.FromId(packedApp.Id)
		if !ok {
			return fmt.Errorf("unsupported app id %q in archive", packedApp.Id)
		}

		appArchivePath := filepath.Join(tempDir, packedApp.Id+".zip")
		if err := extractArchiveFile(&archive.Reader, path.Join("apps", packedApp.Id+".zip"), appArchivePath, 0644); err != nil {
			return fmt.Errorf("failed to extract archive for %s: %w", packedApp.Id, err)
		}

		unpackCtx := &base.UnpackContext{
			RemoteOverride: getRemoteUnpackOverride(cmd, packedApp.Id),
		}
		if packedApp.Remote {
			bundle, err := loadArchivedRemoteBundle(&archive.Reader, packedApp.Id)
			if err != nil {
				return fmt.Errorf("failed to load remote bundle for %s: %w", packedApp.Id, err)
			}
			unpackCtx.RemoteBundle = bundle
		}

		exitCode, err := app.Unpack(unpackCtx, appArchivePath)
		if err != nil {
			return fmt.Errorf("failed to unpack app %s: %w", packedApp.Id, err)
		}
		if exitCode != 0 {
			return fmt.Errorf("failed to unpack app %s: ami unpack exited with %d", packedApp.Id, exitCode)
		}
	}

	log.Info("Unpack successful", "source", source, "apps", len(metadata.Apps))
	return nil
}

func getRemoteUnpackOverride(cmd *cobra.Command, appId string) base.RemoteUnpackOverride {
	switch appId {
	case apps.Node.GetId():
		return base.RemoteUnpackOverride{
			Remote:        util.GetCommandStringFlagS(cmd, NodeRemote),
			RemoteAuth:    util.GetCommandStringFlagS(cmd, NodeRemoteAuth),
			RemoteElevate: ami.RemoteElevationKind(util.GetCommandStringFlagS(cmd, NodeRemoteElevate)),
			RemotePath:    util.GetCommandStringFlagS(cmd, NodeRemotePath),
		}
	case apps.DalNode.GetId():
		return base.RemoteUnpackOverride{
			Remote:        util.GetCommandStringFlagS(cmd, DalRemote),
			RemoteAuth:    util.GetCommandStringFlagS(cmd, DalRemoteAuth),
			RemoteElevate: ami.RemoteElevationKind(util.GetCommandStringFlagS(cmd, DalRemoteElevate)),
			RemotePath:    util.GetCommandStringFlagS(cmd, DalRemotePath),
		}
	default:
		return base.RemoteUnpackOverride{}
	}
}

func init() {
	unpackCmd.Flags().String(UnpackSrcFlag, "tezbake-pack.zip", "Path to the merged archive.")
	unpackCmd.Flags().String(NodeRemote, "", "username@address[:port] override for a packed remote node app.")
	unpackCmd.Flags().String(NodeRemoteAuth, "", "pass|key:<path to key> override for a packed remote node app.")
	unpackCmd.Flags().String(NodeRemoteElevate, "", "Elevation override for a packed remote node app.")
	unpackCmd.Flags().String(NodeRemotePath, "", "Remote instance path override for a packed remote node app.")
	unpackCmd.Flags().String(DalRemote, "", "username@address[:port] override for a packed remote dal app.")
	unpackCmd.Flags().String(DalRemoteAuth, "", "pass|key:<path to key> override for a packed remote dal app.")
	unpackCmd.Flags().String(DalRemoteElevate, "", "Elevation override for a packed remote dal app.")
	unpackCmd.Flags().String(DalRemotePath, "", "Remote instance path override for a packed remote dal app.")
	hideFlagsInHelp(unpackCmd, OUTPUT_FORMAT_FLAG)

	RootCmd.AddCommand(unpackCmd)
}
