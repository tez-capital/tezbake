package system

import (
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/tez-capital/tezbake/cli"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/util"
	"go.alis.is/common/log"
)

func getCurrentUser() *user.User {
	user, err := user.Current()
	util.AssertEE(err, "Failed to get current user!", constants.ExitInvalidUser)
	return user
}

func isElevated() bool {
	user := getCurrentUser()
	return user.Uid == "0"
}

// A completed sudo process carries the command's status, including nonzero exits.
func runElevatedProcess(proc *exec.Cmd) (int, error) {
	err := proc.Run()
	if _, exited := err.(*exec.ExitError); err != nil && !exited {
		return -1, err
	}
	return proc.ProcessState.ExitCode(), nil
}

func RequireElevatedUser(injectArgs ...string) {
	cli.ElevationRequired = true
	if isElevated() {
		log.Trace("Process already elevated...")
		return
	} else {
		log.Trace("Elevation required! Trying to elevate...")
	}

	// try elevate
	elevationKind := os.Getenv("ELEVATION_KIND")
	// elevationUser := os.Getenv("ELEVATION_USER")
	elevationPass := os.Getenv("ELEVATION_PASSWORD")
	switch elevationKind {
	case "sudo":
		sudoArgs := make([]string, 0)
		sudoArgs = append(sudoArgs, "-S", "-E", "--")
		sudoArgs = append(sudoArgs, os.Args...)
		sudoArgs = append(sudoArgs, injectArgs...)
		sudoProc := exec.Command("sudo", sudoArgs...)
		sudoProc.Stdout = os.Stdout
		sudoProc.Stderr = os.Stderr
		sudoProc.Stdin = strings.NewReader(elevationPass + "\n")
		exitCode, err := runElevatedProcess(sudoProc)
		util.AssertEE(err, "Failed to execute sudo!", constants.ExitExternalError)

		os.Exit(exitCode)
	case "su":
		os.Exit(constants.ExitNotSupported)
	}
	// other options?
	if !IsTty() {
		log.Debug("No self elevation method available!")
		os.Exit(constants.ExitElevationRequired)
	}
	_, err := exec.LookPath("sudo")
	util.AssertEE(err, "Sudo not found! Please run process as root manually.", constants.ExitElevationRequired)

	sudoArgs := make([]string, 0)
	sudoArgs = append(sudoArgs, "-S", "-E", "--")
	sudoArgs = append(sudoArgs, os.Args...)
	sudoArgs = append(sudoArgs, injectArgs...)
	sudoProc := exec.Command("sudo", sudoArgs...)
	sudoProc.Stdout = os.Stdout
	sudoProc.Stderr = os.Stderr
	sudoProc.Stdin = os.Stdin
	exitCode, err := runElevatedProcess(sudoProc)
	util.AssertEE(err, "Failed to elevate!", constants.ExitElevationRequired)

	os.Exit(exitCode)
}
