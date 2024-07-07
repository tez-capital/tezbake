package ami

import (
	"errors"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

func ExecuteRaw(args ...string) (int, error) {
	eliPath, err := exec.LookPath("eli")
	if err != nil {
		return -1, errors.New("eli not found")
	}
	amiPath, err := exec.LookPath("ami")
	if err != nil {
		return -1, errors.New("ami not found")
	}

	eliArgs := make([]string, 0)
	eliArgs = append(eliArgs, amiPath)
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

func Execute(workingDir string, args ...string) (int, error) {
	if isRemote, locator := IsRemoteApp(workingDir); isRemote {
		session, err := locator.OpenAppRemoteSession()
		if err != nil {
			return -1, err
		}

		defer session.Close()
		return session.ProxyToRemoteApp()
	}

	eliPath, err := exec.LookPath("eli")
	if err != nil {
		return -1, errors.New("eli not found")
	}
	amiPath, err := exec.LookPath("ami")
	if err != nil {
		return -1, errors.New("ami not found")
	}

	eliArgs := make([]string, 0)
	eliArgs = append(eliArgs, amiPath)
	eliArgs = append(eliArgs, options.ToAmiArgs()...)
	eliArgs = append(eliArgs, "--path="+workingDir)
	eliArgs = append(eliArgs, args...)
	log.Trace("Executing: " + eliPath + " " + strings.Join(eliArgs, " "))
	eliProc := exec.Command(eliPath, eliArgs...)
	eliProc.Stdout = os.Stdout
	eliProc.Stderr = os.Stderr
	eliProc.Stdin = os.Stdin
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
		session, err := locator.OpenAppRemoteSession()
		if err != nil {
			return "", -1, err
		}

		defer session.Close()
		return session.ProxyToRemoteAppGetOutput()
	}

	eliPath, err := exec.LookPath("eli")
	if err != nil {
		return "", -1, errors.New("eli not found")
	}
	amiPath, err := exec.LookPath("ami")
	if err != nil {
		return "", -1, errors.New("ami not found")
	}

	eliArgs := make([]string, 0)
	eliArgs = append(eliArgs, amiPath)
	eliArgs = append(eliArgs, options.ToAmiArgs()...)
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
	amiArgs := []string{
		"--path=" + workingDir,
		"--output-format=json",
		"--log-level=" + options.LogLevel,
		"info",
	}

	if isRemote, locator := IsRemoteApp(workingDir); isRemote {
		session, err := locator.OpenAppRemoteSession()
		if err != nil {
			return []byte{}, -1, err
		}

		defer session.Close()
		return session.ProxyToRemoteAppExecuteInfo(append(amiArgs, args...))
	}

	eliPath, err := exec.LookPath("eli")
	if err != nil {
		return []byte{}, -1, errors.New("eli not found")
	}
	amiPath, err := exec.LookPath("ami")
	if err != nil {
		return []byte{}, -1, errors.New("ami not found")
	}

	eliArgs := make([]string, 0)
	eliArgs = append(eliArgs, amiPath)
	eliArgs = append(eliArgs, amiArgs...)
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
