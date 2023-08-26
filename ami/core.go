package ami

import (
	"errors"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

var (
	AppConfigurationCandidates []string = []string{"app.hjson", "app.json"}
)

func EraseCache() (int, error) {
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
	eliArgs = append(eliArgs, "--erase-cache")
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
