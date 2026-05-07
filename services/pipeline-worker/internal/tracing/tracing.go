package tracing

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/sdk/interceptor"
	tlog "go.temporal.io/sdk/log"
)

const (
	temporalTraceHeaderKey = "temporal-opentelemetry"
	instrumentationName    = "pipeline-worker"
)

type spanContextKey struct{}

type Runtime struct {
	enabled            bool
	clientInterceptors []interceptor.ClientInterceptor
	shutdown           func(context.Context) error
}

type temporalTracer struct {
	interceptor.BaseTracer
	tracer trace.Tracer
}

type temporalSpanRef struct {
	spanContext trace.SpanContext
}

type temporalSpan struct {
	span trace.Span
}

func Init(ctx context.Context, serviceName string, logger zerolog.Logger) (*Runtime, error) {
	if !readBoolEnv("TRACING_ENABLED", true) {
		logger.Info().Msg("Tracing disabled")
		return noopRuntime(), nil
	}

	endpoint := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	if endpoint == "" {
		logger.Info().Msg("Tracing disabled because OTLP endpoint is not configured")
		return noopRuntime(), nil
	}

	exporterOptions := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(endpoint),
	}
	if readBoolEnv("OTEL_EXPORTER_OTLP_INSECURE", true) {
		exporterOptions = append(exporterOptions, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx, exporterOptions...)
	if err != nil {
		return nil, fmt.Errorf("create OTLP trace exporter: %w", err)
	}

	resource := sdkresource.NewWithAttributes(
		"",
		attribute.String("service.name", serviceName),
		attribute.String("service.version", serviceVersion()),
		attribute.String("deployment.environment", deploymentEnvironment()),
	)

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	logger.Info().
		Str("otlp_endpoint", endpoint).
		Str("service_name", serviceName).
		Msg("Tracing enabled")

	return &Runtime{
		enabled: true,
		clientInterceptors: []interceptor.ClientInterceptor{
			interceptor.NewTracingInterceptor(&temporalTracer{
				tracer: otel.Tracer(serviceName + "/temporal"),
			}),
		},
		shutdown: provider.Shutdown,
	}, nil
}

func (r *Runtime) ClientInterceptors() []interceptor.ClientInterceptor {
	if r == nil {
		return nil
	}
	return r.clientInterceptors
}

func (r *Runtime) Shutdown(ctx context.Context) error {
	if r == nil || r.shutdown == nil {
		return nil
	}
	return r.shutdown(ctx)
}

func ContextLogger(ctx context.Context, logger zerolog.Logger) zerolog.Logger {
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		return logger
	}

	return logger.With().
		Str("trace_id", spanContext.TraceID().String()).
		Str("span_id", spanContext.SpanID().String()).
		Logger()
}

func StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	return otel.Tracer(instrumentationName).Start(ctx, name)
}

func (t *temporalTracer) Options() interceptor.TracerOptions {
	return interceptor.TracerOptions{
		SpanContextKey:          spanContextKey{},
		HeaderKey:               temporalTraceHeaderKey,
		AllowInvalidParentSpans: true,
	}
}

func (t *temporalTracer) UnmarshalSpan(serialized map[string]string) (interceptor.TracerSpanRef, error) {
	traceIDValue := strings.TrimSpace(serialized["trace_id"])
	spanIDValue := strings.TrimSpace(serialized["span_id"])
	if traceIDValue == "" || spanIDValue == "" {
		return nil, nil
	}

	traceID, err := trace.TraceIDFromHex(traceIDValue)
	if err != nil {
		return nil, fmt.Errorf("parse trace_id: %w", err)
	}

	spanID, err := trace.SpanIDFromHex(spanIDValue)
	if err != nil {
		return nil, fmt.Errorf("parse span_id: %w", err)
	}

	traceFlags := trace.FlagsSampled
	if configured := strings.TrimSpace(serialized["trace_flags"]); configured != "" {
		parsed, err := strconv.ParseUint(configured, 16, 8)
		if err != nil {
			return nil, fmt.Errorf("parse trace_flags: %w", err)
		}
		traceFlags = trace.TraceFlags(parsed)
	}

	var traceState trace.TraceState
	if configured := strings.TrimSpace(serialized["trace_state"]); configured != "" {
		traceState, err = trace.ParseTraceState(configured)
		if err != nil {
			return nil, fmt.Errorf("parse trace_state: %w", err)
		}
	}

	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: traceFlags,
		TraceState: traceState,
		Remote:     true,
	})
	if !spanContext.IsValid() {
		return nil, fmt.Errorf("invalid span context")
	}

	return temporalSpanRef{spanContext: spanContext}, nil
}

