package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/temporalworker"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/shared/telemetry/go/otelsetup"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/shared/telemetry/go/otelzerolog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/contrib/opentelemetry"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/worker"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	telemetry := otelsetup.New(ctx, "layout-worker")
	defer telemetry.Shutdown()
	log := telemetry.Log

	activities, err := temporalworker.NewActivities(
		env("LAYOUT_TRACKS_CONFIG", "internal/configs/tracks.json"),
		env("LAYOUT_DEVICES_CONFIG", "internal/configs/devices.json"),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("init activities")
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewGoCollector(), prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	metricsServer := &http.Server{
		Addr:              env("METRICS_ADDRESS", ":2114"),
		Handler:           promhttp.HandlerFor(registry, promhttp.HandlerOpts{}),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("metrics stopped")
		}
	}()

	tracingInterceptor, err := opentelemetry.NewTracingInterceptor(opentelemetry.TracerOptions{})
	if err != nil {
		log.Warn().Err(err).Msg("failed to create Temporal tracing interceptor, tracing disabled for activities")
	}

	dialOpts := client.Options{
		HostPort:  env("TEMPORAL_ADDRESS", "localhost:7233"),
		Namespace: env("TEMPORAL_NAMESPACE", "default"),
		Logger:    otelzerolog.NewTemporal(log),
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

	layoutWorker := worker.New(temporalClient, env("TEMPORAL_TASK_QUEUE", "layout"), workerOpts)
	layoutWorker.RegisterActivityWithOptions(activities.PlaceDevices, activity.RegisterOptions{Name: "place_devices"})

	go func() {
		log.Info().Msg("layout worker started")
		if err := layoutWorker.Run(worker.InterruptCh()); err != nil {
			log.Fatal().Err(err).Msg("run worker")
		}
	}()

	<-ctx.Done()
	layoutWorker.Stop()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = metricsServer.Shutdown(shutdownCtx)
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
