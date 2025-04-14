package ami

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/system"

	"golang.org/x/crypto/ssh"

	"github.com/hjson/hjson-go/v4"
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
		session, err := locator.OpenAppRemoteSession()
		if err != nil {
			return nil, "", err
		}
		defer session.Close()

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
	log.Trace("Loading '" + app + "' definition from...")
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
	if config, ok := appDef["configuration"].(map[string]interface{}); ok {
		return config, nil
	}
	return nil, fmt.Errorf("failed to load '%s' configuration - unexpected format", app)
}

func prepareFolderStructure(sshClient *ssh.Client, instancePath string, app string, user string, env *map[string]string) error {
	workingDir := path.Join(instancePath, app)
	log.Tracef("Preparing folder structure for remote %s...", workingDir)
	encodedCmd := base64.StdEncoding.EncodeToString([]byte("mkdir -p " + workingDir))
	result := system.RunSshCommand(sshClient, "tezbake execute --elevate --base64 "+encodedCmd, env)
	if result.Error != nil {
		return result.Error
	}
	encodedCmd = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("chown -R %s:%s ", user, user) + instancePath))
	result = system.RunSshCommand(sshClient, "tezbake execute --elevate --base64 "+encodedCmd, env)
	return result.Error
}

func writeAppConfigurationToRemote(session *TezbakeRemoteSession, workingDir string, configuration map[string]interface{}) error {
	var appDef []byte
	var appDefPath string
	log.Tracef("Writing app configuration to remote %s...", workingDir)

	appDef, err := json.MarshalIndent(configuration, "", "\t")
	if err != nil {
		return err
	}
	_, appDefPath, err = findAppDefinitionRemote(session.sftpSession, workingDir)
	if err != nil || appDefPath == "" {
		appDefPath = path.Join(workingDir, constants.DefaultAppJsonName)
	}
	err = session.writeFileToRemote(appDefPath, appDef, 0644)
	if err != nil {
		return err
	}

	log.Tracef("App configuration written to %s", appDefPath)
	return nil
}

func WriteFile(workingDir string, content []byte, relativePath string) error {
	if isRemote, locator := IsRemoteApp(workingDir); isRemote {
		session, err := locator.OpenAppRemoteSession()
		if err != nil {
			return err
		}
		defer session.Close()

		credentials, err := locator.GetElevationCredentials()
		if err != nil {
			return err
		}
		err = prepareFolderStructure(session.sshClient, locator.InstancePath, locator.App, locator.Username, credentials.ToEnvMap())
		if err != nil {
			return err
		}
		targetPath := path.Join(workingDir, relativePath)
		return session.writeFileToRemote(targetPath, content, 0644)
	}

	targetPath := path.Join(workingDir, relativePath)
	err := os.MkdirAll(path.Dir(targetPath), os.ModePerm)
	if err != nil {
		return err
	}
	return os.WriteFile(targetPath, content, 0644)
}

func WriteAppDefinition(workingDir string, configuration map[string]interface{}, appConfigPath string) error {
	if isRemote, locator := IsRemoteApp(workingDir); isRemote {
		session, err := locator.OpenAppRemoteSession()
		if err != nil {
			return err
		}
		defer session.Close()

		credentials, err := locator.GetElevationCredentials()
		if err != nil {
			return err
		}
		err = prepareFolderStructure(session.sshClient, locator.InstancePath, locator.App, locator.Username, credentials.ToEnvMap())
		if err != nil {
			return err
		}
		return writeAppConfigurationToRemote(session, path.Join(locator.InstancePath, locator.App), configuration)
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
		return nil, errors.New("not supported")
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
	err = hjson.Unmarshal(appDef, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
