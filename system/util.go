package system

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func IsTty() bool {
	if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) != 0 {
		return true
	} else {
		return false
	}
}

func CopySelfToSystem(username string) error {
	selfPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to determine path of the current executable: %v", err)

	}
	dstPath := filepath.Join("/usr/sbin", filepath.Base(selfPath))
	cmd := exec.Command("cp", "-p", selfPath, dstPath)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to copy binary using cp command: %v", err)
	}
	return nil
}
