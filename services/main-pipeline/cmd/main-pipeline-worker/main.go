package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/main-pipeline/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/main-pipeline/internal/logging"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/main-pipeline/internal/metrics"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/main-pipeline/internal/temporalx"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/main-pipeline/internal/tracing"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/main-pipeline/workflows"
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/worker"
)

func main() {
	log.Logger = logging.New("main-pipeline-worker")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(configPath())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load pipeline configuration")
	}

	tracingRuntime, err := tracing.Init(ctx, "main-pipeline-worker", log.Logger)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize tracing")
	}
	defer shutdownTracing(tracingRuntime)

	metricsServer := metrics.NewServer(cfg.Metrics.ListenAddress, log.Logger)
	metricsServer.Start()
	defer shutdownMetrics(metricsServer)

	temporalClient, err := temporalx.Connect(ctx, temporalx.ConnectOptions{
		HostPort:        cfg.Temporal.HostPort,
		Namespace:       cfg.Temporal.Namespace,
		ConnectAttempts: cfg.Temporal.ConnectAttempts,
		Logger:          log.Logger,
		Interceptors:    tracingRuntime.ClientInterceptors(),
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Temporal")
	}
	defer temporalClient.Close()

	temporalWorker := worker.New(temporalClient, cfg.Temporal.TaskQueue, worker.Options{})
	temporalWorker.RegisterWorkflow(workflows.MainPipelineWorkflow)

	log.Info().
		Str("task_queue", cfg.Temporal.TaskQueue).
		Msg("Main pipeline worker started and waiting for workflows")

	if err := temporalWorker.Run(worker.InterruptCh()); err != nil {
		log.Fatal().Err(err).Msg("Main pipeline worker stopped with error")
	}
}

func configPath() string {
	if configured := strings.TrimSpace(os.Getenv("PIPELINE_CONFIG")); configured != "" {
		return configured
	}
	return "config/pipeline.toml"
}

func shutdownTracing(runtime *tracing.Runtime) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := runtime.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to flush tracing provider")
	}
}

func shutdownMetrics(server *metrics.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to stop metrics server")
	}
}
