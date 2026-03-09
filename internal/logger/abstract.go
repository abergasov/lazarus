package logger

import (
	"log/slog"
)

type Field struct {
	a slog.Attr
}

type AppLogger interface {
	Info(message string, args ...Field)
	Error(message string, err error, args ...Field)
	Fatal(message string, err error, args ...Field)
	With(args ...Field) AppLogger
}
