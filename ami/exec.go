package ami

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/tez-capital/tezbake/system"
	"go.alis.is/common/log"
)

func ExecuteRaw(args ...string) (int, error) {
	eliPath, amiPath, err := GetEliAndAmiPath()
	if err != nil {
		return -1, err
	}

	eliArgs := make([]string, 0)
	eliArgs = append(eliArgs, amiPath)
	eliArgs = append(eliArgs, args...)
	log.Trace("Executing:", "eli_path", eliPath, "eli_args", strings.Join(eliArgs, " "))
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

func createAmiCmd(workingDir string, args ...string) (*exec.Cmd, error) {
	eliPath, amiPath, err := GetEliAndAmiPath()
	if err != nil {
		return nil, err
	}

	eliArgs := make([]string, 0)
	eliArgs = append(eliArgs, amiPath)
	eliArgs = append(eliArgs, options.ToAmiArgs()...)
	eliArgs = append(eliArgs, "--path="+workingDir)
	eliArgs = append(eliArgs, args...)

	// AMI changes directory for --path; changing it here would resolve relative paths twice.
	return exec.Command(eliPath, eliArgs...), nil
}

func runAmiCmd(workingDir string, args ...string) (exitCode int, err error) {
	proc, err := createAmiCmd(workingDir, args...)
	if err != nil {
		return -1, err
	}

	proc.Stdin = os.Stdin
	proc.Stdout = os.Stdout
	proc.Stderr = os.Stderr

	err = proc.Start()
	if err != nil {
		return -1, err
	}
	err = proc.Wait()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.ExitCode(), err
		}
		return -1, err
	}
	return 0, nil
}

func runAmiCmdWithOutputChannel(workingDir string, outputChannel chan<- string, args ...string) (exitCode int, err error) {
	proc, err := createAmiCmd(workingDir, args...)
	if err != nil {
		return -1, err
	}

	proc.Stdin = os.Stdin
	stdout, err := proc.StdoutPipe()
	if err != nil {
		return -1, err
	}
	stderr, err := proc.StderrPipe()
	if err != nil {
		return -1, err
	}

	if err := proc.Start(); err != nil {
		return -1, err
	}
	outputErrors := make(chan error, 2)
	go func() { outputErrors <- system.ForwardOutputLines(stdout, outputChannel) }()
	go func() { outputErrors <- system.ForwardOutputLines(stderr, outputChannel) }()
	outputErr := errors.Join(<-outputErrors, <-outputErrors)
	err = proc.Wait()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.ExitCode(), errors.Join(err, outputErr)
		}
		return -1, errors.Join(err, outputErr)
	}
	if outputErr != nil {
		return -1, outputErr
	}
	return 0, nil
}

func Execute(workingDir string, args ...string) (exitCode int, err error) {
	locator, err := loadAppLocator(workingDir)
	if err != nil && !errors.Is(err, ErrNotRemoteApp) {
		return -1, err
	}
	if err == nil {
		session, err := locator.OpenAppRemoteSession()
		if err != nil {
			return -1, err
		}

		defer session.Close()
		return session.ForwardAmiExecute(locator.GetRemoteAppPath(), args...)
	}
	return runAmiCmd(workingDir, args...)
}

func ExecuteWithOutputChannel(workingDir string, outputChannel chan<- string, args ...string) (exitCode int, err error) {
	locator, err := loadAppLocator(workingDir)
	if err != nil && !errors.Is(err, ErrNotRemoteApp) {
		return -1, err
	}
	if err == nil {
		session, err := locator.OpenAppRemoteSession()
		if err != nil {
			return -1, err
		}

		defer session.Close()
		return session.ForwardAmiExecuteWithOutputChannel(locator.GetRemoteAppPath(), outputChannel, args...)
	}
	return runAmiCmdWithOutputChannel(workingDir, outputChannel, args...)
}

func ExecuteGetOutput(workingDir string, args ...string) (output string, exitCode int, err error) {
	locator, err := loadAppLocator(workingDir)
	if err != nil && !errors.Is(err, ErrNotRemoteApp) {
		return "", -1, err
	}
	if err == nil {
		session, err := locator.OpenAppRemoteSession()
		if err != nil {
			return "", -1, err
		}

		defer session.Close()
		return session.ForwardAmiExecuteGetOutput(locator.GetRemoteAppPath(), args...)
	}

	outputChannel := make(chan string)
	var wg sync.WaitGroup
	output = ""

	wg.Add(1)
	go func() {
		defer wg.Done()
		for line := range outputChannel {
			output += line + "\n"
		}
	}()
	exitCode, err = runAmiCmdWithOutputChannel(workingDir, outputChannel, args...)
	close(outputChannel) // Close the channel to signal the goroutine to finish
	// Wait for the goroutine to finish
	wg.Wait()
	return
}

func ExecuteInfo(workingDir string, args ...string) ([]byte, int, error) {
	args = append([]string{"--output-format=json", "--log-level=" + options.LogLevel, "info"}, args...)
	output, exitCode, err := ExecuteGetOutput(workingDir, args...)
	return []byte(output), exitCode, err
}
