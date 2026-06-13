package logging

import (
	"log/slog"
	"os"
	"strings"
)

func Setup(service string) {
	logFormat := strings.TrimSpace(os.Getenv("LOG_FORMAT"))
	logLevel := strings.TrimSpace(os.Getenv("LOG_LEVEL"))

	level := slog.LevelInfo
	switch strings.ToUpper(logLevel) {
	case "DEBUG":
		level = slog.LevelDebug
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: level,
	}

	if logFormat == "console" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	logger := slog.New(handler).With("service", service)
	slog.SetDefault(logger)
}
