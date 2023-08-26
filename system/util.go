package system

import (
	"os"
)

func IsTty() bool {
	if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) != 0 {
		return true
	} else {
		return false
	}
}
