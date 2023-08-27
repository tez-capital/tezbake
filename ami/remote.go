package ami

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"alis.is/bb-cli/cli"
	sshKey "alis.is/bb-cli/ssh"
	"alis.is/bb-cli/system"
	"alis.is/bb-cli/util"

	"github.com/AlecAivazis/survey/v2"
	"github.com/pkg/sftp"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

var (
	BAKER_KEY_HASH_REMOTE_VAR = "BAKER_KEY_HASH"
	REMOTE_VARS               = make(map[string]string)
)

type ERemoteElevationKind string

const (
	REMOTE_ELEVATION_NONE ERemoteElevationKind = ""
	REMOTE_ELEVATION_SU   ERemoteElevationKind = "su"
	REMOTE_ELEVATION_SUDO ERemoteElevationKind = "sudo"
)

type RemoteElevateCredentials struct {
	Kind     ERemoteElevationKind `json:"kind"`
	User     string               `json:"user"`
	Password string               `json:"password"`
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
	ElevationCredentialsDirectory string
	App                           string                    `json:"app"`
	Host                          string                    `json:"host"`
	Username                      string                    `json:"username"`
	InstancePath                  string                    `json:"path"`
	Elevate                       ERemoteElevationKind      `json:"elevate"`
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

func (config *RemoteConfiguration) GetElevationCredentials() (*RemoteElevateCredentials, error) {
	if config.Elevate == REMOTE_ELEVATION_NONE {
		return &RemoteElevateCredentials{Kind: REMOTE_ELEVATION_NONE}, nil
	}

	if config.ElevationCredentials != nil {
		return config.ElevationCredentials, nil
	}

	encPath := filepath.Join(config.ElevationCredentialsDirectory, ElevationCredentialsEncFile)
	plainPath := filepath.Join(config.ElevationCredentialsDirectory, ElevationCredentialsFile)

	if _, err := os.Stat(encPath); !os.IsNotExist(err) {
		var password string
		prompt := &survey.Password{
			Message: "Enter password to unlock credentials for elevation:",
		}
		err := survey.AskOne(prompt, &password)
		if err != nil {
			return nil, err
		}

		encData, err := os.ReadFile(encPath)
		if err != nil {
			return nil, err
		}

		key := util.PrepareAESKey(password, config.Host+config.Username)
		decData, err := util.DecryptAES(key, encData)
		if err != nil {
			return nil, err
		}

		var credentials RemoteElevateCredentials
		if err := json.Unmarshal(decData, &credentials); err != nil {
			return nil, err
		}

		credentials.Kind = config.Elevate
		config.ElevationCredentials = &credentials
		return &credentials, nil
	}

	if _, err := os.Stat(encPath); !os.IsNotExist(err) {
		data, err := os.ReadFile(plainPath)
		if err != nil {
			return nil, err
		}

		var credentials RemoteElevateCredentials
		if err := json.Unmarshal(data, &credentials); err != nil {
			return nil, err
		}
		config.ElevationCredentials = &credentials
		return &credentials, nil
	}

	return nil, errors.New("no elevate credentials found")
}

func (config *RemoteConfiguration) ToAppKeyPair() *AppKeyPair {
	pubKey, err := os.ReadFile(config.PublicKey)
	util.AssertE(err, "Failed to read public key!")
	privKey, err := os.ReadFile(config.PrivateKey)
	util.AssertE(err, "Failed to read private key!")
	return &AppKeyPair{
		PublicKey:  pubKey,
		PrivateKey: privKey,
		IsNew:      false,
	}
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

func LoadRemoteLocator(appDir string) (*RemoteConfiguration, error) {
	if locator, ok := remoteLocatorsCache[appDir]; ok {
		return locator, nil
	}
	remoteConfiguration := RemoteConfiguration{}
	locatorFile := path.Join(appDir, LocatorFile)
	file, err := os.ReadFile(locatorFile)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(file, &remoteConfiguration)
	if err != nil {
		return nil, err
	}
	remoteConfiguration.ElevationCredentialsDirectory = appDir
	remoteLocatorsCache[appDir] = &remoteConfiguration
	return &remoteConfiguration, nil
}

func IsRemoteApp(appDir string) (bool, *RemoteConfiguration) {
	if cli.IsRemoteInstance {
		return false, nil
	}
	locator, err := LoadRemoteLocator(appDir)
	return err == nil, locator
}

func GetAppKeyPair(appDir string, reset bool) *AppKeyPair {
	var err error
	remoteConfiguration := &RemoteConfiguration{}
	if !reset {
		remoteConfiguration, err = LoadRemoteLocator(appDir)
		if err != nil {
			return GetNewAppKeyPair()
		}
	}
	privateKeyPath := remoteConfiguration.PrivateKey
	privateKey, _ := os.ReadFile(privateKeyPath)
	publicKeyPath := remoteConfiguration.PublicKey
	publicKey, _ := os.ReadFile(publicKeyPath)
	if !sshKey.IsValidSSHPrivateKey([]byte(privateKey)) || !sshKey.IsValidSSHPublicKey([]byte(publicKey)) {
		return GetNewAppKeyPair()
	}
	return &AppKeyPair{
		PublicKey:  []byte(publicKey),
		PrivateKey: []byte(privateKey),
		IsNew:      false,
	}
}

func WriteRemoteLocator(appDir string, rc *RemoteConfiguration, reset bool) {
	log.Trace("Writing locator of '" + appDir + "' for '" + rc.InstancePath + "'...")

	util.AssertEE(os.MkdirAll(appDir, os.ModePerm), "Failed to create node directory!", cli.ExitIOError)

	bbKeyPair := GetAppKeyPair(appDir, reset)
	err := os.WriteFile(rc.PublicKey, []byte(strings.Trim(string(bbKeyPair.PublicKey), " \n")), 0644)
	util.AssertE(err, "Failed to write public key!")
	err = os.WriteFile(rc.PrivateKey, bbKeyPair.PrivateKey, 0600)
	util.AssertE(err, "Failed to write private key!")

	serializedRemoteConfiguration, err := json.MarshalIndent(rc, "", "\t")
	util.AssertEE(err, "Failed to serialize remote app locator!", cli.ExitSerializationFailed)
	remoteConfigurationPath := path.Join(appDir, LocatorFile)
	util.AssertEE(os.WriteFile(remoteConfigurationPath, serializedRemoteConfiguration, 0644), "Failed to write remote app locator!", cli.ExitIOError)

	remoteLocatorsCache[appDir] = rc // cache config
}

func WriteRemoteElevationCredentials(appDir string, rc *RemoteConfiguration, credentials *RemoteElevateCredentials) {
	if rc.Elevate == REMOTE_ELEVATION_NONE {
		log.Tracef("No elevation required for '%s', skipping saving elevate credentials", rc.InstancePath)
		return
	}
	log.Trace("Writing elevation credentials of '" + appDir + "' for '" + rc.InstancePath + "'...")
	serializedCredentials, err := json.MarshalIndent(credentials, "", "\t")
	util.AssertEE(err, "Failed to serialize remote elevation credentials!", cli.ExitSerializationFailed)

	password := ""
	prompt := &survey.Password{
		Message: "Enter password to encrypt credentials for elevation:",
	}
	err = survey.AskOne(prompt, &password)
	util.AssertE(err, "failed to get password")

	elevationCredentialsFileName := ElevationCredentialsEncFile
	if password == "" {
		elevationCredentialsFileName = ElevationCredentialsFile
	} else {
		key := util.PrepareAESKey(password, rc.Host+rc.Username)
		serializedCredentials, err = util.EncryptAES(key, serializedCredentials)
		util.AssertE(err, "failed to encrypt credentials")
	}
	credentialsPath := path.Join(appDir, elevationCredentialsFileName)
	util.AssertEE(os.WriteFile(credentialsPath, serializedCredentials, 0644), "Failed to write remote elevation credentials!", cli.ExitIOError)
}

func GetRemoteArchitecture(client *ssh.Client) string {
	result := system.RunSshCommand(client, "uname -m", nil)
	util.AssertE(result.Error, "Failed to get remote cpu architecture!")
	platform := strings.Trim(string(result.Stdout), " \n")

	switch platform {
	case "x86_64":
		return "amd64"
	case "aarch64":
		return "arm64"
	default:
		return "unknown"
	}
}

func executePreparationStage(config *RemoteConfiguration, mode string, key []byte) {
	log.Info("Preparing remote...")
	sshClient, sftp := system.OpenSshSession(config.ToSshConnectionDetails(), mode, key)
	defer sshClient.Close()
	defer sftp.Close()

	credentials, err := config.GetElevationCredentials()
	util.AssertE(err, "Failed to get elevation credentials!")
	platform := GetRemoteArchitecture(sshClient)
	bbCliForRemoteFile := "bb-cli-for-remote"

	log.Trace("Downloading and installing bb-cli for remote...")
	// download bb-cli for remote locally

	remoteCliSource := os.Getenv("REMOTE_CLI_SOURCE")
	if remoteCliSource != "" {
		bbCliForRemoteFile = remoteCliSource
	} else {
		err := util.DownloadFile(fmt.Sprintf(cli.DefaultBbCliUrl, platform), bbCliForRemoteFile, false)
		util.AssertE(err, "Failed to download bb-cli for the remote!")
	}
	// open tmp file in remote
	tmpBbCliPath := path.Join("/tmp", path.Base(bbCliForRemoteFile))
	bbCliFile, err := sftp.Create(tmpBbCliPath)
	util.AssertE(err, "Failed to open bb-cli file on remote!")
	// write to remote
	bbCliForRemoteFileReader, err := os.Open(bbCliForRemoteFile)
	util.AssertE(err, "Failed to open downloaded bb-cli!")
	bbCliFile.ReadFrom(bbCliForRemoteFileReader)
	bbCliFile.Close()
	// move file to sbin
	result := system.RunSshCommand(sshClient, "chmod +x "+tmpBbCliPath, nil)
	util.AssertE(result.Error, "Failed to activate bb-cli!")
	bbcCliDst := "/usr/sbin/bb-cli"
	base64Cmd := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("cp %s %s", tmpBbCliPath, bbcCliDst)))
	result = system.RunPipedSshCommand(sshClient, fmt.Sprintf("%s execute --base64 %s --elevate", tmpBbCliPath, base64Cmd), credentials.ToEnvMap())
	util.AssertE(result.Error, "Failed to copy bb-cli to sbin!")
	util.AssertBE(result.ExitCode == 0, "Failed to copy bb-cli to sbin!", cli.ExitIOError)

	result = system.RunSshCommand(sshClient, fmt.Sprintf("rm %s", tmpBbCliPath), nil)
	util.AssertE(result.Error, "Failed to remove bb-cli residue!")

	log.Trace("Injecting ssh keys...")
	// prepare .ssh
	err = sftp.MkdirAll("~/.ssh")
	util.AssertE(err, "Failed to prepare directory for authorized keys!")
	// read prepared pub key
	pubKey, err := os.ReadFile(config.PublicKey)
	util.AssertE(err, "Failed to locate public key!")
	// write if necessary
	result = system.RunSshCommand(sshClient, fmt.Sprintf("grep \"%s\" ~/.ssh/authorized_keys || echo \"%s\" >> ~/.ssh/authorized_keys", pubKey, pubKey), nil)
	util.AssertE(result.Error, "Failed to inject BB public key!")
	log.Info("Remote prepared!")
}

