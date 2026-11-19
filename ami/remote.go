package ami

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"github.com/tez-capital/tezbake/cli"
	"github.com/tez-capital/tezbake/constants"
	sshKey "github.com/tez-capital/tezbake/ssh"
	"github.com/tez-capital/tezbake/system"
	"github.com/tez-capital/tezbake/util"
	"go.alis.is/common/log"

	"github.com/kballard/go-shellquote"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const (
	BAKER_KEY_HASH_REMOTE_VAR          = "BAKER_KEY_HASH"
	LocatorFile                 string = "locator.json"
	ElevationCredentialsFile    string = "elevate.json"
	ElevationCredentialsEncFile string = "elevate.enc.json"
)

// ErrNotRemoteApp means the app directory has no locator, so the app is local.
var ErrNotRemoteApp = errors.New("not a remote app")

var TEZBAKE_POSSIBLE_RESIDUES = []string{
	"/usr/sbin/tezbake",
}

var (
	REMOTE_VARS                       = make(map[string]string)
	elevationCredentialsCache         = make(map[string]*RemoteElevateCredentials)
	elevationCredentialsPasswordCache = make([]string, 0, 2)
)

type RemoteElevationKind string

const (
	REMOTE_ELEVATION_NONE RemoteElevationKind = ""
	REMOTE_ELEVATION_SU   RemoteElevationKind = "su"
	REMOTE_ELEVATION_SUDO RemoteElevationKind = "sudo"
)

type RemoteElevateCredentials struct {
	Kind     RemoteElevationKind `json:"kind"`
	User     string              `json:"user"`
	Password string              `json:"password"`
}

func (creds *RemoteElevateCredentials) ToEnvMap() *map[string]string {
	switch creds.Kind {
	case REMOTE_ELEVATION_NONE:
		return &map[string]string{}
	case REMOTE_ELEVATION_SU:
		return &map[string]string{
			"ELEVATION_KIND":     string(creds.Kind),
			"ELEVATION_USER":     creds.User,
			"ELEVATION_PASSWORD": creds.Password,
		}
	case REMOTE_ELEVATION_SUDO:
		return &map[string]string{
			"ELEVATION_KIND":     string(creds.Kind),
			"ELEVATION_PASSWORD": creds.Password,
		}
	}
	return &map[string]string{}
}

type RemoteConfiguration struct {
	ElevationCredentialsDirectory string                    `json:"elevation_credentials_directory"`
	App                           string                    `json:"app"`
	Host                          string                    `json:"host"`
	Username                      string                    `json:"username"`
	LocalUsername                 string                    `json:"local_username"`
	InstancePath                  string                    `json:"path"`
	Elevate                       RemoteElevationKind       `json:"elevate"`
	PrivateKey                    string                    `json:"privateKey"`
	PublicKey                     string                    `json:"publicKey"`
	Port                          string                    `json:"port"`
	ElevationCredentials          *RemoteElevateCredentials `json:"-"`
}

// Fills empty values with values from other config
func (config *RemoteConfiguration) PopulateWith(populationSource *RemoteConfiguration) {
	util.AssignStructFieldsIfEmpty(config, populationSource)
}

func (config *RemoteConfiguration) ToSshConnectionDetails() *system.SshConnectionDetails {
	return &system.SshConnectionDetails{
		Username: config.Username,
		Host:     config.Host,
		Port:     config.Port,
	}
}

func (config *RemoteConfiguration) GetRemoteAppPath() string {
	return path.Join(config.InstancePath, config.App)
}

