package logging

import (
	"context"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/shared/telemetry/go/otellog"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/shared/telemetry/go/otelzerolog"
	"github.com/rs/zerolog"
)

type Runtime struct {
	Logger   zerolog.Logger
	shutdown func(context.Context) error
}

// NewWithTelemetry creates a logger that writes to stdout and OTLP (Loki via collector).
func NewWithTelemetry(ctx context.Context, service string) Runtime {
	level := zerolog.InfoLevel
	if configuredLevel := strings.TrimSpace(os.Getenv("LOG_LEVEL")); configuredLevel != "" {
		if parsedLevel, err := zerolog.ParseLevel(configuredLevel); err == nil {
			level = parsedLevel
		}
	}

	zerolog.SetGlobalLevel(level)
	zerolog.TimeFieldFormat = time.RFC3339Nano

	otelShutdown, otelWriter, err := otellog.NewOTLPWriter(ctx, service)
	writers := []io.Writer{os.Stdout}
	if err != nil {
		// stdout-only fallback; tracing may still work independently
		logger := zerolog.New(os.Stdout).
			With().
			Timestamp().
			Str("service", service).
			Logger()
		logger.Warn().Err(err).Msg("OTLP log export disabled; using stdout only")
		return Runtime{Logger: logger, shutdown: otelShutdown}
	}
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" {
		writers = append(writers, otelWriter)
	}

	logger := zerolog.New(zerolog.MultiLevelWriter(writers...)).
		Hook(otelzerolog.TracingHook{}).
		With().
		Timestamp().
		Str("service", service).
		Logger()

	if endpoint := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")); endpoint != "" {
		logger.Info().
			Str("otlp_endpoint", endpoint).
			Str("otlp_protocol", "http").
			Msg("OTLP log export enabled")
	}

	return Runtime{
		Logger:   logger,
		shutdown: otelShutdown,
	}
}

// New returns a stdout-only logger (legacy).
func New(service string) zerolog.Logger {
	return NewWithTelemetry(context.Background(), service).Logger
}

func (r Runtime) Shutdown(ctx context.Context) error {
	if r.shutdown == nil {
		return nil
	}
	return r.shutdown(ctx)
}
