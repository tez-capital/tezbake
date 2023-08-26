package ami

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"

	"alis.is/bb-cli/cli"
	sshKey "alis.is/bb-cli/ssh"
	"alis.is/bb-cli/system"
	"alis.is/bb-cli/util"

	"github.com/pkg/sftp"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

var (
	BAKER_KEY_HASH_REMOTE_VAR = "BAKER_KEY_HASH"
	REMOTE_VARS               = make(map[string]string)
)

type RemoteElevateCredentials struct {
	REMOTE_SU_USER   string
	REMOTE_SU_PASS   string
	REMOTE_SUDO_PASS string
}

func (creds *RemoteElevateCredentials) ToEnvMap() *map[string]string {
	return &map[string]string{
		RemoteSuUser:   creds.REMOTE_SU_USER,
		RemoteSuPass:   creds.REMOTE_SU_PASS,
		RemoteSudoPass: creds.REMOTE_SUDO_PASS,
	}
}

type RemoteConfiguration struct {
	App                  string
	Host                 string
	Username             string
	InstancePath         string
	Elevate              string
	PrivateKey           string
	PublicKey            string
	Port                 string
	ElevationCredentials RemoteElevateCredentials
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

func LoadRemoteLocator(appDir string) (*RemoteConfiguration, error) {
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

	switch config.Elevate {
	case SuElevate:
		// TODO:
	case SudoElevate:
		// TODO:
	}
	platform := GetRemoteArchitecture(sshClient)
	bbCliForRemoteFile := "bb-cli-for-remote"

	log.Trace("Downloading and installing bb-cli for remote...")
	// download bb-cli for remote locally
	err := util.DownloadFile(fmt.Sprintf(cli.DefaultBbCliUrl, platform), bbCliForRemoteFile, false)
	util.AssertE(err, "Failed to download bb-cli for the remote!")
	// open tmp file in remote
	tmpBbCliPath := path.Join("/tmp", bbCliForRemoteFile)
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
	result = system.RunSshCommand(sshClient, fmt.Sprintf("mv '%s' /usr/sbin/bb-cli", tmpBbCliPath), nil)
	util.AssertE(result.Error, "Failed to copy bb-cli to sbin!")

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
	}, err
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
			if skip == true {
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
	proxyArgs = append(proxyArgs, args[1:]...)
	return proxyArgs
}

func (session *AppRemoteSession) ProxyToRemoteApp(env *map[string]string) (int, error) {
	args := session.prepareArgsForProxy(true)
	log.Info("Entering remote land...")
	result := system.RunPipedSshCommand(session.sshClient, strings.Join(args, " "), env)
	// TODO: proper exit code
	log.Info("Returning to homeland...")
	if result.Error != nil {
		return -1, result.Error
	}
	return 0, nil
}

func (session *AppRemoteSession) ProxyToRemoteAppGetOutput(env *map[string]string) (string, int, error) {
	args := session.prepareArgsForProxy(false)
	log.Info("Entering remote land...")
	result := system.RunSshCommand(session.sshClient, strings.Join(args, " "), env)
	// TODO: proper exit code
	log.Info("Returning to homeland...")
	if result.Error != nil {
		return "", -1, result.Error
	}
	return string(result.Stdout), 0, nil
}

func (session *AppRemoteSession) ProxyToRemoteAppExecuteInfo(env *map[string]string) ([]byte, int, error) {
	args := session.prepareArgsForProxy(false)
	result := system.RunSshCommand(session.sshClient, strings.Join(args, " "), env)
	// TODO: proper exit code
	if result.Error != nil {
		return []byte{}, -1, result.Error
	}
	return result.Stdout, 0, nil
}

func (session *AppRemoteSession) IsRemoteModuleInstalled(id string, env *map[string]string) ([]byte, int, error) {
	args := session.prepareArgsForProxy(false)
	args = append(args, fmt.Sprintf("--is-module-installed=%s", id))
	result := system.RunSshCommand(session.sshClient, strings.Join(args, " "), env)
	if result.Error != nil {
		return []byte{}, -1, result.Error
	}
	return result.Stdout, 0, nil
}
