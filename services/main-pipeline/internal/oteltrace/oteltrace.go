// Package oteltrace initialises an OpenTelemetry TracerProvider that exports
// spans to the OTEL Collector via OTLP/HTTP.
//
// Usage:
//
//	shutdown, err := oteltrace.Init(ctx, "api-gateway")
//	if err != nil { ... }
//	defer shutdown(context.Background())
//
// When OTEL_EXPORTER_OTLP_ENDPOINT is empty the function registers a no-op
// provider so the service works without the monitoring stack.
package oteltrace

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// ShutdownFunc gracefully flushes pending spans and shuts down the exporter.
type ShutdownFunc func(ctx context.Context) error

// Init creates an OTLP HTTP trace exporter, wraps it in a TracerProvider and
// registers it as the global provider.  It returns a shutdown function that
// must be called on application exit.
//
// If OTEL_EXPORTER_OTLP_ENDPOINT is not set the function is a no-op: it
// installs the default (no-op) global provider and returns a nil-safe
// shutdown function.
func Init(ctx context.Context, serviceName string) (ShutdownFunc, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		return func(context.Context) error { return nil }, nil
	}

	// Use OTLP/HTTP exporter.  The exporter manages its own HTTP client
	// lifecycle — no manual connection setup required.
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("oteltrace: create exporter for %s: %w", endpoint, err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("oteltrace: create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Register as the global TracerProvider so otelhttp (and any other
	// instrumentation library) picks it up automatically.
	otel.SetTracerProvider(tp)

	// Propagate W3C Trace-Context and Baggage headers so traces are
	// correlated across service boundaries.
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	shutdown := func(ctx context.Context) error {
		return tp.Shutdown(ctx)
	}

	return shutdown, nil
}