func (config *RemoteConfiguration) GetElevationCredentials() (*RemoteElevateCredentials, error) {
	if config.Elevate == REMOTE_ELEVATION_NONE {
		return &RemoteElevateCredentials{Kind: REMOTE_ELEVATION_NONE}, nil
	}

	if config.ElevationCredentials != nil {
		return config.ElevationCredentials, nil
	}

	cacheKey := remoteConfigurationCacheKey(config.ElevationCredentialsDirectory)
	if credentials, ok := elevationCredentialsCache[cacheKey]; ok {
		return credentials, nil
	}

	encPath := filepath.Join(config.ElevationCredentialsDirectory, ElevationCredentialsEncFile)
	plainPath := filepath.Join(config.ElevationCredentialsDirectory, ElevationCredentialsFile)

	tryDecrypt := func(password string, data []byte) ([]byte, error) {
		if len(data) < 16 {
			return nil, errors.New("data is too short to decrypt")
		}
		salt := data[len(data)-16:]
		encData := data[:len(data)-16]

		key := util.PrepareAESKey(password, salt)
		decData, err := util.DecryptAES(key, encData)
		if err != nil {
			return nil, err
		}
		return decData, nil
	}

	if _, err := os.Stat(encPath); !os.IsNotExist(err) {
		var password string

		encFileData, err := os.ReadFile(encPath)
		if err != nil {
			return nil, err
		}

		var decData []byte
		// try to decrypt with cached password
		for _, cachedPassword := range elevationCredentialsPasswordCache {
			decData, err = tryDecrypt(cachedPassword, encFileData)
			if err == nil {
				password = cachedPassword
				break
			}
		}

		if decData == nil {
			password, err = util.PromptPassword(fmt.Sprintf("Enter password to unlock credentials for elevation (%s):", config.App))
			if err != nil {
				return nil, err
			}
			decData, err = tryDecrypt(password, encFileData)
			if err != nil {
				return nil, err
			}
		}
		if !slices.Contains(elevationCredentialsPasswordCache, password) {
			elevationCredentialsPasswordCache = append(elevationCredentialsPasswordCache, password)
		}

		var credentials RemoteElevateCredentials
		if err := json.Unmarshal(decData, &credentials); err != nil {
			return nil, err
		}

		credentials.Kind = config.Elevate
		config.ElevationCredentials = &credentials
		elevationCredentialsCache[cacheKey] = &credentials
		return &credentials, nil
	}

	if _, err := os.Stat(plainPath); !os.IsNotExist(err) {
		data, err := os.ReadFile(plainPath)
		if err != nil {
			return nil, err
		}

		var credentials RemoteElevateCredentials
		if err := json.Unmarshal(data, &credentials); err != nil {
			return nil, err
		}
		credentials.Kind = config.Elevate
		config.ElevationCredentials = &credentials
		elevationCredentialsCache[cacheKey] = &credentials
		return &credentials, nil
	}

	return nil, errors.New("no elevate credentials found")
}

func (config *RemoteConfiguration) ToAppKeyPair() (*AppKeyPair, error) {
	pubKey, err := os.ReadFile(config.PublicKey)
	if err != nil {
		return nil, errors.Join(errors.New("failed to read public key"), err)
	}
	privKey, err := os.ReadFile(config.PrivateKey)
	if err != nil {
		return nil, errors.Join(errors.New("failed to read private key"), err)
	}
	return &AppKeyPair{
		PublicKey:  pubKey,
		PrivateKey: privKey,
		IsNew:      false,
	}, nil
}

type AppKeyPair struct {
	PublicKey  []byte
	PrivateKey []byte
	IsNew      bool
}

func GetNewAppKeyPair() *AppKeyPair {
	generated := sshKey.GenerateBBKeys()
	return &AppKeyPair{
		PublicKey:  generated.PublicKey,
		PrivateKey: generated.PrivateKey,
		IsNew:      true,
	}
}

var (
	remoteLocatorsCache = make(map[string]*RemoteConfiguration)
)

func remoteConfigurationCacheKey(appDir string) string {
	if absolutePath, err := filepath.Abs(appDir); err == nil {
		return absolutePath
	}
	return filepath.Clean(appDir)
}

// ForgetRemoteConfiguration must follow replacement or removal of locator/key/credential files.
func ForgetRemoteConfiguration(appDir string) {
	cacheKey := remoteConfigurationCacheKey(appDir)
	if locator := remoteLocatorsCache[cacheKey]; locator != nil {
		locator.ElevationCredentials = nil
		delete(elevationCredentialsCache, remoteConfigurationCacheKey(locator.ElevationCredentialsDirectory))
	}
	delete(remoteLocatorsCache, cacheKey)
	delete(elevationCredentialsCache, cacheKey)
}

