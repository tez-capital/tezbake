package ami

import (
	"fmt"
	"os"
	"strings"

	"go.alis.is/common/log"
)

func SetupApp(appDir string, args ...string) (int, error) {
	log.Trace("Installing...", "app_dir", appDir)
	exitCode, err := Execute(appDir, "--erase-cache")
	if err != nil {
		log.Error("Failed to erase cache:", "error", err)
		return exitCode, err
	}
	if exitCode != 0 {
		log.Error("Failed to erase cache:", "exitcode", exitCode)
		return exitCode, fmt.Errorf("failed to erase cache")
	}

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
	installed, _ := IsAppInstalledS(app)
	return installed
}

// IsAppInstalledS distinguishes an absent app from an execution or remote configuration failure.
func IsAppInstalledS(app string) (bool, error) {
	if _, err := os.Stat(app); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	output, exitCode, err := ExecuteGetOutput(app, "--is-app-installed")
	if exitCode == 2 { // AMI's EXIT_NOT_INSTALLED.
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if exitCode != 0 {
		return false, fmt.Errorf("installation check exited with code %d", exitCode)
	}
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) == "true" {
			return true, nil
		}
	}
	return false, fmt.Errorf("unexpected installation check output: %s", strings.TrimSpace(output))
}
