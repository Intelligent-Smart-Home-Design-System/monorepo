package worker

import (
	"context"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type TracingRuntime struct {
	shutdown func(context.Context) error
}

func InitTracing(ctx context.Context, settings Settings, logger zerolog.Logger) (*TracingRuntime, error) {
	if !settings.TracingEnabled || settings.OTLPEndpoint == "" {
		logger.Info().Msg("Tracing disabled")
		return &TracingRuntime{shutdown: func(context.Context) error { return nil }}, nil
	}

	options := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(settings.OTLPEndpoint)}
	if settings.OTLPInsecure {
		options = append(options, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx, options...)
	if err != nil {
		return nil, err
	}

	resource := sdkresource.NewWithAttributes(
		"",
		attribute.String("service.name", settings.ServiceName),
		attribute.String("deployment.environment", getEnv("APP_ENV", "development")),
		attribute.String("service.version", getEnv("SERVICE_VERSION", "dev")),
	)

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	logger.Info().Str("otlp_endpoint", settings.OTLPEndpoint).Msg("Tracing enabled")
	return &TracingRuntime{shutdown: provider.Shutdown}, nil
}

func (r *TracingRuntime) Shutdown(ctx context.Context) error {
	if r == nil || r.shutdown == nil {
		return nil
	}
	return r.shutdown(ctx)
}

func traceLogger(ctx context.Context, logger zerolog.Logger) zerolog.Logger {
	spanContext := traceSpanContext(ctx)
	if spanContext.traceID == "" {
		return logger
	}

	return logger.With().
		Str("trace_id", spanContext.traceID).
		Str("span_id", spanContext.spanID).
		Logger()
}

type spanIDs struct {
	traceID string
	spanID  string
}

func traceSpanContext(ctx context.Context) spanIDs {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if !sc.IsValid() {
		return spanIDs{}
	}
	return spanIDs{
		traceID: sc.TraceID().String(),
		spanID:  sc.SpanID().String(),
	}
}