func LoadRemoteLocator(appDir string) (*RemoteConfiguration, error) {
	cacheKey := remoteConfigurationCacheKey(appDir)
	if locator, ok := remoteLocatorsCache[cacheKey]; ok {
		return locator, nil
	}
	remoteConfiguration := RemoteConfiguration{}
	locatorFile := path.Join(appDir, LocatorFile)
	file, err := os.ReadFile(locatorFile)
	if err != nil {
		// Only a truly absent locator means local; a dangling symlink or unreadable file stays an error.
		if _, statErr := os.Lstat(locatorFile); os.IsNotExist(statErr) {
			return nil, ErrNotRemoteApp
		}
		return nil, err
	}
	err = json.Unmarshal(file, &remoteConfiguration)
	if err != nil {
		return nil, err
	}
	remoteConfiguration.ElevationCredentialsDirectory = cacheKey
	remoteLocatorsCache[cacheKey] = &remoteConfiguration
	return &remoteConfiguration, nil
}

// loadAppLocator is LoadRemoteLocator honoring options.DoNotCheckForLocator.
// Callers run locally only on ErrNotRemoteApp; any other error must not redirect work locally.
func loadAppLocator(appDir string) (*RemoteConfiguration, error) {
	if options.DoNotCheckForLocator {
		return nil, ErrNotRemoteApp
	}
	return LoadRemoteLocator(appDir)
}

func IsRemoteApp(appDir string) (bool, *RemoteConfiguration) {
	locator, err := loadAppLocator(appDir)
	return err == nil, locator
}

func GetAppKeyPair(appDir string, rekey bool) *AppKeyPair {
	if rekey {
		return GetNewAppKeyPair()
	}
	remoteConfiguration, err := LoadRemoteLocator(appDir)
	if err != nil {
		return GetNewAppKeyPair()
	}
	privateKeyPath := remoteConfiguration.PrivateKey
	privateKey, _ := os.ReadFile(privateKeyPath)
	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return GetNewAppKeyPair()
	}
	return &AppKeyPair{
		PublicKey:  ssh.MarshalAuthorizedKey(signer.PublicKey()),
		PrivateKey: privateKey,
		IsNew:      false,
	}
}

func WriteRemoteLocator(appDir string, rc *RemoteConfiguration, rekey bool) *RemoteConfiguration {
	log.Trace("Writing locator...", "app_dir", appDir, "instance_path", rc.InstancePath)
	util.AssertEE(os.MkdirAll(appDir, os.ModePerm), "Failed to create node directory!", constants.ExitIOError)

	bbKeyPair := GetAppKeyPair(appDir, rekey)
	err := WriteFileAtomic(rc.PublicKey, []byte(strings.Trim(string(bbKeyPair.PublicKey), " \n")), 0644)
	util.AssertE(err, "Failed to write public key!")
	err = WriteFileAtomic(rc.PrivateKey, bbKeyPair.PrivateKey, 0600)
	util.AssertE(err, "Failed to write private key!")

	serializedRemoteConfiguration, err := json.MarshalIndent(rc, "", "\t")
	util.AssertEE(err, "Failed to serialize remote app locator!", constants.ExitSerializationFailed)
	remoteConfigurationPath := path.Join(appDir, LocatorFile)
	util.AssertEE(WriteFileAtomic(remoteConfigurationPath, serializedRemoteConfiguration, 0644), "Failed to write remote app locator!", constants.ExitIOError)

	credentials := rc.ElevationCredentials
	ForgetRemoteConfiguration(appDir)
	rc.ElevationCredentialsDirectory = remoteConfigurationCacheKey(appDir)
	rc.ElevationCredentials = credentials
	remoteLocatorsCache[remoteConfigurationCacheKey(appDir)] = rc // cache config
	return rc
}

func WriteRemoteElevationCredentials(appDir string, config *RemoteConfiguration, credentials *RemoteElevateCredentials) {
	if config.Elevate == REMOTE_ELEVATION_NONE {
		log.Trace("No elevation required, skipping saving elevate credentials", "instance_path", config.InstancePath)
		return
	}
	password := util.RequirePasswordE("Enter password to encrypt credentials for elevation:", "failed to get password", constants.ExitInternalError)
	util.AssertEE(SaveRemoteElevationCredentials(appDir, config, credentials, password), "Failed to save remote elevation credentials!", constants.ExitIOError)
}

