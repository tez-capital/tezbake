package ami

import (
	"errors"
	"os"
	"os/exec"
	"strings"

	"github.com/tez-capital/tezbake/logging"
)

var (
	AppConfigurationCandidates []string = []string{"app.hjson", "app.json"}
)

func GetFromPathCandidates(candidates []string) (string, error) {
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", errors.New("not found")
}

func GetEliAndAmiPath() (string, string, error) {
	logging.Debug("Looking for eli and ami in PATH", "PATH", os.Getenv("PATH"))

	// Try to find eli and ami in /usr/local/bin first
	eliPath, _ := GetFromPathCandidates([]string{"/usr/local/bin/eli"})
	amiPath, _ := GetFromPathCandidates([]string{"/usr/local/bin/ami"})
	if eliPath == "" {
		var err error
		eliPath, err = exec.LookPath("eli")
		if err != nil {
			return "", "", errors.New("eli not found")
		}
	}

	if amiPath == "" {
		var err error
		amiPath, err = exec.LookPath("ami")
		if err != nil {
			return "", "", errors.New("ami not found")
		}
	}

	return eliPath, amiPath, nil
}

func EraseCache() (int, error) {
	eliPath, amiPath, err := GetEliAndAmiPath()
	if err != nil {
		return -1, err
	}

	eliArgs := make([]string, 0)
	eliArgs = append(eliArgs, amiPath)
	eliArgs = append(eliArgs, options.ToAmiArgs()...)
	eliArgs = append(eliArgs, "--erase-cache")
	logging.Trace("Executing:", "eliPath", eliPath, "eliArgs", strings.Join(eliArgs, " "))
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