func (t *temporalTracer) MarshalSpan(span interceptor.TracerSpan) (map[string]string, error) {
	spanContext, ok := spanContextFromRef(span)
	if !ok || !spanContext.IsValid() {
		return map[string]string{}, nil
	}

	serialized := map[string]string{
		"trace_id":    spanContext.TraceID().String(),
		"span_id":     spanContext.SpanID().String(),
		"trace_flags": fmt.Sprintf("%02x", uint8(spanContext.TraceFlags())),
	}
	if traceState := spanContext.TraceState().String(); traceState != "" {
		serialized["trace_state"] = traceState
	}

	return serialized, nil
}

func (t *temporalTracer) SpanFromContext(ctx context.Context) interceptor.TracerSpan {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return nil
	}
	return temporalSpan{span: span}
}

func (t *temporalTracer) ContextWithSpan(ctx context.Context, span interceptor.TracerSpan) context.Context {
	switch typed := span.(type) {
	case temporalSpan:
		return trace.ContextWithSpan(ctx, typed.span)
	case *temporalSpan:
		return trace.ContextWithSpan(ctx, typed.span)
	default:
		return ctx
	}
}

func (t *temporalTracer) StartSpan(options *interceptor.TracerStartSpanOptions) (interceptor.TracerSpan, error) {
	ctx := context.Background()
	if parent, ok := spanContextFromRef(options.Parent); ok && parent.IsValid() {
		if parent.IsRemote() {
			ctx = trace.ContextWithRemoteSpanContext(ctx, parent)
		} else {
			ctx = trace.ContextWithSpanContext(ctx, parent)
		}
	}

	startOptions := []trace.SpanStartOption{
		trace.WithSpanKind(spanKindForOperation(options.Operation)),
		trace.WithAttributes(attributesFromOptions(options)...),
	}
	if !options.Time.IsZero() {
		startOptions = append(startOptions, trace.WithTimestamp(options.Time))
	}

	_, span := t.tracer.Start(ctx, t.SpanName(options), startOptions...)
	return temporalSpan{span: span}, nil
}

func (t *temporalTracer) GetLogger(logger tlog.Logger, ref interceptor.TracerSpanRef) tlog.Logger {
	spanContext, ok := spanContextFromRef(ref)
	if !ok || !spanContext.IsValid() {
		return logger
	}

	return tlog.With(
		logger,
		"trace_id", spanContext.TraceID().String(),
		"span_id", spanContext.SpanID().String(),
	)
}

func (s temporalSpan) Finish(options *interceptor.TracerFinishSpanOptions) {
	if options != nil && options.Error != nil {
		s.span.RecordError(options.Error)
		s.span.SetStatus(codes.Error, options.Error.Error())
	}
	s.span.End()
}

func attributesFromOptions(options *interceptor.TracerStartSpanOptions) []attribute.KeyValue {
	attributes := []attribute.KeyValue{
		attribute.String("temporal.operation", options.Operation),
		attribute.String("temporal.name", options.Name),
	}
	if options.IdempotencyKey != "" {
		attributes = append(attributes, attribute.String("temporal.idempotency_key", options.IdempotencyKey))
	}
	if len(options.Tags) == 0 {
		return attributes
	}

	keys := make([]string, 0, len(options.Tags))
	for key := range options.Tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		attributes = append(attributes, attribute.String(key, options.Tags[key]))
	}
	return attributes
}

func spanKindForOperation(operation string) trace.SpanKind {
	switch operation {
	case "ExecuteWorkflow", "SignalWorkflow", "QueryWorkflow", "CreateSchedule", "CancelWorkflow", "TerminateWorkflow", "UpdateWorkflow":
		return trace.SpanKindClient
	case "RunWorkflow", "RunActivity", "HandleSignal", "HandleQuery":
		return trace.SpanKindServer
	case "StartActivity", "StartChildWorkflow", "SignalChildWorkflow":
		return trace.SpanKindProducer
	default:
		return trace.SpanKindInternal
	}
}

func spanContextFromRef(ref interceptor.TracerSpanRef) (trace.SpanContext, bool) {
	switch typed := ref.(type) {
	case temporalSpanRef:
		return typed.spanContext, true
	case *temporalSpanRef:
		return typed.spanContext, true
	case temporalSpan:
		return typed.span.SpanContext(), true
	case *temporalSpan:
		return typed.span.SpanContext(), true
	case trace.SpanContext:
		return typed, typed.IsValid()
	default:
		return trace.SpanContext{}, false
	}
}

func noopRuntime() *Runtime {
	return &Runtime{
		shutdown: func(context.Context) error { return nil },
	}
}

func readBoolEnv(key string, defaultValue bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func serviceVersion() string {
	if configured := strings.TrimSpace(os.Getenv("SERVICE_VERSION")); configured != "" {
		return configured
	}
	return "dev"
}

func deploymentEnvironment() string {
	if configured := strings.TrimSpace(os.Getenv("APP_ENV")); configured != "" {
		return configured
	}
	return "development"
}
