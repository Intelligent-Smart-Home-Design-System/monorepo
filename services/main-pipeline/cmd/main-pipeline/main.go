package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/main-pipeline/internal/pipeline"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/main-pipeline/workflows"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/shared/telemetry/go/otelsetup"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/shared/telemetry/go/otelzerolog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/contrib/opentelemetry"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/worker"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	telemetry := otelsetup.New(ctx, "main-pipeline")
	defer telemetry.Shutdown()
	log := telemetry.Log

	temporalAddress := env("TEMPORAL_ADDRESS", "localhost:7233")
	namespace := env("TEMPORAL_NAMESPACE", "default")
	metricsAddress := env("METRICS_ADDRESS", ":2112")

	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewGoCollector(), prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	tracingInterceptor, err := opentelemetry.NewTracingInterceptor(opentelemetry.TracerOptions{})
	if err != nil {
		log.Warn().Err(err).Msg("failed to create Temporal tracing interceptor, tracing disabled for workflows")
	}

	dialOpts := client.Options{
		HostPort:      temporalAddress,
		Namespace:     namespace,
		Logger:        otelzerolog.NewTemporal(log),
		DataConverter: pipeline.NewDataConverter(),
	}
	if tracingInterceptor != nil {
		dialOpts.Interceptors = []interceptor.ClientInterceptor{tracingInterceptor}
	}

	temporalClient, err := client.Dial(dialOpts)
	if err != nil {
		log.Fatal().Err(err).Msg("connect temporal")
	}
	defer temporalClient.Close()

	workerOpts := worker.Options{}
	if tracingInterceptor != nil {
		workerOpts.Interceptors = []interceptor.WorkerInterceptor{tracingInterceptor}
	}

	workflowWorker := worker.New(temporalClient, workflows.MainPipelineTaskQueue, workerOpts)
	workflowWorker.RegisterWorkflow(workflows.MainPipelineWorkflow)
	go func() {
		log.Info().Msg("workflow worker started")
		if err := workflowWorker.Run(worker.InterruptCh()); err != nil {
			log.Fatal().Err(err).Msg("run workflow worker")
		}
	}()

	metricsServer := &http.Server{
		Addr:              metricsAddress,
		ReadHeaderTimeout: 5 * time.Second,
		Handler:           promhttp.HandlerFor(registry, promhttp.HandlerOpts{}),
	}

	go func() {
		log.Info().Str("address", metricsAddress).Msg("metrics listening")
		if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("metrics stopped")
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = metricsServer.Shutdown(shutdownCtx)
	workflowWorker.Stop()
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
