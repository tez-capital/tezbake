package ami

import (
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

func SetupApp(appDir string, args ...string) (int, error) {
	log.Trace("Installing '" + appDir + "...")
	execArgs := make([]string, 0)
	execArgs = append(execArgs, "setup")
	execArgs = append(execArgs, args...)
	return Execute(appDir, execArgs...)
}

func StartApp(appDir string, args ...string) (int, error) {
	execArgs := make([]string, 0)
	execArgs = append(execArgs, "start")
	execArgs = append(execArgs, args...)
	return Execute(appDir, execArgs...)
}

func StopApp(appDir string, args ...string) (int, error) {
	execArgs := make([]string, 0)
	execArgs = append(execArgs, "stop")
	execArgs = append(execArgs, args...)
	return Execute(appDir, execArgs...)
}

func RemoveApp(app string, all bool, args ...string) (int, error) {
	if _, err := os.Stat(app); os.IsNotExist(err) {
		return 0, nil
	}
	StopApp(app)
	execArgs := make([]string, 0)
	execArgs = append(execArgs, "remove")
	if all {
		execArgs = append(execArgs, "--all")
	}
	execArgs = append(execArgs, args...)
	return Execute(app, execArgs...)
}

func IsAppInstalled(app string) bool {
	if isRemote, locator := IsRemoteApp(app); isRemote {
		session, err := locator.OpenAppRemoteSession()
		if err != nil {
			return false
		}
		defer session.Close()

		output, exitCode, err := session.IsRemoteAppInstalled(app)
		return err == nil && exitCode == 0 && strings.Contains(string(output), "true")
	}
	output, exitCode, err := ExecuteGetOutput(app, "--is-app-installed")
	return err == nil && exitCode == 0 && strings.Contains(output, "true")
}
