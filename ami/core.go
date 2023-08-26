package ami

import (
	"os"
	"os/exec"
	"strings"

	"alis.is/bb-cli/cli"
	"alis.is/bb-cli/util"

	log "github.com/sirupsen/logrus"
)

var (
	AppConfigurationCandidates []string = []string{"app.hjson", "app.json"}
)

func EraseCache() (int, error) {
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
