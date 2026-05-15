package worker

import (
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

func NewLogger(service string) zerolog.Logger {
	level := zerolog.InfoLevel
	if configuredLevel := strings.TrimSpace(os.Getenv("LOG_LEVEL")); configuredLevel != "" {
		if parsedLevel, err := zerolog.ParseLevel(configuredLevel); err == nil {
			level = parsedLevel
		}
	}

	zerolog.SetGlobalLevel(level)
	zerolog.TimeFieldFormat = time.RFC3339Nano

	return zerolog.New(os.Stdout).
		With().
		Timestamp().
		Str("service", service).
		Logger()
}
