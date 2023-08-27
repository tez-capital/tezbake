package system

import (
	"os"
	"os/exec"
	"os/user"
	"time"

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
	elevationKind := os.Getenv("ELEVATION_KIND")
	// elevationUser := os.Getenv("ELEVATION_USER")
	elevationPass := os.Getenv("ELEVATION_PASSWORD")
	switch elevationKind {
	case "sudo":
		// test with exit 0
		testArgs := make([]string, 0)
		testArgs = append(testArgs, "-S", "-E", "--")
		testArgs = append(testArgs, "sh", "-c", "exit 0")
		testProc := exec.Command("sudo", testArgs...)
		testSuccess := false
		done := make(chan error, 1)
		go func() {
			testProc.Stdout = os.Stdout
			testProc.Stderr = os.Stderr
			testStdin, err := testProc.StdinPipe()
			util.AssertEE(err, "Failed to access sudo stdin!", cli.ExitElevationRequired)
			testStdin.Write([]byte(elevationPass + "\n"))
			done <- testProc.Run()
		}()

		select {
		case <-time.After(3 * time.Second):
			log.Warn("Timeout occurred while testing sudo access")
		case err := <-done:
			util.AssertEE(err, "Failed to execute test sudo!", cli.ExitExternalError)
			testSuccess = true
		}

		util.AssertBE(testSuccess, "Sudo access test failed!", cli.ExitElevationRequired)

		sudoArgs := make([]string, 0)
		sudoArgs = append(sudoArgs, "-S", "-E", "--")
		sudoArgs = append(sudoArgs, os.Args...)
		sudoArgs = append(sudoArgs, injectArgs...)
		sudoProc := exec.Command("sudo", sudoArgs...)
		sudoProc.Stdout = os.Stdout
		sudoProc.Stderr = os.Stderr
		sudoStdin, err := sudoProc.StdinPipe()
		util.AssertEE(err, "Failed to access sudo stdin!", cli.ExitElevationRequired)
		sudoStdin.Write([]byte(elevationPass + "\n"))
		err = sudoProc.Run()
		util.AssertEE(err, "Failed to execute sudo!", cli.ExitExternalError)

		os.Exit(sudoProc.ProcessState.ExitCode())
	case "su":
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
