package cmd

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/apps"
	"github.com/tez-capital/tezbake/apps/base"
)

const (
	PackModeFlag      = "mode"
	PackOutputFlag    = "output"
	UnpackSrcFlag     = "source"
	UnpackConfirmFlag = "confirm"

	tezbakePackFormatVersion = 1
	tezbakePackMetadataFile  = "__tezbake_packed_metadata.json"
	archiveSmallFileLimit    = 1 << 20
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

func getInstalledAppsForPack() ([]base.BakeBuddyApp, error) {
	result := make([]base.BakeBuddyApp, 0, len(apps.All))
	for _, app := range apps.All {
		installed, err := ami.IsAppInstalledS(app.GetLocalPath())
		if err != nil {
			return nil, fmt.Errorf("failed to check installation of app %s: %w", app.GetId(), err)
		}
		if installed {
			result = append(result, app)
		}
	}
	return result, nil
}

func loadTezbakePackMetadata(reader *zip.Reader) (*tezbakePackMetadata, error) {
	raw, _, err := readSmallArchiveFile(reader, tezbakePackMetadataFile)
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
	if err := base.ValidatePackMode(metadata.Mode); err != nil {
		return nil, err
	}
	if len(metadata.Apps) == 0 {
		return nil, fmt.Errorf("archive contains no apps")
	}

	seenApps := make(map[string]struct{}, len(metadata.Apps))
	for _, app := range metadata.Apps {
		if _, ok := apps.FromId(app.Id); !ok {
			return nil, fmt.Errorf("unsupported app id %q in archive", app.Id)
		}
		if _, seen := seenApps[app.Id]; seen {
			return nil, fmt.Errorf("duplicate app id %q in archive", app.Id)
		}
		seenApps[app.Id] = struct{}{}
	}

	return &metadata, nil
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

func readSmallArchiveFile(reader *zip.Reader, archivePath string) ([]byte, os.FileMode, error) {
	file, err := findArchiveFile(reader, archivePath)
	if err != nil {
		return nil, 0, err
	}
	if file.UncompressedSize64 > archiveSmallFileLimit {
		return nil, 0, fmt.Errorf("archive entry %s exceeds %d bytes", archivePath, archiveSmallFileLimit)
	}

	fileReader, err := file.Open()
	if err != nil {
		return nil, 0, err
	}
	defer fileReader.Close()

	data, err := io.ReadAll(io.LimitReader(fileReader, archiveSmallFileLimit+1))
	if err != nil {
		return nil, 0, err
	}
	if len(data) > archiveSmallFileLimit {
		return nil, 0, fmt.Errorf("archive entry %s exceeds %d bytes", archivePath, archiveSmallFileLimit)
	}
	return data, file.Mode(), nil
}

func extractArchiveFile(reader *zip.Reader, archivePath string, destination string, mode os.FileMode) error {
	file, err := findArchiveFile(reader, archivePath)
	if err != nil {
		return err
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	return dst.Close()
}

func findArchiveFile(reader *zip.Reader, archivePath string) (*zip.File, error) {
	var found *zip.File
	for _, file := range reader.File {
		if file.Name == archivePath {
			if found != nil {
				return nil, fmt.Errorf("duplicate archive entry %s", archivePath)
			}
			if !file.Mode().IsRegular() {
				return nil, fmt.Errorf("archive entry %s is not a regular file", archivePath)
			}
			found = file
		}
	}
	if found == nil {
		return nil, os.ErrNotExist
	}
	return found, nil
}
