package ami

import (
	"bufio"
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

func executeInternal(workingDir string, outputChannel chan<- string, args ...string) (exitCode int, err error) {
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

	switch outputChannel {
	case nil:
		eliProc.Stdout = os.Stdout
		eliProc.Stderr = os.Stderr
	default:
		stdout, err := eliProc.StdoutPipe()
		if err != nil {
			return -1, err
		}
		stderr, err := eliProc.StderrPipe()
		if err != nil {
			return -1, err
		}
		go func() {
			// feed the output channel with the output of the command
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				// Send each line of output to the channel
				outputChannel <- scanner.Text()
			}
		}()
		go func() {
			// feed the output channel with the output of the command
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				// Send each line of output to the channel
				outputChannel <- scanner.Text()
			}
		}()
	}

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

func Execute(workingDir string, args ...string) (exitCode int, err error) {
	return executeInternal(workingDir, nil, args...)
}

func ExecuteWithOutputChannel(workingDir string, outputChannel chan<- string, args ...string) (exitCode int, err error) {
	return executeInternal(workingDir, outputChannel, args...)
}

func ExecuteGetOutput(workingDir string, args ...string) (output string, exitCode int, err error) {
	outputChannel := make(chan string)
	exitCode, err = executeInternal(workingDir, outputChannel, args...)
	output = ""
	for line := range outputChannel {
		output += line + "\n"
	}
	return
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
