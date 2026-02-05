package ami

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/alis-is/go-common/log"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/system"
	"github.com/tez-capital/tezbake/util"

	"github.com/hjson/hjson-go/v4"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func findAppDefinitionRemote(sftpClient *sftp.Client, workingDir string) (map[string]any, string, error) {
	for _, candidate := range AppConfigurationCandidates {
		appDefPath := path.Join(workingDir, candidate)
		appDefFile, err := sftpClient.OpenFile(appDefPath, os.O_RDONLY)
		if err == nil {
			log.Trace("App definition found in", "app_def_path", appDefPath)
			appDef := make(map[string]any)
			appDefContent, err := io.ReadAll(appDefFile)
			if err != nil {
				return nil, "", err
			}

			err = hjson.Unmarshal(appDefContent, &appDef)
			if err != nil {
				return nil, "", err
			}
			return appDef, appDefPath, err
		}
	}
	return nil, "", errors.New("failed to load app configuration (no valid configuration found)")
}

func FindAppDefinition(workingDir string) (map[string]any, string, error) {
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
			log.Trace("App definition found in", "app_def_path", appDefPath)
			appDef := make(map[string]any)
			err = hjson.Unmarshal(appDefContent, &appDef)
			return appDef, appDefPath, err
		}
	}
	return nil, "", errors.New("failed to load app configuration (no valid configuration found)")
}

func LoadAppDefinition(app string) (map[string]any, error) {
	log.Trace("Loading app definition from...", "app", app)
	appDef, _, err := FindAppDefinition(app)
	if err != nil {
		return nil, err
	}
	return appDef, nil
}

func LoadAppConfiguration(app string) (map[string]any, error) {
	appDef, err := LoadAppDefinition(app)
	if err != nil {
		return nil, err
	}
	if config, ok := appDef["configuration"].(map[string]any); ok {
		return config, nil
	}
	return nil, fmt.Errorf("failed to load '%s' configuration - unexpected format", app)
}

func UpdateAppConfiguration(app string, configuration map[string]any) error {
	appDef, err := LoadAppDefinition(app)
	if err != nil {
		return err
	}
	if _, ok := appDef["configuration"].(map[string]any); !ok {
		return fmt.Errorf("failed to load '%s' configuration - unexpected format", app)
	}
	appDef["configuration"] = configuration
	return WriteAppDefinition(app, appDef, constants.DefaultAppJsonName)
}

func GetAppActiveModel(workingDir string) (map[string]any, error) {
	output, exitCode, err := ExecuteGetOutput(workingDir, "--print-model")
	if err != nil {
		return nil, err
	}
	if exitCode != 0 {
		return nil, fmt.Errorf("failed to get active model - %s", output)
	}
	var model map[string]any
	err = hjson.Unmarshal([]byte(output), &model)
	if err != nil {
		return nil, err
	}
	if model == nil {
		return nil, fmt.Errorf("failed to get active model - unexpected format")
	}
	return model, nil
}

func prepareFolderStructure(sshClient *ssh.Client, instancePath string, app string, user string, env *map[string]string) error {
	workingDir := path.Join(instancePath, app)
	log.Trace("Preparing folder structure for remote...", "working_dir", workingDir)
	encodedCmd := base64.StdEncoding.EncodeToString([]byte("mkdir -p " + workingDir))
	result := system.RunSshCommand(sshClient, "tezbake execute --elevate --base64 "+encodedCmd, env)
	if result.Error != nil {
		return result.Error
	}
	encodedCmd = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("chown -R %s:%s ", user, user) + instancePath))
	result = system.RunSshCommand(sshClient, "tezbake execute --elevate --base64 "+encodedCmd, env)
	return result.Error
}

func writeAppConfigurationToRemote(session *TezbakeRemoteSession, workingDir string, configuration map[string]any) error {
	var appDef []byte
	var appDefPath string
	log.Trace("Writing app configuration to remote...", "working_dir", workingDir)

	appDef, err := json.MarshalIndent(configuration, "", "\t")
	if err != nil {
		return err
	}
	_, appDefPath, err = findAppDefinitionRemote(session.sftpSession, workingDir)
	if err != nil || appDefPath == "" {
		appDefPath = path.Join(workingDir, constants.DefaultAppJsonName)
	}
	newAppDefPath := appDefPath + ".new"
	if err = session.writeFileToRemote(newAppDefPath, appDef, 0644); err != nil {
		return err
	}

	if err = session.sftpSession.PosixRename(newAppDefPath, appDefPath); err != nil {
		return err
	}

	log.Trace("App configuration written to", "app_def_path", appDefPath)
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
		util.AssertEE(err, "Failed to get elevation credentials!", constants.ExitInvalidRemoteCredentials)

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

func WriteAppDefinition(workingDir string, configuration map[string]any, appConfigPath string) error {
	if isRemote, locator := IsRemoteApp(workingDir); isRemote {
		session, err := locator.OpenAppRemoteSession()
		if err != nil {
			return err
		}
		defer session.Close()

		credentials, err := locator.GetElevationCredentials()
		util.AssertEE(err, "Failed to get elevation credentials!", constants.ExitInvalidRemoteCredentials)

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

	newAppDefPath := appDefPath + ".new"
	if err = os.WriteFile(newAppDefPath, appDef, 0644); err != nil {
		return err
	}
	return os.Rename(newAppDefPath, appDefPath)
}

func ReadAppDefinition(workingDir string, appConfigPath string) (map[string]any, error) {
	if isRemote, locator := IsRemoteApp(workingDir); isRemote {
		session, err := locator.OpenAppRemoteSession()
		if err != nil {
			return nil, err
		}
		defer session.Close()

		appDef, _, err := findAppDefinitionRemote(session.sftpSession, path.Join(locator.InstancePath, locator.App))
		if err != nil {
			return nil, err
		}
		return appDef, nil
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

	result := make(map[string]any)
	err = hjson.Unmarshal(appDef, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
