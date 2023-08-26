package system

import (
	"os"
	"os/exec"
	"os/user"
	"strings"

	"alis.is/bb-cli/cli"
	"alis.is/bb-cli/util"

	log "github.com/sirupsen/logrus"
)

func GetCurrentUser() *user.User {
	user, err := user.Current()
	util.AssertEE(err, "Failed to get current user!", cli.ExitInvalidUser)
	return user
}

func IsElevated() bool {
	user := GetCurrentUser()
	return user.Uid == "0"
}

func RequireElevatedUser(injectArgs ...string) {
	if IsElevated() {
		log.Trace("Process already elevated...")
		return
	} else {
		log.Trace("Elevation required! Trying to elevate...")
	}

	// try elevate
	sudoPass := os.Getenv("SUDO_PASS")
	suUser := os.Getenv("SU_USER")
	suPass := os.Getenv("SU_PASS")
	if strings.Trim(sudoPass, " ") != "" {
		// sudo
		sudoArgs := make([]string, 0)
		sudoArgs = append(sudoArgs, "-S", "-E", "--")
		sudoArgs = append(sudoArgs, os.Args...)
		sudoArgs = append(sudoArgs, injectArgs...)
		sudoProc := exec.Command("sudo", sudoArgs...)
		sudoProc.Stdout = os.Stdout
		sudoProc.Stderr = os.Stderr
		sudoStdin, err := sudoProc.StdinPipe()
		util.AssertEE(err, "Failed to access sudo stdin!", cli.ExitElevationRequired)
		sudoStdin.Write([]byte(os.Getenv("SUDO_PASS") + "\n"))
		err = sudoProc.Run()
		util.AssertEE(err, "Failed to execute sudo!", cli.ExitExternalError)

		os.Exit(sudoProc.ProcessState.ExitCode())
	}

	if strings.Trim(suUser, " ") != "" && strings.Trim(suPass, " ") != "" {
		// su
		os.Exit(cli.ExitNotSupported)
	}

	// other options?
	util.AssertBE(IsTty(), "No self elevation method available!", cli.ExitElevationRequired)
	_, err := exec.LookPath("sudo")
	util.AssertEE(err, "Sudo not found! Please run process as root manually.", cli.ExitElevationRequired)

	sudoArgs := make([]string, 0)
	sudoArgs = append(sudoArgs, "-S", "-E", "--")
	sudoArgs = append(sudoArgs, os.Args...)
	sudoArgs = append(sudoArgs, injectArgs...)
	sudoProc := exec.Command("sudo", sudoArgs...)
	sudoProc.Stdout = os.Stdout
	sudoProc.Stderr = os.Stderr
	sudoProc.Stdin = os.Stdin
	err = sudoProc.Run()
	util.AssertBE(err == nil || sudoProc.ProcessState.ExitCode() != 1, "Failed to elevate!", cli.ExitElevationRequired)

	os.Exit(sudoProc.ProcessState.ExitCode())
}
