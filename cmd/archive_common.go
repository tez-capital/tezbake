package cmd

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps"
	"github.com/tez-capital/tezbake/apps/base"
)

const (
	PackModeFlag   = "mode"
	PackOutputFlag = "output"
	UnpackSrcFlag  = "source"

	NodeRemotePath = "node-remote-path"
	DalRemotePath  = "dal-remote-path"

	tezbakePackFormatVersion = 1
	tezbakePackMetadataFile  = "__tezbake_packed_metadata.json"
)

type tezbakePackedApp struct {
	Id     string `json:"id"`
	Remote bool   `json:"remote"`
}

type tezbakePackMetadata struct {
	Version int                `json:"version"`
	Mode    string             `json:"mode"`
	Apps    []tezbakePackedApp `json:"apps"`
}

func hideFlagsInHelp(cmd *cobra.Command, flagNames ...string) {
	defaultHelpFunc := cmd.HelpFunc()
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		hiddenFlags := make([]*pflag.Flag, 0, len(flagNames))
		for _, flagName := range flagNames {
			flag := cmd.InheritedFlags().Lookup(flagName)
			if flag == nil || flag.Hidden {
				continue
			}
			flag.Hidden = true
			hiddenFlags = append(hiddenFlags, flag)
		}

		defaultHelpFunc(cmd, args)

		for _, flag := range hiddenFlags {
			flag.Hidden = false
		}
	})
}

func validatePackMode(mode string) (string, error) {
	if err := base.ValidatePackMode(mode); err != nil {
		return "", err
	}
	return mode, nil
}

func getInstalledAppsForPack() []base.BakeBuddyApp {
	result := make([]base.BakeBuddyApp, 0, len(apps.All))
	for _, app := range apps.All {
		if app.IsInstalled() {
			result = append(result, app)
		}
	}
	return result
}

func loadTezbakePackMetadata(reader *zip.Reader) (*tezbakePackMetadata, error) {
	raw, _, err := readArchiveFile(reader, tezbakePackMetadataFile)
	if err != nil {
		return nil, err
	}

	var metadata tezbakePackMetadata
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return nil, err
	}
	if metadata.Version != tezbakePackFormatVersion {
		return nil, fmt.Errorf("unsupported tezbake archive version %d", metadata.Version)
	}
	if _, err := validatePackMode(metadata.Mode); err != nil {
		return nil, err
	}

	seenApps := make(map[string]struct{}, len(metadata.Apps))
	for _, app := range metadata.Apps {
		if !isSupportedAppId(app.Id) {
			return nil, fmt.Errorf("unsupported app id %q in archive", app.Id)
		}
		if _, seen := seenApps[app.Id]; seen {
			return nil, fmt.Errorf("duplicate app id %q in archive", app.Id)
		}
		seenApps[app.Id] = struct{}{}
	}

	return &metadata, nil
}

func addRemoteBundleToArchive(writer *zip.Writer, appId string, bundle *base.RemoteBundle) error {
	if bundle == nil || bundle.Locator == nil {
		return fmt.Errorf("missing remote locator")
	}

	locatorRaw, err := json.MarshalIndent(bundle.Locator, "", "\t")
	if err != nil {
		return err
	}
	if err := addBytesToArchive(writer, path.Join("remote", appId, ami.LocatorFile), locatorRaw, 0644); err != nil {
		return err
	}

	for _, fileName := range base.RemoteBundleFiles {
		if fileName == ami.LocatorFile {
			continue
		}

		file, ok := bundle.Files[fileName]
		if !ok {
			continue
		}
		if err := addBytesToArchive(writer, path.Join("remote", appId, fileName), file.Data, file.Mode); err != nil {
			return err
		}
	}

	return nil
}

func loadArchivedRemoteBundle(reader *zip.Reader, appId string) (*base.RemoteBundle, error) {
	locatorRaw, _, err := readArchiveFile(reader, path.Join("remote", appId, ami.LocatorFile))
	if err != nil {
		return nil, err
	}

	var locator ami.RemoteConfiguration
	if err := json.Unmarshal(locatorRaw, &locator); err != nil {
		return nil, err
	}

	bundle := &base.RemoteBundle{
		Locator: &locator,
		Files:   make(map[string]base.RemoteBundleFile),
	}

	for _, fileName := range base.RemoteBundleFiles {
		if fileName == ami.LocatorFile {
			continue
		}

		raw, mode, err := readArchiveFile(reader, path.Join("remote", appId, fileName))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		bundle.Files[fileName] = base.RemoteBundleFile{
			Data: raw,
			Mode: mode,
		}
	}

	return bundle, nil
}

func addFileToArchive(writer *zip.Writer, archivePath string, sourcePath string) error {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = archivePath
	header.Method = zip.Deflate

	fileWriter, err := writer.CreateHeader(header)
	if err != nil {
		return err
	}

	fileReader, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer fileReader.Close()

	_, err = io.Copy(fileWriter, fileReader)
	return err
}

func addBytesToArchive(writer *zip.Writer, archivePath string, content []byte, mode os.FileMode) error {
	header := &zip.FileHeader{
		Name:   archivePath,
		Method: zip.Deflate,
	}
	header.SetMode(mode)

	fileWriter, err := writer.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = fileWriter.Write(content)
	return err
}

func readArchiveFile(reader *zip.Reader, archivePath string) ([]byte, os.FileMode, error) {
	file := findArchiveFile(reader, archivePath)
	if file == nil {
		return nil, 0, os.ErrNotExist
	}

	fileReader, err := file.Open()
	if err != nil {
		return nil, 0, err
	}
	defer fileReader.Close()

	data, err := io.ReadAll(fileReader)
	if err != nil {
		return nil, 0, err
	}
	return data, file.Mode(), nil
}

func extractArchiveFile(reader *zip.Reader, archivePath string, destination string, mode os.FileMode) error {
	data, _, err := readArchiveFile(reader, archivePath)
	if err != nil {
		return err
	}
	return os.WriteFile(destination, data, mode)
}

func findArchiveFile(reader *zip.Reader, archivePath string) *zip.File {
	for _, file := range reader.File {
		if file.Name == archivePath {
			return file
		}
	}
	return nil
}

func isSupportedAppId(appId string) bool {
	_, ok := apps.FromId(appId)
	return ok
}
