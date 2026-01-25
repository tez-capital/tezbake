package util

import (
	"log/slog"
	"os"
)

func AssertSB(check bool, msg string) {
	if !check {
		slog.Error(msg)
	}
}

func AssertB(check bool, msg string) {
	if !check {
		slog.Error(msg)
		os.Exit(-1)
	}
}

func AssertBE(check bool, msg string, exitCode int) {
	if !check {
		slog.Error(msg)
		os.Exit(exitCode)
	}
}

func AssertSE(err error, msg string) {
	if err != nil {
		slog.Error(msg, "error", err)
	}
}

func AssertE(err error, msg string) {
	if err != nil {
		slog.Error(msg, "error", err)
		os.Exit(-1)
	}
}

func AssertEE(err error, msg string, exitCode int) {
	if err != nil {
		slog.Error(msg, "error", err)
		os.Exit(exitCode)
	}
}
