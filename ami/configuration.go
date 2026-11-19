package ami

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/system"
	"go.alis.is/common/log"

	"github.com/hjson/hjson-go/v4"
	"github.com/kballard/go-shellquote"
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
			closeErr := appDefFile.Close()
			if err != nil {
				return nil, "", err
			}
			if closeErr != nil {
				return nil, "", closeErr
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
	locator, err := loadAppLocator(workingDir)
	if err != nil && !errors.Is(err, ErrNotRemoteApp) {
		return nil, "", err
	}
	if err == nil {
		session, err := locator.OpenAppRemoteSession()
		if err != nil {
			return nil, "", err
		}
		defer session.Close()

		return findAppDefinitionRemote(session.sftpSession, locator.GetRemoteAppPath())
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
	encodedCmd := base64.StdEncoding.EncodeToString([]byte(shellquote.Join("mkdir", "-p", "--", workingDir)))
	result := system.RunSshCommand(sshClient, "tezbake execute --elevate --base64 "+encodedCmd, env)
	if result.Error != nil {
		return result.Error
	}
	encodedCmd = base64.StdEncoding.EncodeToString([]byte(shellquote.Join("chown", "--", user+":"+user, workingDir)))
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
	newAppDefPath := appDefPath + ".new-" + rand.Text()
	file, err := session.sftpSession.OpenFile(newAppDefPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL)
	if err != nil {
		return err
	}
	defer session.sftpSession.Remove(newAppDefPath)
	defer file.Close()
	if _, err := file.Write(appDef); err != nil {
		return err
	}
	if err := file.Chmod(0644); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	if err = session.sftpSession.PosixRename(newAppDefPath, appDefPath); err != nil {
		return err
	}

	log.Trace("App configuration written to", "app_def_path", appDefPath)
	return nil
}

func WriteFile(workingDir string, content []byte, relativePath string) error {
	locator, err := loadAppLocator(workingDir)
	if err != nil && !errors.Is(err, ErrNotRemoteApp) {
		return err
	}
	if err == nil {
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
		targetPath := path.Join(locator.GetRemoteAppPath(), relativePath)
		if err := session.sftpSession.MkdirAll(path.Dir(targetPath)); err != nil {
			return err
		}
		return session.writeFileToRemote(targetPath, content, 0644)
	}

	targetPath := path.Join(workingDir, relativePath)
	err = os.MkdirAll(path.Dir(targetPath), os.ModePerm)
	if err != nil {
		return err
	}
	return os.WriteFile(targetPath, content, 0644)
}

func WriteAppDefinition(workingDir string, configuration map[string]any, appConfigPath string) error {
	locator, err := loadAppLocator(workingDir)
	if err != nil && !errors.Is(err, ErrNotRemoteApp) {
		return err
	}
	if err == nil {
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
		return writeAppConfigurationToRemote(session, locator.GetRemoteAppPath(), configuration)
	}
	var appDef []byte
	var appDefPath string
	err = os.MkdirAll(workingDir, os.ModePerm)
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

	return WriteFileAtomic(appDefPath, appDef, 0644)
}

func EnsureRemoteAppDirectory(locator *RemoteConfiguration) error {
	session, err := locator.OpenAppRemoteSession()
	if err != nil {
		return err
	}
	defer session.Close()

	credentials, err := locator.GetElevationCredentials()
	if err != nil {
		return err
	}

	return prepareFolderStructure(session.sshClient, locator.InstancePath, locator.App, locator.Username, credentials.ToEnvMap())
}

func ReadAppDefinition(workingDir string, appConfigPath string) (map[string]any, error) {
	locator, err := loadAppLocator(workingDir)
	if err != nil && !errors.Is(err, ErrNotRemoteApp) {
		return nil, err
	}
	if err == nil {
		session, err := locator.OpenAppRemoteSession()
		if err != nil {
			return nil, err
		}
		defer session.Close()

		appDef, _, err := findAppDefinitionRemote(session.sftpSession, locator.GetRemoteAppPath())
		if err != nil {
			return nil, err
		}
		return appDef, nil
	}
	var appDefPath string

	_, appDefPath, err = FindAppDefinition(workingDir)
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
