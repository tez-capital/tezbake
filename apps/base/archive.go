package base

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"time"

	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/system"
	"github.com/tez-capital/tezbake/util"
	"go.alis.is/common/log"
)

const (
	PackModeFull         = "full"
	PackModeLight        = "light"
	AmiPackedArchiveName = "app.zip"
)

var RemoteBundleFiles = []string{
	ami.LocatorFile,
	constants.PrivateKeyFile,
	constants.PublicKeyFile,
	ami.ElevationCredentialsFile,
	ami.ElevationCredentialsEncFile,
}

func ValidatePackMode(mode string) error {
	switch mode {
	case PackModeFull, PackModeLight:
		return nil
	default:
		return fmt.Errorf("unsupported mode %q", mode)
	}
}

func DefaultPack(app BakeBuddyApp, ctx *PackContext, output string) (int, error) {
	mode := PackModeFull
	if ctx != nil && ctx.Mode != "" {
		mode = ctx.Mode
	}
	if err := ValidatePackMode(mode); err != nil {
		return -1, err
	}

	localAppPath := app.GetLocalPath()
	if isRemote, locator := ami.IsRemoteApp(localAppPath); isRemote {
		return packRemoteApp(locator, output, mode)
	}
	return packLocalApp(localAppPath, output, mode)
}

func DefaultUnpack(app BakeBuddyApp, ctx *UnpackContext, source string) (int, error) {
	if ctx == nil {
		ctx = &UnpackContext{}
	}

	localAppPath := app.GetLocalPath()
	if ctx.RemoteBundle != nil {
		return unpackRemoteApp(localAppPath, app.GetId(), source, ctx.RemoteBundle, ctx.RemoteOverride)
	}
	if ctx.RemoteOverride.Active() {
		return -1, fmt.Errorf("remote override flags were provided for local app %s", app.GetId())
	}
	return unpackLocalApp(localAppPath, source)
}

func LoadRemoteBundle(app BakeBuddyApp) (*RemoteBundle, error) {
	return LoadRemoteBundleFromPath(app.GetLocalPath())
}

func LoadRemoteBundleFromPath(localAppPath string) (*RemoteBundle, error) {
	locator, err := ami.LoadRemoteLocator(localAppPath)
	if err != nil {
		return nil, err
	}

	bundle := &RemoteBundle{
		Locator: locator,
		Files:   make(map[string]RemoteBundleFile),
	}

	for _, fileName := range RemoteBundleFiles {
		if fileName == ami.LocatorFile {
			continue
		}

		fullPath := filepath.Join(localAppPath, fileName)
		info, err := os.Stat(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				if slices.Contains([]string{constants.PrivateKeyFile, constants.PublicKeyFile}, fileName) {
					return nil, fmt.Errorf("missing required remote file %s", fullPath)
				}
				continue
			}
			return nil, err
		}

		raw, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, err
		}

		bundle.Files[fileName] = RemoteBundleFile{
			Data: raw,
			Mode: info.Mode(),
		}
	}

	return bundle, nil
}

func packLocalApp(localAppPath string, output string, mode string) (int, error) {
	args := []string{"pack", "--output=" + output}
	if mode == PackModeLight {
		args = append(args, "--light")
	}

	return ami.Execute(localAppPath, args...)
}

func packRemoteApp(locator *ami.RemoteConfiguration, output string, mode string) (int, error) {
	session, err := locator.OpenAppRemoteSession()
	if err != nil {
		return -1, err
	}
	defer session.Close()

	remoteArchivePath := path.Join("/tmp", fmt.Sprintf("tezbake-pack-%s-%d.zip", locator.App, time.Now().UnixNano()))
	defer func() {
		if err := session.RemoveRemoteFile(remoteArchivePath); err != nil {
			log.Warn("Failed to remove remote pack archive", "app", locator.App, "path", remoteArchivePath, "error", err)
		}
	}()

	args := []string{"pack", "--output=" + remoteArchivePath}
	if mode == PackModeLight {
		args = append(args, "--light")
	}

	exitCode, err := session.ForwardAmiExecute(locator.GetRemoteAppPath(), args...)
	if err != nil {
		return exitCode, err
	}

	if err := session.DownloadFile(remoteArchivePath, output, 0644); err != nil {
		return exitCode, err
	}

	return exitCode, nil
}

func unpackLocalApp(localAppPath string, archivePath string) (int, error) {
	if err := os.MkdirAll(localAppPath, 0755); err != nil {
		return -1, err
	}
	if err := removeLocalRemoteArtifacts(localAppPath); err != nil {
		return -1, err
	}

	localArchivePath := filepath.Join(localAppPath, AmiPackedArchiveName)
	if err := copyFile(archivePath, localArchivePath, 0644); err != nil {
		return -1, err
	}
	defer os.Remove(localArchivePath)

	exitCode, err := ami.Execute(localAppPath, "unpack")
	if err != nil || exitCode != 0 {
		return exitCode, err
	}

	userName, err := getLocalAppUser(localAppPath)
	if err == nil && userName != "" {
		util.ChownR(userName, localAppPath)
	}
	return 0, nil
}