func PrepareRemote(appDir string, config *RemoteConfiguration, auth string) error {
	session, err := OpenAppRemoteSessionS(appDir)
	if err == nil { // try to connect with BB keys
		session.Close()
		key, err := os.ReadFile(config.PrivateKey)
		if err == nil {
			executePreparationStage(config, system.SSH_MODE_KEY, key)
			return nil
		}
	}
	r, _ := regexp.Compile("^key:(?P<key>.*)")
	if r.MatchString(auth) {
		matches := r.FindStringSubmatch(auth)
		keyFile := matches[r.SubexpIndex("key")]
		key, err := os.ReadFile(keyFile)
		if err != nil {
			return fmt.Errorf("%s - %s", "failed to load ssh key", err.Error())
		}
		executePreparationStage(config, system.SSH_MODE_KEY, key)
		return nil
	}
	executePreparationStage(config, system.SSH_MODE_PASS, []byte{})
	return nil
}

type AppRemoteSession struct {
	sshClient    *ssh.Client
	sftpSession  *sftp.Client
	instancePath string
	locator      *RemoteConfiguration
}

func (session *AppRemoteSession) Close() {
	session.sshClient.Close()
}

func OpenAppRemoteSessionS(appDir string) (*AppRemoteSession, error) {
	configuration, err := LoadRemoteLocator(appDir)

	if err != nil {
		return nil, err
	}
	return configuration.OpenAppRemoteSessionS()
}

