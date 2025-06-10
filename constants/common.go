package constants

import (
	"runtime"
)

const (
	TezbakeRepository string = "tez-capital/tezbake"

	defaultBBDirectory      string = "/bake-buddy"
	defaultBBDirectoryMacOS string = "/usr/local/bake-buddy"
	DefaultRemoteUser       string = "bb"
	DefaultSshUser          string = "root"

	DefaultAppJsonName string = "app.json"

	TzktConsensusKeyCheckingEndpoint = "https://api.tzkt.io/"
)

var (
	DefaultBBDirectory string = defaultBBDirectory
)

func init() {
	if runtime.GOOS == "darwin" {
		DefaultBBDirectory = defaultBBDirectoryMacOS
	}
}