// Replace configuration files atomically so old permissions and symlinks cannot expose or redirect writes.
func WriteFileAtomic(filePath string, data []byte, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(filePath), ".tezbake-secret-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	// like os.WriteFile, a replacement keeps the owner (e.g. the signer user reading idkey); a planted symlink does not count
	if info, err := os.Lstat(filePath); err == nil && info.Mode().IsRegular() {
		if owner, ok := info.Sys().(*syscall.Stat_t); ok {
			if err := tmp.Chown(int(owner.Uid), int(owner.Gid)); err != nil {
				tmp.Close()
				return err
			}
		}
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), filePath)
}

// SaveRemoteElevationCredentials atomically replaces credentials and removes the other encoding.
func SaveRemoteElevationCredentials(appDir string, config *RemoteConfiguration, credentials *RemoteElevateCredentials, encryptionPassword string) error {
	if config == nil || credentials == nil {
		return errors.New("missing remote elevation configuration or credentials")
	}
	if config.Elevate != REMOTE_ELEVATION_SU && config.Elevate != REMOTE_ELEVATION_SUDO {
		return fmt.Errorf("invalid remote elevation kind %q", config.Elevate)
	}
	log.Trace("Writing elevation credentials...", "app_dir", appDir, "instance_path", config.InstancePath)
	updatedCredentials := *credentials
	updatedCredentials.Kind = config.Elevate
	serializedCredentials, err := json.MarshalIndent(&updatedCredentials, "", "\t")
	if err != nil {
		return err
	}

	elevationCredentialsFileName := ElevationCredentialsEncFile
	otherFileName := ElevationCredentialsFile
	if encryptionPassword == "" {
		elevationCredentialsFileName = ElevationCredentialsFile
		otherFileName = ElevationCredentialsEncFile
	} else {
		salt := make([]byte, 16)
		_, err = rand.Read(salt)
		if err != nil {
			return err
		}

		key := util.PrepareAESKey(encryptionPassword, salt)
		serializedCredentials, err = util.EncryptAES(key, serializedCredentials)
		if err != nil {
			return err
		}

		serializedCredentials = append(serializedCredentials, salt...) // append salt to the end of the file
	}
	if err := WriteFileAtomic(filepath.Join(appDir, elevationCredentialsFileName), serializedCredentials, 0600); err != nil {
		return err
	}
	ForgetRemoteConfiguration(appDir)
	config.ElevationCredentialsDirectory = remoteConfigurationCacheKey(appDir)
	config.ElevationCredentials = nil
	if err := os.Remove(filepath.Join(appDir, otherFileName)); err != nil && !os.IsNotExist(err) {
		return err
	}
	config.ElevationCredentials = &updatedCredentials
	elevationCredentialsCache[remoteConfigurationCacheKey(appDir)] = &updatedCredentials
	if encryptionPassword != "" && !slices.Contains(elevationCredentialsPasswordCache, encryptionPassword) {
		elevationCredentialsPasswordCache = append(elevationCredentialsPasswordCache, encryptionPassword)
	}
	return nil
}

func getRemoteArchitecture(client *ssh.Client) (string, error) {
	result := system.RunSshCommand(client, "uname -m", nil)
	if result.Error != nil {
		return "", errors.Join(errors.New("failed to get remote cpu architecture"), result.Error)
	}
	platform := strings.Trim(string(result.Stdout), " \n")

	result = system.RunSshCommand(client, "uname -s", nil)
	if result.Error != nil {
		return "", errors.Join(errors.New("failed to get remote OS"), result.Error)
	}
	os := strings.Trim(string(result.Stdout), " \n")

	switch platform {
	case "x86_64":
		platform = "amd64"
	case "aarch64":
		platform = "arm64"
	default:
		return "", errors.New("unsupported architecture: " + platform)
	}

	switch os {
	case "Linux":
		return fmt.Sprintf("%s-%s", "linux", platform), nil
	case "Darwin":
		return fmt.Sprintf("%s-%s", "macos", platform), nil
	default:
		return "", errors.New("unsupported OS: " + os)
	}
}

