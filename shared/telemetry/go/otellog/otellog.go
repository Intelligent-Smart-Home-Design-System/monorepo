// Package otellog provides a zerolog io.Writer that sends log records to an
// OpenTelemetry Collector via OTLP/HTTP.
package otellog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type ShutdownFunc func(ctx context.Context) error

func NewOTLPWriter(ctx context.Context, serviceName string) (ShutdownFunc, *Writer, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		return func(context.Context) error { return nil }, &Writer{}, nil
	}

	exporter, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpoint(endpoint),
		otlploghttp.WithInsecure(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("otellog: create exporter for %s: %w", endpoint, err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("otellog: create resource: %w", err)
	}

	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		sdklog.WithResource(res),
	)

	logger := provider.Logger("otellog")

	shutdown := func(ctx context.Context) error {
		return provider.Shutdown(ctx)
	}

	return shutdown, &Writer{logger: logger}, nil
}

type Writer struct {
	logger otellog.Logger
	mu     sync.Mutex
}

func (w *Writer) Write(p []byte) (int, error) {
	if w.logger == nil {
		return len(p), nil
	}

	var fields map[string]interface{}
	if err := json.Unmarshal(p, &fields); err != nil {
		var rec otellog.Record
		rec.SetBody(otellog.StringValue(string(p)))
		rec.SetTimestamp(time.Now())
		w.emit(rec)
		return len(p), nil
	}

	var rec otellog.Record

	if msg, ok := fields["message"].(string); ok {
		rec.SetBody(otellog.StringValue(msg))
		delete(fields, "message")
	}

	if lvl, ok := fields["level"].(string); ok {
		rec.SetSeverity(mapSeverity(lvl))
		rec.SetSeverityText(lvl)
		delete(fields, "level")
	}

	if ts, ok := fields["time"].(string); ok {
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			rec.SetTimestamp(t)
		} else {
			rec.SetTimestamp(time.Now())
		}
		delete(fields, "time")
	} else {
		rec.SetTimestamp(time.Now())
	}

	attrs := make([]otellog.KeyValue, 0, len(fields))
	for k, v := range fields {
		attrs = append(attrs, otellog.KeyValue{
			Key:   k,
			Value: toOTELValue(v),
		})
	}
	rec.AddAttributes(attrs...)

	w.emit(rec)
	return len(p), nil
}

func (w *Writer) emit(rec otellog.Record) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.logger.Emit(context.Background(), rec)
}

func mapSeverity(level string) otellog.Severity {
	switch level {
	case "trace":
		return otellog.SeverityTrace
	case "debug":
		return otellog.SeverityDebug
	case "info":
		return otellog.SeverityInfo
	case "warn":
		return otellog.SeverityWarn
	case "error":
		return otellog.SeverityError
	case "fatal":
		return otellog.SeverityFatal
	case "panic":
		return otellog.SeverityFatal2
	default:
		return otellog.SeverityInfo
	}
}

func toOTELValue(v interface{}) otellog.Value {
	switch val := v.(type) {
	case string:
		return otellog.StringValue(val)
	case float64:
		return otellog.Float64Value(val)
	case bool:
		return otellog.BoolValue(val)
	case nil:
		return otellog.StringValue("<nil>")
	default:
		b, _ := json.Marshal(val)
		return otellog.StringValue(string(b))
	}
}
