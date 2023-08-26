package ami

import (
	"os"
	"os/exec"
	"strings"

	"alis.is/bb-cli/cli"
	"alis.is/bb-cli/util"

	log "github.com/sirupsen/logrus"
)

func Execute(workingDir string, args ...string) (int, error) {
	if isRemote, locator := IsRemoteApp(workingDir); isRemote {
		session, err := locator.OpenAppRemoteSessionS()
		if err != nil {
			return -1, err
		}
		return session.ProxyToRemoteApp()
	}

	eliPath, err := exec.LookPath("eli")
	util.AssertEE(err, "eli not found!", cli.ExitEliNotFound)
	amiPath, err := exec.LookPath("ami")
	util.AssertEE(err, "ami not found!", cli.ExitAmiNotFound)

	eliArgs := make([]string, 0)
	eliArgs = append(eliArgs, amiPath)
	if cli.JsonLogFormat {
		eliArgs = append(eliArgs, "--output-format=json")
	} else {
		eliArgs = append(eliArgs, "--output-format=standard")
	}
	eliArgs = append(eliArgs, "--log-level="+cli.LogLevel)
	eliArgs = append(eliArgs, "--path="+workingDir)
	eliArgs = append(eliArgs, args...)
	log.Trace("Executing: " + eliPath + " " + strings.Join(eliArgs, " "))
	eliProc := exec.Command(eliPath, eliArgs...)
	eliProc.Stdout = os.Stdout
	eliProc.Stderr = os.Stderr
	err = eliProc.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.ExitCode(), err
		}
		return -1, err
	}
	return 0, nil
}

func ExecuteGetOutput(workingDir string, args ...string) (string, int, error) {
	if isRemote, locator := IsRemoteApp(workingDir); isRemote {
		session, err := locator.OpenAppRemoteSessionS()
		if err != nil {
			return "", -1, err
		}
		return session.ProxyToRemoteAppGetOutput()
	}

	eliPath, err := exec.LookPath("eli")
	util.AssertEE(err, "eli not found!", cli.ExitEliNotFound)
	amiPath, err := exec.LookPath("ami")
	util.AssertEE(err, "ami not found!", cli.ExitAmiNotFound)

	eliArgs := make([]string, 0)
	eliArgs = append(eliArgs, amiPath)
	if cli.JsonLogFormat {
		eliArgs = append(eliArgs, "--output-format=json")
	} else {
		eliArgs = append(eliArgs, "--output-format=standard")
	}
	eliArgs = append(eliArgs, "--log-level="+cli.LogLevel)
	eliArgs = append(eliArgs, "--path="+workingDir)
	eliArgs = append(eliArgs, args...)
	log.Trace("Executing: " + eliPath + " " + strings.Join(eliArgs, " "))
	output, err := exec.Command(eliPath, eliArgs...).CombinedOutput()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return string(output), exitError.ExitCode(), err
		}
		return string(output), -1, err
	}
	return string(output), 0, nil
}

func ExecuteInfo(workingDir string, args ...string) ([]byte, int, error) {
	if isRemote, locator := IsRemoteApp(workingDir); isRemote {
		session, err := locator.OpenAppRemoteSessionS()
		if err != nil {
			return []byte{}, -1, err
		}
		return session.ProxyToRemoteAppExecuteInfo()
	}

	eliPath, err := exec.LookPath("eli")
	util.AssertEE(err, "eli not found!", cli.ExitEliNotFound)
	amiPath, err := exec.LookPath("ami")
	util.AssertEE(err, "ami not found!", cli.ExitAmiNotFound)

	eliArgs := make([]string, 0)
	eliArgs = append(eliArgs, amiPath)
	eliArgs = append(eliArgs, "--path="+workingDir)
	eliArgs = append(eliArgs, "--output-format=json")
	eliArgs = append(eliArgs, "--log-level="+cli.LogLevel)
	eliArgs = append(eliArgs, "info")
	eliArgs = append(eliArgs, args...)
	output, err := exec.Command(eliPath, eliArgs...).CombinedOutput()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return output, exitError.ExitCode(), err
		}
		return output, -1, err
	}
	return output, 0, nil
}
