package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	layoutworker "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/worker"
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	tworker "go.temporal.io/sdk/worker"
)

func main() {
	settings := layoutworker.LoadSettings()
	log.Logger = layoutworker.NewLogger(settings.ServiceName)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	tracingRuntime, err := layoutworker.InitTracing(ctx, settings, log.Logger)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize tracing")
	}
	defer shutdownTracing(tracingRuntime)

	metricsServer := layoutworker.NewMetricsServer(settings.MetricsListenAddress, log.Logger)
	metricsServer.Start()
	defer shutdownMetrics(metricsServer)

	temporalClient, err := client.Dial(client.Options{
		HostPort:  settings.TemporalAddress,
		Namespace: settings.TemporalNamespace,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Temporal")
	}
	defer temporalClient.Close()

	service := layoutworker.NewActivityService(settings, metricsServer.Collector(), log.Logger)
	temporalWorker := tworker.New(temporalClient, settings.TemporalTaskQueue, tworker.Options{
		MaxConcurrentActivityExecutionSize: settings.MaxConcurrentActivities,
	})
	temporalWorker.RegisterActivityWithOptions(service.BuildLayout, activity.RegisterOptions{
		Name: "layout.build_layout",
	})

	log.Info().
		Str("task_queue", settings.TemporalTaskQueue).
		Int("max_concurrent_activities", settings.MaxConcurrentActivities).
		Int64("compute_concurrency", settings.ComputeConcurrency).
		Msg("Layout worker started")

	if err := temporalWorker.Run(tworker.InterruptCh()); err != nil {
		log.Fatal().Err(err).Msg("Layout worker stopped with error")
	}
}

func shutdownTracing(runtime *layoutworker.TracingRuntime) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := runtime.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to flush tracing provider")
	}
}

func shutdownMetrics(server *layoutworker.MetricsServer) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to stop metrics server")
	}
}
