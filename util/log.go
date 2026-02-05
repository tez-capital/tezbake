package util

import (
	"os"

	"go.alis.is/common/log"
)

func AssertSB(check bool, msg string) {
	if !check {
		log.Error(msg)
	}
}

func AssertB(check bool, msg string) {
	if !check {
		log.Error(msg)
		os.Exit(-1)
	}
}

func AssertBE(check bool, msg string, exitCode int) {
	if !check {
		log.Error(msg)
		os.Exit(exitCode)
	}
}

func AssertSE(err error, msg string) {
	if err != nil {
		log.Error(msg, "error", err)
	}
}

func AssertE(err error, msg string) {
	if err != nil {
		log.Error(msg, "error", err)
		os.Exit(-1)
	}
}

func AssertEE(err error, msg string, exitCode int) {
	if err != nil {
		log.Error(msg, "error", err)
		os.Exit(exitCode)
	}
}
