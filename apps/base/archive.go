package base

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/hjson/hjson-go/v4"
	"github.com/tez-capital/tezbake/ami"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/util"
	"go.alis.is/common/log"
)

const (
	PackModeFull         = "full"
	PackModeLight        = "light"
	AmiPackedArchiveName = "app.zip"

	amiPackedMetadataFile = "__ami_packed_metadata.json"
	appArchiveConfigLimit = 1 << 20
)

func ValidatePackMode(mode string) error {
	switch mode {
	case PackModeFull, PackModeLight:
		return nil
	default:
		return fmt.Errorf("unsupported mode %q", mode)
	}
}

// restartIfRunning starts the app again after a stop; start errors are joined to *err.
func restartIfRunning(app BakeBuddyApp, wasRunning bool, exitCode *int, err *error) {
	if !wasRunning {
		return
	}
	code, startErr := app.Start()
	if startErr == nil && code != 0 {
		startErr = fmt.Errorf("failed to start %s: exit code %d", app.GetId(), code)
	}
	if startErr != nil {
		*err = errors.Join(*err, startErr)
		if *exitCode == 0 {
			*exitCode = code
		}
	}
}

// pack

func DefaultPack(app BakeBuddyApp, ctx *PackContext, output string) (exitCode int, err error) {
	mode := PackModeFull
	if ctx != nil && ctx.Mode != "" {
		mode = ctx.Mode
	}
	if err := ValidatePackMode(mode); err != nil {
		return -1, err
	}
	output, err = filepath.Abs(output)
	if err != nil {
		return -1, err
	}

	// an unknown status could mean packing a live app
	wasRunning, err := IsAnyServiceStatus(app, "running")
	if err != nil {
		return -1, fmt.Errorf("failed to get %s status: %w", app.GetId(), err)
	}
	if wasRunning {
		if exitCode, err := app.Stop(); err != nil {
			return exitCode, err
		}
	}
	defer restartIfRunning(app, wasRunning, &exitCode, &err)

	localAppPath := app.GetLocalPath()
	if isRemote, locator := ami.IsRemoteApp(localAppPath); isRemote {
		return packRemoteApp(locator, output, mode)
	}
	return ami.Execute(localAppPath, amiPackArgs(output, mode)...)
}

func amiPackArgs(output string, mode string) []string {
	args := []string{"pack", "--output=" + output}
	if mode == PackModeLight {
		args = append(args, "--light")
	}
	return args
}

func packRemoteApp(locator *ami.RemoteConfiguration, output string, mode string) (exitCode int, resultErr error) {
	session, err := locator.OpenAppRemoteSession()
	if err != nil {
		return -1, err
	}
	defer session.Close()

	remoteArchivePath, err := session.CreateRemoteTempArchive()
	if err != nil {
		return -1, err
	}
	defer func() {
		if err := session.RemoveRemoteTempArchive(remoteArchivePath); err != nil && resultErr == nil {
			resultErr = fmt.Errorf("failed to remove remote archive %s: %w", remoteArchivePath, err)
		}
	}()

	exitCode, err = session.ForwardAmiExecute(locator.GetRemoteAppPath(), amiPackArgs(remoteArchivePath, mode)...)
	if err != nil || exitCode != 0 {
		return exitCode, err
	}
	if err := session.MakeRemoteArchiveReadable(remoteArchivePath); err != nil {
		return -1, err
	}
	if err := session.DownloadFile(remoteArchivePath, output, 0600); err != nil {
		return -1, err
	}
	return 0, nil
}

// unpack

// restoreIdentity keeps the installed target's id, user and wiring keys on top of the archived definition.
func restoreIdentity(restored, installed map[string]any, wiringKeys []string) {
	for _, key := range []string{"id", "user"} {
		if value, ok := installed[key]; ok {
			restored[key] = value
		} else {
			delete(restored, key)
		}
	}
	if len(wiringKeys) == 0 {
		return
	}
	installedConfiguration, _ := installed["configuration"].(map[string]any)
	restoredConfiguration, ok := restored["configuration"].(map[string]any)
	if !ok {
		restoredConfiguration = map[string]any{}
		restored["configuration"] = restoredConfiguration
	}
	for _, key := range wiringKeys {
		if value, ok := installedConfiguration[key]; ok {
			restoredConfiguration[key] = value
		} else {
			delete(restoredConfiguration, key)
		}
	}
}

