// Package otelzerolog bridges zerolog with Temporal SDK logging and
// OpenTelemetry trace context in log records.
package otelzerolog

import (
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

// TracingHook injects trace_id and span_id from the active OTEL span into
// zerolog events that carry a Go context (Event.Ctx).
type TracingHook struct{}

func (TracingHook) Run(e *zerolog.Event, _ zerolog.Level, _ string) {
	ctx := e.GetCtx()
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		e.Str("trace_id", span.SpanContext().TraceID().String())
		e.Str("span_id", span.SpanContext().SpanID().String())
	}
}

// Temporal adapts zerolog to go.temporal.io/sdk/log.Logger.
type Temporal struct {
	Log zerolog.Logger
}

func NewTemporal(log zerolog.Logger) Temporal {
	return Temporal{Log: log}
}

func (l Temporal) Debug(msg string, keyvals ...interface{}) {
	l.Log.Debug().Fields(keyValues(keyvals...)).Msg(msg)
}

func (l Temporal) Info(msg string, keyvals ...interface{}) {
	l.Log.Info().Fields(keyValues(keyvals...)).Msg(msg)
}

func (l Temporal) Warn(msg string, keyvals ...interface{}) {
	l.Log.Warn().Fields(keyValues(keyvals...)).Msg(msg)
}

func (l Temporal) Error(msg string, keyvals ...interface{}) {
	l.Log.Error().Fields(keyValues(keyvals...)).Msg(msg)
}

func keyValues(keyvals ...interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(keyvals)/2)
	for i := 0; i+1 < len(keyvals); i += 2 {
		if key, ok := keyvals[i].(string); ok {
			out[key] = keyvals[i+1]
		}
	}
	return out
}