func setupTezbakeForRemote(sshClient *ssh.Client, sftp *sftp.Client, locator *RemoteConfiguration, tagName string) error {
	credentials, err := locator.GetElevationCredentials()
	if err != nil {
		return fmt.Errorf("failed to get elevation credentials: %w", err)
	}

	bbCliForRemoteFile := os.Getenv("REMOTE_TEZBAKE_SOURCE")
	switch {
	case bbCliForRemoteFile != "":
		log.Debug("Using tezbake from", "remote_file", bbCliForRemoteFile)
	default:
		// download tezbake for remote
		architecture, err := getRemoteArchitecture(sshClient)
		if err != nil {
			return err
		}

		binaryName := fmt.Sprintf("tezbake-%s", architecture)
		release, err := util.FetchGithubRelease(context.Background(), false, tagName)
		if err != nil {
			return fmt.Errorf("failed to fetch tezbake release: %w", err)
		}
		url, _, err := release.FindAsset(binaryName)
		if err != nil {
			return fmt.Errorf("failed to find remote tezbake asset: %w", err)
		}
		if result := runSshCommand(sshClient, "tezbake --version", locator, system.RunSshCommand); strings.Contains(string(result.Stdout), release.TagName) {
			return nil
		}

		localTemporaryFile, err := os.CreateTemp("", "tezbake-for-remote-*")
		if err != nil {
			return err
		}
		bbCliForRemoteFile = localTemporaryFile.Name()
		defer os.Remove(bbCliForRemoteFile)
		if err := localTemporaryFile.Close(); err != nil {
			return err
		}
		log.Trace("Downloading and installing tezbake for remote...", "url", url)
		if err := util.DownloadFile(url, bbCliForRemoteFile, false); err != nil {
			return fmt.Errorf("failed to download remote tezbake: %w", err)
		}
	}

	session := &TezbakeRemoteSession{sshClient: sshClient, sftpSession: sftp, locator: locator}
	temporaryArchivePath, err := session.CreateRemoteTempArchive()
	if err != nil {
		return err
	}
	defer session.RemoveRemoteTempArchive(temporaryArchivePath)
	tmpBbCliPath := path.Join(path.Dir(temporaryArchivePath), "tezbake")
	if err := session.UploadFile(bbCliForRemoteFile, tmpBbCliPath, 0755); err != nil {
		return fmt.Errorf("failed to upload remote tezbake: %w", err)
	}

	bbcCliDst := "/usr/bin/tezbake"
	base64Cmd := base64.StdEncoding.EncodeToString([]byte(shellquote.Join("cp", "--", tmpBbCliPath, bbcCliDst)))
	result := system.RunPipedSshCommand(sshClient, shellquote.Join(tmpBbCliPath, "execute", "--base64", base64Cmd, "--elevate"), credentials.ToEnvMap())
	if result.Error != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to install remote tezbake (exit %d): %w", result.ExitCode, result.Error)
	}

	// cleanup residues
	cleanupCmd := shellquote.Join(append([]string{"rm", "-f", "--"}, TEZBAKE_POSSIBLE_RESIDUES...)...)
	base64Cmd = base64.StdEncoding.EncodeToString([]byte(cleanupCmd))
	result = system.RunPipedSshCommand(sshClient, shellquote.Join(bbcCliDst, "execute", "--base64", base64Cmd, "--elevate"), credentials.ToEnvMap())
	if result.Error != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to remove remote tezbake residues (exit %d): %w", result.ExitCode, result.Error)
	}

	// setup ami
	setupAmiCmd := "tezbake setup-ami --silent"
	base64Cmd = base64.StdEncoding.EncodeToString([]byte(setupAmiCmd))
	result = system.RunPipedSshCommand(sshClient, shellquote.Join(bbcCliDst, "execute", "--base64", base64Cmd, "--elevate"), credentials.ToEnvMap())
	if result.Error != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to setup remote ami (exit %d): %w", result.ExitCode, result.Error)
	}
	return nil
}

func SetupRemoteTezbake(appDir string, tagname string) {
	config, err := LoadRemoteLocator(appDir) // try to connect with BB keys
	util.AssertE(err, "Failed to load remote locator!")
	session, err := config.OpenAppRemoteSession()
	util.AssertE(err, "Failed to open remote session!")
	defer session.Close()

	util.AssertE(setupTezbakeForRemote(session.sshClient, session.sftpSession, config, tagname), "Failed to setup remote tezbake!")
}

// umask 077 creates ~/.ssh as 0700 and authorized_keys as 0600
const addAuthorizedKeyScript = `umask 077 && mkdir -p ~/.ssh && touch ~/.ssh/authorized_keys &&
	{ grep -qxF -- "$1" ~/.ssh/authorized_keys || printf '%s\n' "$1" >> ~/.ssh/authorized_keys; }`