// DefaultUnpack overlays the archive onto an installed app, like upgrade; the caller validates the archive.
// The installation keeps its id, user, remote connection and the given configuration wiringKeys.
func DefaultUnpack(app BakeBuddyApp, source string, wiringKeys ...string) (exitCode int, err error) {
	source, err = filepath.Abs(source)
	if err != nil {
		return -1, err
	}
	localAppPath := app.GetLocalPath()
	installed, _, err := ami.FindAppDefinition(localAppPath)
	if err != nil {
		return -1, fmt.Errorf("failed to read installed %s definition: %w", app.GetId(), err)
	}

	wasRunning, err := IsAnyServiceStatus(app, "running")
	if err != nil {
		return -1, fmt.Errorf("failed to get %s status: %w", app.GetId(), err)
	}
	if wasRunning {
		if exitCode, err := app.Stop(); err != nil {
			return exitCode, err
		}
	}
	defer func() {
		// a failed restore may leave a half-merged app; keep it stopped like a failed node upgrade
		if err == nil && exitCode == 0 {
			restartIfRunning(app, wasRunning, &exitCode, &err)
		}
	}()

	isRemote, locator := ami.IsRemoteApp(localAppPath)
	if isRemote {
		exitCode, err = unpackRemoteApp(locator, app.GetId(), source)
	} else {
		exitCode, err = unpackLocalApp(localAppPath, source)
	}
	if err != nil || exitCode != 0 {
		return exitCode, err
	}

	restored, defPath, err := ami.FindAppDefinition(localAppPath)
	if err != nil {
		return -1, fmt.Errorf("failed to read restored %s definition: %w", app.GetId(), err)
	}
	restoreIdentity(restored, installed, wiringKeys)
	if err = ami.WriteAppDefinition(localAppPath, restored, filepath.Base(defPath)); err != nil {
		return -1, err
	}
	// a remote locator forwards setup to the remote host
	if exitCode, err = ami.Execute(localAppPath, "setup", "--configure"); err != nil || exitCode != 0 {
		return exitCode, err
	}
	owner := ""
	if isRemote {
		// remote setup does not set local permissions
		owner = locator.LocalUsername
	} else {
		owner, _ = restored["user"].(string)
	}
	if owner != "" {
		if _, err := util.ChownRS(owner, localAppPath); err != nil {
			return -1, fmt.Errorf("failed to set ownership of %s: %w", localAppPath, err)
		}
	}
	return 0, nil
}

func unpackLocalApp(localAppPath string, archivePath string) (int, error) {
	localArchivePath := filepath.Join(localAppPath, AmiPackedArchiveName)
	if err := copyFile(archivePath, localArchivePath, 0600); err != nil {
		return -1, err
	}
	defer os.Remove(localArchivePath)
	return ami.Execute(localAppPath, "unpack")
}

func unpackRemoteApp(locator *ami.RemoteConfiguration, appId string, archivePath string) (int, error) {
	session, err := locator.OpenAppRemoteSession()
	if err != nil {
		return -1, err
	}
	defer session.Close()

	remoteAppPath := locator.GetRemoteAppPath()
	remoteArchivePath := path.Join(remoteAppPath, AmiPackedArchiveName)
	if err := session.UploadFile(archivePath, remoteArchivePath, 0600); err != nil {
		return -1, err
	}
	defer func() {
		if err := session.RemoveRemoteFile(remoteArchivePath); err != nil {
			log.Warn("Failed to remove remote unpack archive", "app", appId, "path", remoteArchivePath, "error", err)
		}
	}()

	return session.ForwardAmiExecute(remoteAppPath, "unpack")
}

func copyFile(sourcePath string, destinationPath string, mode os.FileMode) error {
	src, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer src.Close()

	// a new temp file plus rename never writes through a planted symlink
	dst, err := os.CreateTemp(filepath.Dir(destinationPath), ".tezbake-copy-*")
	if err != nil {
		return err
	}
	defer os.Remove(dst.Name())
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	if err := dst.Chmod(mode); err != nil {
		return err
	}
	if err := dst.Close(); err != nil {
		return err
	}
	return os.Rename(dst.Name(), destinationPath)
}

// validation

