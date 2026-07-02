package utils

import (
	"log/slog"
	"os"
)

type LogLevel slog.Level

const (
	LevelDebug LogLevel = LogLevel(slog.LevelDebug)
	LevelInfo           = LogLevel(slog.LevelInfo)
	LevelWarn           = LogLevel(slog.LevelWarn)
	LevelError          = LogLevel(slog.LevelError)
)

func InitLogging() {
	level := slog.LevelInfo
	switch Settings.LogLevel {
	case "DEBUG":
		level = slog.LevelDebug
	case "WARN", "WARNING":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))
}
