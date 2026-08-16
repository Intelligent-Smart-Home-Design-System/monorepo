// Package otelmetric initialises an OpenTelemetry MeterProvider that exports
// metrics to the OTEL Collector via OTLP/HTTP.
package otelmetric

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const exportInterval = time.Second

type ShutdownFunc func(ctx context.Context) error

func Init(ctx context.Context, serviceName string) (ShutdownFunc, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		return func(context.Context) error { return nil }, nil
	}

	exporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(endpoint),
		otlpmetrichttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("otelmetric: create exporter for %s: %w", endpoint, err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("otelmetric: create resource: %w", err)
	}

	provider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(exporter, metric.WithInterval(exportInterval))),
	)

	otel.SetMeterProvider(provider)

	shutdown := func(ctx context.Context) error {
		return provider.Shutdown(ctx)
	}

	return shutdown, nil
}