// ValidateAppArchive checks paths and all file contents before AMI can overwrite an app.
func ValidateAppArchive(source, expectedAppID string) error {
	archive, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer archive.Close()

	entries := make(map[string]*zip.File, len(archive.File))
	for _, file := range archive.File {
		name := strings.TrimSuffix(file.Name, "/")
		if name == "." || !fs.ValidPath(name) || strings.ContainsAny(name, "\\\x00") || (len(name) > 1 && name[1] == ':') {
			return fmt.Errorf("unsafe app archive path %q", file.Name)
		}
		rootName, _, _ := strings.Cut(name, "/")
		if strings.EqualFold(rootName, AmiPackedArchiveName) || strings.EqualFold(rootName, ami.LocatorFile) {
			return fmt.Errorf("reserved app archive path %q", file.Name)
		}
		mode := file.Mode()
		if (mode.Type() != 0 && mode.Type() != fs.ModeDir) || mode.IsDir() != strings.HasSuffix(file.Name, "/") {
			return fmt.Errorf("unsupported app archive file type for %q", file.Name)
		}
		if mode.IsDir() && file.UncompressedSize64 != 0 {
			return fmt.Errorf("app archive directory %q contains data", file.Name)
		}
		if _, exists := entries[name]; exists {
			return fmt.Errorf("duplicate app archive path %q", file.Name)
		}
		entries[name] = file
	}
	for name := range entries {
		for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
			if entry := entries[parent]; entry != nil && !entry.Mode().IsDir() {
				return fmt.Errorf("app archive file %q is a parent of %q", parent, name)
			}
		}
	}

	metadataFile := entries[amiPackedMetadataFile]
	if metadataFile == nil || !metadataFile.Mode().IsRegular() {
		return fmt.Errorf("missing AMI archive metadata %s", amiPackedMetadataFile)
	}
	// AMI always loads app.hjson first, including when an app.json also exists.
	configurationFile := entries["app.hjson"]
	if configurationFile == nil {
		configurationFile = entries["app.json"]
	}
	if configurationFile == nil || !configurationFile.Mode().IsRegular() {
		return fmt.Errorf("app archive is missing app.hjson or app.json")
	}

	var metadataRaw, configurationRaw []byte
	for _, file := range archive.File {
		capture := file == metadataFile || file == configurationFile
		raw, err := validateAppArchiveEntry(file, capture)
		if err != nil {
			return err
		}
		if file == metadataFile {
			metadataRaw = raw
		} else if file == configurationFile {
			configurationRaw = raw
		}
	}
	var metadata struct {
		Version int `json:"VERSION"`
		Options any `json:"options"`
	}
	if err := hjson.Unmarshal(metadataRaw, &metadata); err != nil {
		return fmt.Errorf("invalid AMI archive metadata: %w", err)
	}
	if metadata.Version != 1 {
		return fmt.Errorf("unsupported AMI archive version %d", metadata.Version)
	}
	// AMI serializes an empty Lua options table as [], rather than {}.
	switch options := metadata.Options.(type) {
	case map[string]any:
		// AMI reruns unpack with these options; another source bypasses this validation.
		if source, exists := options["source"]; exists && source != AmiPackedArchiveName {
			return fmt.Errorf("AMI archive metadata cannot select another unpack source")
		}
	case []any:
	default:
		return fmt.Errorf("invalid AMI archive metadata options")
	}
	var configuration map[string]any
	if err := hjson.Unmarshal(configurationRaw, &configuration); err != nil {
		return fmt.Errorf("invalid %s in app archive: %w", configurationFile.Name, err)
	}
	if configuration == nil {
		return fmt.Errorf("invalid %s in app archive: expected an object", configurationFile.Name)
	}
	if id, _ := configuration["id"].(string); id == "" {
		return fmt.Errorf("app archive has an invalid id")
	}
	appID, err := restoreAppIDFromType(configuration)
	if err != nil {
		return err
	}
	if appID != expectedAppID {
		return fmt.Errorf("app archive type belongs to %s, expected %s", appID, expectedAppID)
	}
	return nil
}

func validateAppArchiveEntry(file *zip.File, capture bool) ([]byte, error) {
	if capture && file.UncompressedSize64 > appArchiveConfigLimit {
		return nil, fmt.Errorf("app archive entry %q exceeds %d bytes", file.Name, appArchiveConfigLimit)
	}
	reader, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open app archive entry %q: %w", file.Name, err)
	}
	var raw []byte
	if capture {
		raw, err = io.ReadAll(io.LimitReader(reader, appArchiveConfigLimit+1))
		if err == nil && len(raw) > appArchiveConfigLimit {
			err = fmt.Errorf("exceeds %d bytes", appArchiveConfigLimit)
		}
	} else {
		_, err = io.Copy(io.Discard, reader)
	}
	if err = errors.Join(err, reader.Close()); err != nil {
		return nil, fmt.Errorf("invalid app archive entry %q: %w", file.Name, err)
	}
	return raw, nil
}

func restoreAppIDFromType(definition map[string]any) (string, error) {
	appType, _ := definition["type"].(string)
	if typed, ok := definition["type"].(map[string]any); ok {
		appType, _ = typed["id"].(string)
	}
	appType, _, _ = strings.Cut(appType, "@")
	for appID, prefix := range map[string]string{
		constants.NodeAppId:   "xtz.node",
		constants.DalAppId:    "xtz.dal",
		constants.SignerAppId: "xtz.signer",
		constants.PeakAppId:   "tzc.tezpeak",
		constants.TezpayAppId: "tzc.tezpay",
	} {
		if appType == prefix || strings.HasPrefix(appType, prefix+".") {
			return appID, nil
		}
	}
	return "", fmt.Errorf("unsupported app archive type %q", appType)
}
