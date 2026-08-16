// Package otelsetup wires shared OTLP log/trace exporters into a zerolog logger.
package otelsetup

import (
	"context"
	"os"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/shared/telemetry/go/otellog"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/shared/telemetry/go/otelmetric"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/shared/telemetry/go/oteltrace"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/shared/telemetry/go/otelzerolog"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

const shutdownTimeout = 5 * time.Second

type Runtime struct {
	Log      zerolog.Logger
	Meter    metric.Meter
	shutdown []func(context.Context) error
}

func New(ctx context.Context, serviceName string) Runtime {
	var shutdowns []func(context.Context) error

	otelShutdown, otelWriter, err := otellog.NewOTLPWriter(ctx, serviceName)
	if err != nil {
		fallback := zerolog.New(os.Stdout).With().Timestamp().Str("service", serviceName).Logger()
		fallback.Warn().Err(err).Msg("failed to initialize OTLP log writer, falling back to stdout")
		otelShutdown = func(context.Context) error { return nil }
		otelWriter = &otellog.Writer{}
	}
	shutdowns = append(shutdowns, otelShutdown)

	traceShutdown, err := oteltrace.Init(ctx, serviceName)
	if err != nil {
		fallback := zerolog.New(os.Stdout).With().Timestamp().Str("service", serviceName).Logger()
		fallback.Warn().Err(err).Msg("failed to initialize OTLP trace provider, tracing disabled")
		traceShutdown = func(context.Context) error { return nil }
	}
	shutdowns = append(shutdowns, traceShutdown)

	metricShutdown, err := otelmetric.Init(ctx, serviceName)
	if err != nil {
		fallback := zerolog.New(os.Stdout).With().Timestamp().Str("service", serviceName).Logger()
		fallback.Warn().Err(err).Msg("failed to initialize OTLP metric provider, metrics disabled")
		metricShutdown = func(context.Context) error { return nil }
	}
	shutdowns = append(shutdowns, metricShutdown)

	log := zerolog.New(zerolog.MultiLevelWriter(os.Stdout, otelWriter)).
		Hook(otelzerolog.TracingHook{}).
		With().Timestamp().Str("service", serviceName).Logger()

	return Runtime{
		Log:      log,
		Meter:    otel.Meter(serviceName),
		shutdown: shutdowns,
	}
}

func (r Runtime) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	for _, fn := range r.shutdown {
		_ = fn(ctx)
	}
}
