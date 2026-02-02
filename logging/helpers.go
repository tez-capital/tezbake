package logging

import (
	"context"
	"fmt"
	"log/slog"
)

func Trace(msg string, args ...any) {
	slog.Log(context.Background(), levelTrace, msg, args...)
}

func Tracef(format string, args ...any) {
	slog.Log(context.Background(), levelTrace, fmt.Sprintf(format, args...))
}

func Debug(msg string, args ...any) {
	slog.Debug(msg, args...)
}

func Debugf(format string, args ...any) {
	slog.Debug(fmt.Sprintf(format, args...))
}

func Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

func Infof(format string, args ...any) {
	slog.Info(fmt.Sprintf(format, args...))
}

func Warn(msg string, args ...any) {
	slog.Warn(msg, args...)
}

func Warnf(format string, args ...any) {
	slog.Warn(fmt.Sprintf(format, args...))
}

func Error(msg string, args ...any) {
	slog.Error(msg, args...)
}

func Errorf(format string, args ...any) {
	slog.Error(fmt.Sprintf(format, args...))
}