func executePreparationStage(config *RemoteConfiguration, mode string, key []byte) error {
	log.Info("Preparing remote...")
	sshClient, sftp, err := system.OpenSshSessionS(config.ToSshConnectionDetails(), mode, key)
	if err != nil {
		return fmt.Errorf("failed to connect to remote: %w", err)
	}
	defer sshClient.Close()
	defer sftp.Close()

	if err := setupTezbakeForRemote(sshClient, sftp, config, "latest"); err != nil {
		return err
	}

	log.Trace("Injecting ssh keys...")
	// read prepared pub key
	pubKey, err := os.ReadFile(config.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to locate public key: %w", err)
	}
	parsedPublicKey, _, _, _, err := ssh.ParseAuthorizedKey(pubKey)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}
	// write if necessary; the key arrives as $1, so it is never interpolated into the script
	authorizedKey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(parsedPublicKey)))
	result := system.RunSshCommand(sshClient, shellquote.Join("sh", "-c", addAuthorizedKeyScript, "sh", authorizedKey), nil)
	if result.Error != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to install SSH public key (exit %d): %w", result.ExitCode, result.Error)
	}
	log.Info("Remote prepared!")
	return nil
}

func PrepareRemote(appDir string, config *RemoteConfiguration, auth string) error {
	configuration, err := LoadRemoteLocator(appDir) // try to connect with BB keys
	if err == nil {
		session, err := configuration.OpenAppRemoteSession()
		if err == nil {
			session.Close()
			key, err := os.ReadFile(config.PrivateKey)
			if err == nil {
				return executePreparationStage(config, system.SSH_MODE_KEY, key)
			}
		}
	}

	if keyFile, ok := strings.CutPrefix(auth, "key:"); ok {
		key, err := os.ReadFile(keyFile)
		if err != nil {
			return fmt.Errorf("%s - %s", "failed to load ssh key", err.Error())
		}
		return executePreparationStage(config, system.SSH_MODE_KEY, key)
	}
	return executePreparationStage(config, system.SSH_MODE_PASS, []byte{})
}

type TezbakeRemoteSession struct {
	sshClient   *ssh.Client
	sftpSession *sftp.Client
	locator     *RemoteConfiguration
}

func (session *TezbakeRemoteSession) Close() {
	session.sshClient.Close()
	session.sftpSession.Close()
}

func (locator *RemoteConfiguration) OpenAppRemoteSession() (*TezbakeRemoteSession, error) {
	keys, err := locator.ToAppKeyPair()
	if err != nil {
		return nil, err
	}
	client, sftp, err := system.OpenSshSessionS(locator.ToSshConnectionDetails(), system.SSH_MODE_KEY, keys.PrivateKey)
	if err != nil {
		return nil, err
	}

	return &TezbakeRemoteSession{
		sshClient:   client,
		sftpSession: sftp,
		locator:     locator,
	}, nil
}

func runSshCommand(client *ssh.Client, cmd string, locator *RemoteConfiguration, fn func(*ssh.Client, string, *map[string]string) *system.SshCommandResult) *system.SshCommandResult {
	log.Debug("Entering remote land...")
	defer log.Debug("Returning to homeland...")
	log.Debug("remote executing:", "cmd", cmd)
	result := fn(client, cmd, nil)
	if result.Error != nil && result.ExitCode == constants.ExitElevationRequired {
		if locator.Elevate == REMOTE_ELEVATION_NONE {
			return result
		}
		elevationCredentials, err := locator.GetElevationCredentials()
		if err != nil {
			return &system.SshCommandResult{ExitCode: -1, Error: err}
		}
		if elevationCredentials.Kind == REMOTE_ELEVATION_NONE {
			return result
		}
		result = fn(client, cmd, elevationCredentials.ToEnvMap())
	}
	return result
}

