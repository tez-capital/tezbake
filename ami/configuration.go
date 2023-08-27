package ami

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"

	"alis.is/bb-cli/cli"
	"alis.is/bb-cli/system"
	"golang.org/x/crypto/ssh"

	"github.com/hjson/hjson-go"
	"github.com/pkg/sftp"
	log "github.com/sirupsen/logrus"
)

func findAppDefinitionRemote(sftpClient *sftp.Client, workingDir string) (map[string]interface{}, string, error) {
	for _, candidate := range AppConfigurationCandidates {
		appDefPath := path.Join(workingDir, candidate)
		appDefFile, err := sftpClient.OpenFile(appDefPath, os.O_RDONLY)
		if err == nil {
			log.Trace("App definition found in " + appDefPath)
			appDef := make(map[string]interface{})
			appDefContent, err := io.ReadAll(appDefFile)
			if err == nil {
				err = hjson.Unmarshal(appDefContent, &appDef)
				return appDef, appDefPath, err
			}
		}
	}
	return nil, "", errors.New("failed to load app configuration (no valid configuration found)")
}

func FindAppDefinition(workingDir string) (map[string]interface{}, string, error) {
	if isRemote, locator := IsRemoteApp(workingDir); isRemote {
		session, err := locator.OpenAppRemoteSessionS()
		if err != nil {
			return nil, "", err
		}
		return findAppDefinitionRemote(session.sftpSession, path.Join(locator.InstancePath, locator.App))
	}

	for _, candidate := range AppConfigurationCandidates {
		appDefPath := path.Join(workingDir, candidate)
		appDefContent, err := os.ReadFile(appDefPath)
		if err == nil {
			log.Trace("App definition found in " + appDefPath)
			appDef := make(map[string]interface{})
			err = hjson.Unmarshal(appDefContent, &appDef)
			return appDef, appDefPath, err
		}
	}
	return nil, "", errors.New("failed to load app configuration (no valid configuration found)")
}

func LoadAppDefinition(app string) (map[string]interface{}, error) {
	log.Trace("Loading '" + app + "' definition from '" + cli.BBdir + "'...")
	appDef, _, err := FindAppDefinition(app)
	if err != nil {
		return nil, err
	}
	return appDef, nil
}

func LoadAppConfiguration(app string) (map[string]interface{}, error) {
	appDef, err := LoadAppDefinition(app)
	if err != nil {
		return nil, err
	}
	return appDef["configuration"].(map[string]interface{}), nil
}

func writeAppConfigurationToRemote(sftpClient *sftp.Client, workingDir string, configuration map[string]interface{}) error {
	var appDef []byte
	var appDefPath string
	log.Tracef("Writing app configuration to remote %s...", workingDir)
	err := os.MkdirAll(workingDir, os.ModePerm)
	if err == nil {
		appDef, err = json.MarshalIndent(configuration, "", "\t")
	}
	if err != nil {
		return err
	}
	_, appDefPath, err = findAppDefinitionRemote(sftpClient, workingDir)
	if err != nil || appDefPath == "" {
		appDefPath = path.Join(workingDir, "app.json")
	}
	appDefFile, err := sftpClient.OpenFile(appDefPath, os.O_WRONLY)
	if err != nil {
		return err
	}
	defer appDefFile.Close()
	_, err = appDefFile.Write(appDef)
	if err == nil {
		err = appDefFile.Chmod(0644)
	}
	log.Tracef("App configuration written to %s", appDefPath)
	return err
}

func prepareFolderStructure(sshClient *ssh.Client, workingDir string, user string, env *map[string]string) error {
	log.Tracef("Preparing folder structure for remote %s...", workingDir)
	encodedCmd := base64.StdEncoding.EncodeToString([]byte("mkdir -p " + workingDir))
	result := system.RunSshCommand(sshClient, "bb-cli execute --elevate --base64 "+encodedCmd, env)
	if result.Error != nil {
		return result.Error
	}
	log.Tracef("Setting ownership for remote %s...", workingDir)
	encodedCmd = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("chown -R %s:%s ", user, user) + workingDir))
	result = system.RunSshCommand(sshClient, "bb-cli execute --elevate --base64 "+encodedCmd, env)
	return result.Error
}

func WriteAppDefinition(workingDir string, configuration map[string]interface{}, appConfigPath string) error {
	if isRemote, locator := IsRemoteApp(workingDir); isRemote {
		session, err := locator.OpenAppRemoteSessionS()
		if err != nil {
			return err
		}
		credentials, err := locator.GetElevationCredentials()
		if err != nil {
			return err
		}
		err = prepareFolderStructure(session.sshClient, path.Join(locator.InstancePath, locator.App), locator.Username, credentials.ToEnvMap())
		if err != nil {
			return err
		}
		return writeAppConfigurationToRemote(session.sftpSession, path.Join(locator.InstancePath, locator.App), configuration)
	}
	var appDef []byte
	var appDefPath string
	err := os.MkdirAll(workingDir, os.ModePerm)
	if err == nil {
		appDef, err = json.MarshalIndent(configuration, "", "\t")
	}
	if err != nil {
		return err
	}
	_, appDefPath, err = FindAppDefinition(workingDir)
	if err != nil || appDefPath == "" {
		appDefPath = path.Join(workingDir, appConfigPath)
	}
	return os.WriteFile(appDefPath, appDef, 0644)
}

func ReadAppDefinition(workingDir string, appConfigPath string) (*map[string]interface{}, error) {
	if isRemote, _ := IsRemoteApp(workingDir); isRemote {
		// session, err := locator.OpenAppRemoteSessionS()
		// if err != nil {
		// 	return nil, err
		// }
		return nil, errors.New("Not supported!")
	}
	var appDefPath string

	_, appDefPath, err := FindAppDefinition(workingDir)
	if err != nil || appDefPath == "" {
		appDefPath = path.Join(workingDir, appConfigPath)
	}
	appDef, err := os.ReadFile(appDefPath)
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	err = json.Unmarshal(appDef, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