func unpackRemoteApp(localAppPath string, appId string, archivePath string, bundle *RemoteBundle, override RemoteUnpackOverride) (int, error) {
	locator, err := restoreRemoteBundle(localAppPath, appId, bundle, override)
	if err != nil {
		return -1, err
	}

	if override.Active() {
		if err := ami.PrepareRemote(localAppPath, locator, override.RemoteAuth); err != nil {
			return -1, fmt.Errorf("failed to prepare remote for %s: %w", appId, err)
		}
	}
	if err := ami.EnsureRemoteAppDirectory(locator); err != nil {
		return -1, fmt.Errorf("failed to prepare remote app directory for %s: %w", appId, err)
	}

	session, err := locator.OpenAppRemoteSession()
	if err != nil {
		return -1, err
	}
	defer session.Close()

	remoteArchivePath := path.Join(locator.GetRemoteAppPath(), AmiPackedArchiveName)
	defer func() {
		if err := session.RemoveRemoteFile(remoteArchivePath); err != nil {
			log.Warn("Failed to remove remote unpack archive", "app", appId, "path", remoteArchivePath, "error", err)
		}
	}()

	if err := session.UploadFile(archivePath, remoteArchivePath, 0644); err != nil {
		return -1, err
	}

	exitCode, err := session.ForwardAmiExecute(locator.GetRemoteAppPath(), "unpack")
	if err != nil || exitCode != 0 {
		return exitCode, err
	}

	if locator.LocalUsername != "" {
		util.ChownR(locator.LocalUsername, localAppPath)
	}
	return 0, nil
}

func restoreRemoteBundle(localAppPath string, appId string, bundle *RemoteBundle, override RemoteUnpackOverride) (*ami.RemoteConfiguration, error) {
	if bundle == nil || bundle.Locator == nil {
		return nil, fmt.Errorf("missing remote locator")
	}

	if err := os.MkdirAll(localAppPath, 0755); err != nil {
		return nil, err
	}
	if err := removeLocalRemoteArtifacts(localAppPath); err != nil {
		return nil, err
	}
	if err := writeRemoteCredentialArtifacts(localAppPath, bundle); err != nil {
		return nil, err
	}

	locator := *bundle.Locator
	locator.App = appId
	locator.ElevationCredentialsDirectory = localAppPath
	locator.PrivateKey = filepath.Join(localAppPath, constants.PrivateKeyFile)
	locator.PublicKey = filepath.Join(localAppPath, constants.PublicKeyFile)

	if override.Remote != "" {
		connection := system.GetRemoteConnectionDetails(override.Remote)
		locator.Username = connection.Username
		locator.Host = connection.Host
		locator.Port = connection.Port
	}
	if override.RemoteElevate != "" {
		locator.Elevate = override.RemoteElevate
	}
	if override.RemotePath != "" {
		locator.InstancePath = override.RemotePath
	}

	if override.Active() {
		updated := ami.WriteRemoteLocator(localAppPath, &locator, true)
		return updated, nil
	}

	for _, keyFile := range []string{constants.PrivateKeyFile, constants.PublicKeyFile} {
		file, ok := bundle.Files[keyFile]
		if !ok {
			return nil, fmt.Errorf("missing %s in archive", keyFile)
		}
		if err := os.WriteFile(filepath.Join(localAppPath, keyFile), file.Data, file.Mode); err != nil {
			return nil, err
		}
	}

	if err := writeRemoteLocator(localAppPath, &locator); err != nil {
		return nil, err
	}
	return &locator, nil
}

func writeRemoteCredentialArtifacts(localAppPath string, bundle *RemoteBundle) error {
	for _, fileName := range []string{ami.ElevationCredentialsFile, ami.ElevationCredentialsEncFile} {
		file, ok := bundle.Files[fileName]
		if !ok {
			continue
		}
		if err := os.WriteFile(filepath.Join(localAppPath, fileName), file.Data, file.Mode); err != nil {
			return err
		}
	}
	return nil
}

func writeRemoteLocator(localAppPath string, locator *ami.RemoteConfiguration) error {
	raw, err := json.MarshalIndent(locator, "", "\t")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(localAppPath, ami.LocatorFile), raw, 0644)
}

func removeLocalRemoteArtifacts(localAppPath string) error {
	for _, fileName := range RemoteBundleFiles {
		err := os.Remove(filepath.Join(localAppPath, fileName))
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func getLocalAppUser(localAppPath string) (string, error) {
	def, _, err := ami.FindAppDefinition(localAppPath)
	if err != nil {
		return "", err
	}
	userName, _ := def["user"].(string)
	return userName, nil
}

func copyFile(sourcePath string, destinationPath string, mode os.FileMode) error {
	src, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(destinationPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}

	return dst.Chmod(mode)
}
