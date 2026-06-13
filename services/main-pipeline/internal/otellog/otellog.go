// Package otellog provides a zerolog io.Writer that sends log records to an
// OpenTelemetry Collector via OTLP/gRPC.
//
// Usage:
//
//	shutdown, writer, err := otellog.NewOTLPWriter(ctx, "api-gateway")
//	if err != nil { ... }
//	defer shutdown(context.Background())
//	log := zerolog.New(zerolog.MultiLevelWriter(os.Stdout, writer)).With().Timestamp().Logger()
package otellog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ShutdownFunc gracefully flushes and shuts down the OTLP log pipeline.
type ShutdownFunc func(ctx context.Context) error

// NewOTLPWriter creates an io.Writer that forwards zerolog JSON lines to the
// OTEL Collector specified by OTEL_EXPORTER_OTLP_ENDPOINT (env var).
// If the env var is empty, it returns a no-op writer so the service still works
// without the monitoring stack.
func NewOTLPWriter(ctx context.Context, serviceName string) (ShutdownFunc, *Writer, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		return func(context.Context) error { return nil }, &Writer{}, nil
	}

	conn, err := grpc.NewClient(endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("otellog: grpc dial %s: %w", endpoint, err)
	}

	exporter, err := otlploggrpc.New(ctx, otlploggrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, nil, fmt.Errorf("otellog: create exporter: %w", err)
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

	logger := provider.Logger(serviceName)

	shutdown := func(ctx context.Context) error {
		if err := provider.Shutdown(ctx); err != nil {
			return err
		}
		return conn.Close()
	}

	return shutdown, &Writer{logger: logger}, nil
}

// Writer implements io.Writer. Each Write call is expected to receive a single
// zerolog JSON line. It parses the JSON, extracts standard fields (level,
// message, time) and forwards the rest as OTLP log attributes.
type Writer struct {
	logger otellog.Logger
	mu     sync.Mutex
}

// Write parses a zerolog JSON line and emits it as an OTLP log record.
func (w *Writer) Write(p []byte) (int, error) {
	if w.logger == nil {
		return len(p), nil // no-op when OTLP is not configured
	}

	var fields map[string]interface{}
	if err := json.Unmarshal(p, &fields); err != nil {
		// If we can't parse the JSON, send the raw line as body.
		var rec otellog.Record
		rec.SetBody(otellog.StringValue(string(p)))
		rec.SetTimestamp(time.Now())
		w.emit(rec)
		return len(p), nil
	}

	var rec otellog.Record

	// Extract and set the log body (message).
	if msg, ok := fields["message"].(string); ok {
		rec.SetBody(otellog.StringValue(msg))
		delete(fields, "message")
	}

	// Extract and set severity from zerolog level field.
	if lvl, ok := fields["level"].(string); ok {
		rec.SetSeverity(mapSeverity(lvl))
		rec.SetSeverityText(lvl)
		delete(fields, "level")
	}

	// Extract and set timestamp.
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

	// All remaining fields become OTLP attributes.
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
