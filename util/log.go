package util

import (
	"os"

	log "github.com/sirupsen/logrus"
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
		log.WithFields(log.Fields{"error": err}).Error(msg)
	}
}

func AssertE(err error, msg string) {
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error(msg)
		os.Exit(-1)
	}
}

func AssertEE(err error, msg string, exitCode int) {
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error(msg)
		os.Exit(exitCode)
	}
}