func (session *TezbakeRemoteSession) prepareArgsForAmiForward(workingDir string, args []string) ([]string, error) {
	forwardArgs := []string{"tezbake"}

	remoteVarsWithValues := make([]string, 0)
	for k, v := range REMOTE_VARS {
		remoteVarsWithValues = append(remoteVarsWithValues, fmt.Sprintf("%s=%s", k, v))
	}
	if len(remoteVarsWithValues) > 0 {
		forwardArgs = append(forwardArgs, fmt.Sprintf("--remote-instance-vars=%s", strings.Join(remoteVarsWithValues, ";")))
	}

	forwardArgs = append(forwardArgs, "execute-ami")
	if cli.ElevationRequired {
		forwardArgs = append(forwardArgs, "--elevate")
	}

	forwardArgs = append(forwardArgs, "--app", workingDir)

	jsonEncodedArgs, err := json.Marshal(args)
	if err != nil {
		return nil, errors.Join(errors.New("failed to encode ami args"), err)
	}

	encodedArgs := base64.StdEncoding.EncodeToString(jsonEncodedArgs)
	forwardArgs = append(forwardArgs, "--base64-args", encodedArgs)

	return forwardArgs, nil
}

func (session *TezbakeRemoteSession) ForwardAmiExecute(workingDir string, args ...string) (int, error) {
	forwardArgs, err := session.prepareArgsForAmiForward(workingDir, args)
	if err != nil {
		return -1, err
	}
	result := runSshCommand(session.sshClient, shellquote.Join(forwardArgs...), session.locator, system.RunPipedSshCommand)
	return result.ExitCode, result.Error
}

func (session *TezbakeRemoteSession) ForwardAmiExecuteGetOutput(workingDir string, args ...string) (string, int, error) {
	forwardArgs, err := session.prepareArgsForAmiForward(workingDir, args)
	if err != nil {
		return "", -1, err
	}

	result := runSshCommand(session.sshClient, shellquote.Join(forwardArgs...), session.locator, system.RunSshCommand)
	return string(result.Stdout), result.ExitCode, result.Error
}

// ExecuteRemoteCommand runs quoted argv through the same elevation mechanism as forwarded AMI commands.
func (session *TezbakeRemoteSession) ExecuteRemoteCommand(args ...string) (string, int, error) {
	if len(args) == 0 {
		return "", -1, errors.New("missing remote command")
	}
	encodedCommand := base64.StdEncoding.EncodeToString([]byte(shellquote.Join(args...)))
	result := runSshCommand(session.sshClient, "tezbake execute --elevate --base64 "+encodedCommand, session.locator, system.RunSshCommand)
	return string(result.Stdout), result.ExitCode, result.Error
}

func (session *TezbakeRemoteSession) GetRemoteTezbakeVersion() (string, error) {
	result := runSshCommand(session.sshClient, "tezbake --version", session.locator, system.RunSshCommand)
	if result.Error != nil {
		return "", errors.Join(errors.New("failed to get remote tezbake version"), result.Error)
	}
	if result.ExitCode != 0 {
		return "", errors.New("failed to get remote tezbake version - exit code " + fmt.Sprint(result.ExitCode))
	}
	version := strings.Trim(string(result.Stdout), " \n")
	if version == "" {
		return "", errors.New("failed to get remote tezbake version - empty output")
	}
	return version, nil
}

func (session *TezbakeRemoteSession) ForwardAmiExecuteWithOutputChannel(workingDir string, outputChannel chan<- string, args ...string) (int, error) {
	forwardArgs, err := session.prepareArgsForAmiForward(workingDir, args)
	if err != nil {
		return -1, err
	}

	log.Debug("Entering remote land...")
	defer log.Debug("Returning to homeland...")

	client := session.sshClient
	cmd := shellquote.Join(forwardArgs...)
	locator := session.locator

	log.Debug("remote executing:", "cmd", cmd)

	result := system.RunSshCommandWithOutputChannel(client, cmd, nil, outputChannel)
	if result.Error != nil && result.ExitCode == constants.ExitElevationRequired {
		if locator.Elevate == REMOTE_ELEVATION_NONE {
			return result.ExitCode, result.Error
		}
		elevationCredentials, err := locator.GetElevationCredentials()
		if err != nil {
			return -1, err
		}
		if elevationCredentials.Kind == REMOTE_ELEVATION_NONE {
			return result.ExitCode, result.Error
		}
		result = system.RunSshCommandWithOutputChannel(client, cmd, elevationCredentials.ToEnvMap(), outputChannel)
	}
	return result.ExitCode, result.Error
}

