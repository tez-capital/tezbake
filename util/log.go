package util

import (
	"os"

	"github.com/tez-capital/tezbake/logging"
)

func AssertSB(check bool, msg string) {
	if !check {
		logging.Error(msg)
	}
}

func AssertB(check bool, msg string) {
	if !check {
		logging.Error(msg)
		os.Exit(-1)
	}
}

func AssertBE(check bool, msg string, exitCode int) {
	if !check {
		logging.Error(msg)
		os.Exit(exitCode)
	}
}

func AssertSE(err error, msg string) {
	if err != nil {
		logging.Error(msg, "error", err)
	}
}

func AssertE(err error, msg string) {
	if err != nil {
		logging.Error(msg, "error", err)
		os.Exit(-1)
	}
}

func AssertEE(err error, msg string, exitCode int) {
	if err != nil {
		logging.Error(msg, "error", err)
		os.Exit(exitCode)
	}
}