func OpenAppRemoteSession(appDir string) *AppRemoteSession {
	session, err := OpenAppRemoteSessionS(appDir)
	util.AssertE(err, "Failed to load remote app locator!")
	return session
}

func (locator *RemoteConfiguration) OpenAppRemoteSessionS() (*AppRemoteSession, error) {
	keys := locator.ToAppKeyPair()
	client, sftp, err := system.OpenSshSessionS(locator.ToSshConnectionDetails(), system.SSH_MODE_KEY, keys.PrivateKey)

	return &AppRemoteSession{
		sshClient:    client,
		sftpSession:  sftp,
		instancePath: locator.InstancePath,
		locator:      locator,
	}, err
}

func filterNonPassableArgs(args []string) []string {
	filteredArgs := []string{}

	prefixes := []string{
		"--remote-",
		"--user",
	}

	for i := 0; i < len(args); i++ {
		skip := false
		for _, prefix := range prefixes {
			if strings.HasPrefix(args[i], prefix) {
				if !strings.Contains(args[i], "=") {
					i++
				}
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		filteredArgs = append(filteredArgs, args[i])
	}

	return filteredArgs
}

func keepJustRootCmdArgs(args []string) []string {
	if len(args) < 1 {
		return args
	}

	var i int
	for i = 1; i < len(args); i++ { // we skip first which is command
		if strings.HasPrefix(args[i], "-") {
			if !strings.Contains(args[i], "=") {
				i++
			}
			continue
		}
		break
	}

	return args[:i]
}

func (session *AppRemoteSession) prepareArgsForProxy(passthrough bool) []string {
	args := os.Args
	proxyArgs := make([]string, 0)
	proxyArgs = append(proxyArgs, "bb-cli")
	if passthrough {
		proxyArgs = append(proxyArgs, "--output-format=text")
	} else {
		proxyArgs = append(proxyArgs, "--output-format=json")
	}

	remoteVarsWithValues := make([]string, 0)
	for k, v := range REMOTE_VARS {
		remoteVarsWithValues = append(remoteVarsWithValues, fmt.Sprintf("%s=%s", k, v))
	}
	if len(remoteVarsWithValues) > 0 {
		proxyArgs = append(proxyArgs, fmt.Sprintf("--remote-instance-vars=%s", strings.Join(remoteVarsWithValues, ";")))
	}

	proxyArgs = append(proxyArgs, "--remote-instance")
	if session.instancePath != "" { // strip --path/-p
		strippedArgs := make([]string, 0)
		skip := false
		for _, v := range args {
			if skip {
				skip = false
				continue
			}
			if v == "-p" || v == "--path" {
				skip = true
				continue
			}
			if strings.HasPrefix(v, "-p=") || strings.HasPrefix(v, "--path=") {
				continue
			}
			strippedArgs = append(strippedArgs, v)
		}
		args = strippedArgs
		proxyArgs = append(proxyArgs, "--path", session.instancePath)
	}

	args = filterNonPassableArgs(args)

	proxyArgs = append(proxyArgs, args[1:]...)
	return proxyArgs
}

func runSshCommand(client *ssh.Client, cmd string, locator *RemoteConfiguration, fn func(*ssh.Client, string, *map[string]string) *system.SshCommandResult) *system.SshCommandResult {
	log.Debug("Entering remote land...")
	defer log.Debug("Returning to homeland...")
	log.Debug("remote executing: " + cmd)
	result := fn(client, cmd, nil)
	if result.Error != nil && result.ExitCode == cli.ExitElevationRequired {
		if locator.Elevate == REMOTE_ELEVATION_NONE {
			return result
		}
		elevationCredentials, err := locator.GetElevationCredentials()
		if err != nil {
			return result
		}
		if elevationCredentials.Kind == REMOTE_ELEVATION_NONE {
			return result
		}
		result = fn(client, cmd, elevationCredentials.ToEnvMap())
	}
	return result
}

func (session *AppRemoteSession) ProxyToRemoteApp() (int, error) {
	args := session.prepareArgsForProxy(true)
	result := runSshCommand(session.sshClient, strings.Join(args, " "), session.locator, system.RunPipedSshCommand)
	return result.ExitCode, result.Error
}

func (session *AppRemoteSession) ProxyToRemoteAppGetOutput() (string, int, error) {
	args := session.prepareArgsForProxy(false)

	result := runSshCommand(session.sshClient, strings.Join(args, " "), session.locator, system.RunSshCommand)

	return string(result.Stdout), result.ExitCode, result.Error
}

func (session *AppRemoteSession) ProxyToRemoteAppExecuteInfo() ([]byte, int, error) {
	args := session.prepareArgsForProxy(false)

	result := runSshCommand(session.sshClient, strings.Join(args, " "), session.locator, system.RunSshCommand)

	return result.Stdout, result.ExitCode, result.Error
}

func (session *AppRemoteSession) IsRemoteModuleInstalled(id string) ([]byte, int, error) {
	args := keepJustRootCmdArgs(session.prepareArgsForProxy(false))
	args = append(args, fmt.Sprintf("--is-module-installed=%s", id))

	result := runSshCommand(session.sshClient, strings.Join(args, " "), session.locator, system.RunSshCommand)

	return result.Stdout, result.ExitCode, result.Error
}