func (session *TezbakeRemoteSession) writeReaderToRemote(fullPath string, src io.Reader, mode os.FileMode) (err error) {
	// SFTP OpenFile cannot request a create-time mode. mktemp creates a private
	// inode on the destination filesystem, including when replacing a public file.
	prefix := path.Join(path.Dir(fullPath), ".tezbake-upload.")
	result := system.RunSshCommand(session.sshClient, shellquote.Join("mktemp", "--", prefix+"XXXXXXXXXX"), nil)
	if result.Error != nil {
		return fmt.Errorf("create private remote upload: %w: %s", result.Error, strings.TrimSpace(string(result.Stderr)))
	}
	stagedPath := strings.TrimSuffix(string(result.Stdout), "\n")
	if !strings.HasPrefix(stagedPath, prefix) || path.Dir(stagedPath) != path.Dir(fullPath) {
		return fmt.Errorf("unexpected path from remote mktemp: %q", stagedPath)
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, session.RemoveRemoteFile(stagedPath))
		}
	}()

	file, err := session.sftpSession.OpenFile(stagedPath, os.O_WRONLY)
	if err != nil {
		return err
	}
	if _, err = io.Copy(file, src); err == nil {
		err = file.Chmod(mode)
	}
	if err = errors.Join(err, file.Close()); err != nil {
		return err
	}
	return session.sftpSession.PosixRename(stagedPath, fullPath)
}

func (session *TezbakeRemoteSession) writeFileToRemote(fullPath string, content []byte, mode os.FileMode) error {
	return session.writeReaderToRemote(fullPath, bytes.NewReader(content), mode)
}

func (session *TezbakeRemoteSession) UploadFile(localPath string, remotePath string, mode os.FileMode) error {
	src, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer src.Close()

	return session.writeReaderToRemote(remotePath, src, mode)
}

func (session *TezbakeRemoteSession) DownloadFile(remotePath string, localPath string, mode os.FileMode) error {
	src, err := session.sftpSession.Open(remotePath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(localPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}

	if err := dst.Chmod(mode); err != nil {
		return err
	}
	return dst.Close()
}

func (session *TezbakeRemoteSession) RemoveRemoteFile(remotePath string) error {
	err := session.sftpSession.Remove(remotePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// CreateRemoteTempArchive reserves a private directory owned by the SSH user.
// The archive must remain absent: eli's zip writer cannot open an empty existing file.
func (session *TezbakeRemoteSession) CreateRemoteTempArchive() (string, error) {
	dir := path.Join("/tmp", "tezbake-pack-"+rand.Text())
	if err := session.sftpSession.Mkdir(dir); err != nil {
		return "", err
	}
	if err := session.sftpSession.Chmod(dir, 0700); err != nil {
		return "", errors.Join(err, session.sftpSession.RemoveDirectory(dir))
	}
	return path.Join(dir, "app.zip"), nil
}

func remoteTempArchiveDirectory(remotePath string) (string, error) {
	dir := path.Dir(remotePath)
	if path.Clean(remotePath) != remotePath || path.Base(remotePath) != "app.zip" || path.Dir(dir) != "/tmp" || !strings.HasPrefix(path.Base(dir), "tezbake-pack-") {
		return "", fmt.Errorf("invalid remote temporary archive path %q", remotePath)
	}
	return dir, nil
}

// MakeRemoteArchiveReadable allows SFTP to download a root-created archive inside its private directory.
func (session *TezbakeRemoteSession) MakeRemoteArchiveReadable(remotePath string) error {
	dir, err := remoteTempArchiveDirectory(remotePath)
	if err != nil {
		return err
	}
	info, err := session.sftpSession.Lstat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode().Perm() != 0700 {
		return errors.New("remote temporary archive directory is not private")
	}
	info, err = session.sftpSession.Lstat(remotePath)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("remote temporary archive is not a regular file")
	}
	_, exitCode, err := session.ExecuteRemoteCommand("chmod", "0644", "--", remotePath)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		return fmt.Errorf("failed to set remote archive permissions: exit code %d", exitCode)
	}
	return nil
}

func (session *TezbakeRemoteSession) RemoveRemoteTempArchive(remotePath string) error {
	dir, err := remoteTempArchiveDirectory(remotePath)
	if err != nil {
		return err
	}
	info, err := session.sftpSession.Lstat(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("remote temporary archive directory is not a directory")
	}
	files, err := session.sftpSession.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, file := range files {
		if err := session.RemoveRemoteFile(path.Join(dir, file.Name())); err != nil {
			return err
		}
	}
	return session.sftpSession.RemoveDirectory(dir)
}
