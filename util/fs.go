package util

import (
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"

	"go.alis.is/common/log"

	"github.com/tez-capital/tezbake/constants"
)

func ChownRS(username string, targetPath string) (int, error) {
	userInfo, err := user.Lookup(username)
	if err != nil {
		return constants.ExitUserNotFound, err
	}

	if runtime.GOOS != "windows" {
		uid, err := strconv.Atoi(userInfo.Uid)
		if err != nil {
			return constants.ExitInvalidUser, err
		}
		gid, err := strconv.Atoi(userInfo.Gid)
		if err != nil {
			return constants.ExitInvalidUser, err
		}

		err = filepath.Walk(targetPath, func(path string, info fs.FileInfo, err error) error {
			if err == nil {
				err = os.Chown(path, uid, gid)
				if err != nil {
					log.Warn("Failed to change ownership:", "path", path, "error", err.Error())
				}
			}
			return err
		})
		if err != nil {
			return constants.ExitIOError, err
		}
	}
	return 0, nil
}

func ChownR(username string, targetPath string) {
	exitCode, err := ChownRS(username, targetPath)
	AssertEE(err, "Failed to convert user id", exitCode)
}
