package ami

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/tez-capital/tezbake/cli"
	"github.com/tez-capital/tezbake/constants"
	sshKey "github.com/tez-capital/tezbake/ssh"
	"github.com/tez-capital/tezbake/system"
	"github.com/tez-capital/tezbake/util"

	"github.com/AlecAivazis/survey/v2"
	"github.com/pkg/sftp"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

const (
	BAKER_KEY_HASH_REMOTE_VAR          = "BAKER_KEY_HASH"
	LocatorFile                 string = "locator.json"
	ElevationCredentialsFile    string = "elevate.json"
	ElevationCredentialsEncFile string = "elevate.enc.json"
)

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

func (config *RemoteConfiguration) GetElevationCredentials() (*RemoteElevateCredentials, error) {
	if config.Elevate == REMOTE_ELEVATION_NONE {
		return &RemoteElevateCredentials{Kind: REMOTE_ELEVATION_NONE}, nil
	}

	if config.ElevationCredentials != nil {
		return config.ElevationCredentials, nil
	}

	if credentials, ok := elevationCredentialsCache[config.ElevationCredentialsDirectory]; ok {
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
		prompt := &survey.Password{
			Message: fmt.Sprintf("Enter password to unlock credentials for elevation (%s):", config.App),
		}

		encFileData, err := os.ReadFile(encPath)
		if err != nil {
			return nil, err
		}

		var decData []byte
		// try to decrypt with cached password
		for _, password := range elevationCredentialsPasswordCache {
			decData, err = tryDecrypt(password, encFileData)
			if err == nil {
				break
			}
		}

		if decData == nil {
			err = survey.AskOne(prompt, &password)
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
		elevationCredentialsCache[config.ElevationCredentialsDirectory] = &credentials
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
		elevationCredentialsCache[config.ElevationCredentialsDirectory] = &credentials
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
	if options.DoNotCheckForLocator {
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
	util.AssertEE(os.MkdirAll(appDir, os.ModePerm), "Failed to create node directory!", constants.ExitIOError)

	bbKeyPair := GetAppKeyPair(appDir, reset)
	err := os.WriteFile(rc.PublicKey, []byte(strings.Trim(string(bbKeyPair.PublicKey), " \n")), 0644)
	util.AssertE(err, "Failed to write public key!")
	err = os.WriteFile(rc.PrivateKey, bbKeyPair.PrivateKey, 0600)
	util.AssertE(err, "Failed to write private key!")

	serializedRemoteConfiguration, err := json.MarshalIndent(rc, "", "\t")
	util.AssertEE(err, "Failed to serialize remote app locator!", constants.ExitSerializationFailed)
	remoteConfigurationPath := path.Join(appDir, LocatorFile)
	util.AssertEE(os.WriteFile(remoteConfigurationPath, serializedRemoteConfiguration, 0644), "Failed to write remote app locator!", constants.ExitIOError)

	remoteLocatorsCache[appDir] = rc // cache config
}

func WriteRemoteElevationCredentials(appDir string, config *RemoteConfiguration, credentials *RemoteElevateCredentials) {
	if config.Elevate == REMOTE_ELEVATION_NONE {
		log.Tracef("No elevation required for '%s', skipping saving elevate credentials", config.InstancePath)
		return
	}
	log.Trace("Writing elevation credentials of '" + appDir + "' for '" + config.InstancePath + "'...")
	serializedCredentials, err := json.MarshalIndent(credentials, "", "\t")
	util.AssertEE(err, "Failed to serialize remote elevation credentials!", constants.ExitSerializationFailed)

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
		salt := make([]byte, 16)
		_, err = rand.Read(salt)
		util.AssertE(err, "failed to generate salt")

		key := util.PrepareAESKey(password, salt)
		serializedCredentials, err = util.EncryptAES(key, serializedCredentials)
		util.AssertE(err, "failed to encrypt credentials")

		serializedCredentials = append(serializedCredentials, salt...) // append salt to the end of the file
	}
	credentialsPath := path.Join(appDir, elevationCredentialsFileName)
	util.AssertEE(os.WriteFile(credentialsPath, serializedCredentials, 0644), "Failed to write remote elevation credentials!", constants.ExitIOError)
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

func setupTezbakeForRemote(sshClient *ssh.Client, sftp *sftp.Client, locator *RemoteConfiguration, tagName string) {
	bbCliForRemoteFile := "tezbake-for-remote"

	credentials, err := locator.GetElevationCredentials()
	util.AssertE(err, "Failed to get elevation credentials!")

	remoteCliSource := os.Getenv("REMOTE_TEZBAKE_SOURCE")
	switch {
	case remoteCliSource != "":
		bbCliForRemoteFile = remoteCliSource
		log.Debugf("Using tezbake from '%s'", bbCliForRemoteFile)
	default:
		// download tezbake for remote
		architecture, err := getRemoteArchitecture(sshClient)
		util.AssertE(err, "Failed to get remote architecture!")

		binaryName := fmt.Sprintf("tezbake-%s", architecture)
		release, err := util.FetchGithubRelease(context.Background(), false, tagName)
		util.AssertE(err, "failed to fetch tezbake release")
		url, _, err := release.FindAsset(binaryName)
		util.AssertE(err, "failed to find tezbake asset in github release")
		if result := runSshCommand(sshClient, "tezbake --version", locator, system.RunSshCommand); strings.Contains(string(result.Stdout), release.TagName) {
			return
		}

		log.Trace(fmt.Sprintf("Downloading and installing tezbake (%s) for remote...", url))
		err = util.DownloadFile(url, bbCliForRemoteFile, false)
		util.AssertE(err, "Failed to download tezbake for the remote!")
	}

	// open tmp file in remote
	tmpBbCliPath := path.Join("/tmp", path.Base(bbCliForRemoteFile))
	bbCliFile, err := sftp.Create(tmpBbCliPath)
	util.AssertE(err, "Failed to open tezbake file on remote!")
	// write to remote
	bbCliForRemoteFileReader, err := os.Open(bbCliForRemoteFile)
	util.AssertE(err, "Failed to open downloaded tezbake!")
	bbCliFile.ReadFrom(bbCliForRemoteFileReader)
	bbCliFile.Close()
	// move file to sbin
	result := system.RunSshCommand(sshClient, "chmod +x "+tmpBbCliPath, nil)
	util.AssertE(result.Error, "Failed to activate tezbake!")
	bbcCliDst := "/usr/bin/tezbake"
	base64Cmd := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("cp %s %s", tmpBbCliPath, bbcCliDst)))
	result = system.RunPipedSshCommand(sshClient, fmt.Sprintf("%s execute --base64 %s --elevate", tmpBbCliPath, base64Cmd), credentials.ToEnvMap())
	util.AssertE(result.Error, "Failed to copy tezbake to sbin!")
	util.AssertBE(result.ExitCode == 0, "Failed to copy tezbake to sbin!", constants.ExitIOError)

	result = system.RunSshCommand(sshClient, fmt.Sprintf("rm %s", tmpBbCliPath), nil)
	util.AssertE(result.Error, "Failed to remove tezbake residue!")

	// cleanup residues
	cleanupCmd := "rm -f " + strings.Join(TEZBAKE_POSSIBLE_RESIDUES, " ")
	base64Cmd = base64.StdEncoding.EncodeToString([]byte(cleanupCmd))
	result = system.RunPipedSshCommand(sshClient, fmt.Sprintf("%s execute --base64 %s --elevate", bbcCliDst, base64Cmd), credentials.ToEnvMap())
	util.AssertE(result.Error, "Failed to remove tezbake residues!")
	util.AssertBE(result.ExitCode == 0, "Failed to remove tezbake residues!", constants.ExitIOError)

	// setup ami
	setupAmiCmd := "tezbake setup-ami --silent"
	base64Cmd = base64.StdEncoding.EncodeToString([]byte(setupAmiCmd))
	result = system.RunPipedSshCommand(sshClient, fmt.Sprintf("%s execute --base64 %s --elevate", bbcCliDst, base64Cmd), credentials.ToEnvMap())
	util.AssertE(result.Error, "Failed to setup ami!")
	util.AssertBE(result.ExitCode == 0, "Failed to setup ami!", constants.ExitIOError)
}

func SetupRemoteTezbake(appDir string, tagname string) {
	config, err := LoadRemoteLocator(appDir) // try to connect with BB keys
	util.AssertE(err, "Failed to load remote locator!")
	session, err := config.OpenAppRemoteSession()
	util.AssertE(err, "Failed to open remote session!")
	defer session.Close()

	setupTezbakeForRemote(session.sshClient, session.sftpSession, config, tagname)
}

func executePreparationStage(config *RemoteConfiguration, mode string, key []byte) {
	log.Info("Preparing remote...")
	sshClient, sftp := system.OpenSshSession(config.ToSshConnectionDetails(), mode, key)
	defer sshClient.Close()
	defer sftp.Close()

	setupTezbakeForRemote(sshClient, sftp, config, "latest")

	log.Trace("Injecting ssh keys...")
	// prepare .ssh
	err := sftp.MkdirAll("~/.ssh")
	util.AssertE(err, "Failed to prepare directory for authorized keys!")
	// read prepared pub key
	pubKey, err := os.ReadFile(config.PublicKey)
	util.AssertE(err, "Failed to locate public key!")
	// write if necessary
	result := system.RunSshCommand(sshClient, fmt.Sprintf("grep \"%s\" ~/.ssh/authorized_keys || echo \"%s\" >> ~/.ssh/authorized_keys", pubKey, pubKey), nil)
	util.AssertE(result.Error, "Failed to inject BB public key!")
	log.Info("Remote prepared!")
}

func PrepareRemote(appDir string, config *RemoteConfiguration, auth string) error {
	configuration, err := LoadRemoteLocator(appDir) // try to connect with BB keys
	if err == nil {
		session, err := configuration.OpenAppRemoteSession()
		if err == nil {
			session.Close()
			key, err := os.ReadFile(config.PrivateKey)
			if err == nil {
				executePreparationStage(config, system.SSH_MODE_KEY, key)
				return nil
			}
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

type TezbakeRemoteSession struct {
	sshClient    *ssh.Client
	sftpSession  *sftp.Client
	instancePath string
	locator      *RemoteConfiguration
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

	return &TezbakeRemoteSession{
		sshClient:    client,
		sftpSession:  sftp,
		instancePath: locator.InstancePath,
		locator:      locator,
	}, err
}

func runSshCommand(client *ssh.Client, cmd string, locator *RemoteConfiguration, fn func(*ssh.Client, string, *map[string]string) *system.SshCommandResult) *system.SshCommandResult {
	log.Debug("Entering remote land...")
	defer log.Debug("Returning to homeland...")
	log.Debug("remote executing: " + cmd)
	result := fn(client, cmd, nil)
	if result.Error != nil && result.ExitCode == constants.ExitElevationRequired {
		if locator.Elevate == REMOTE_ELEVATION_NONE {
			return result
		}
		elevationCredentials, err := locator.GetElevationCredentials()
		util.AssertEE(err, "Failed to get elevation credentials!", constants.ExitInvalidRemoteCredentials)
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
		forwardArgs = append(forwardArgs, "--path", session.instancePath)
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

	result := runSshCommand(session.sshClient, strings.Join(forwardArgs, " "), session.locator, system.RunPipedSshCommand)
	return result.ExitCode, result.Error
}

func (session *TezbakeRemoteSession) ForwardAmiExecuteGetOutput(workingDir string, args ...string) (string, int, error) {
	forwardArgs, err := session.prepareArgsForAmiForward(workingDir, args)
	if err != nil {
		return "", -1, err
	}

	result := runSshCommand(session.sshClient, strings.Join(forwardArgs, " "), session.locator, system.RunSshCommand)
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
	cmd := strings.Join(forwardArgs, " ")
	locator := session.locator

	log.Debug("remote executing: " + cmd)

	result := system.RunSshCommandWithOutputChannel(client, cmd, nil, outputChannel)
	if result.Error != nil && result.ExitCode == constants.ExitElevationRequired {
		if locator.Elevate == REMOTE_ELEVATION_NONE {
			return result.ExitCode, result.Error
		}
		elevationCredentials, err := locator.GetElevationCredentials()
		util.AssertEE(err, "Failed to get elevation credentials!", constants.ExitInvalidRemoteCredentials)
		if elevationCredentials.Kind == REMOTE_ELEVATION_NONE {
			return result.ExitCode, result.Error
		}
		result = system.RunSshCommandWithOutputChannel(client, cmd, elevationCredentials.ToEnvMap(), outputChannel)
	}
	return result.ExitCode, result.Error
}

func (session *TezbakeRemoteSession) writeFileToRemote(fullPath string, content []byte, mode os.FileMode) error {
	file, err := session.sftpSession.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(content)
	if err != nil {
		return err
	}

	err = file.Chmod(mode)
	return err
}
