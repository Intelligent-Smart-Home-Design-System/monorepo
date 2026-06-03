package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/pipeline-worker/activities"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/pipeline-worker/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/pipeline-worker/internal/docker"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/pipeline-worker/internal/logging"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/pipeline-worker/internal/metrics"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/pipeline-worker/internal/temporalx"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/pipeline-worker/internal/tracing"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/pipeline-worker/workflows"
	"github.com/rs/zerolog/log"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	log.Logger = logging.New("pipeline-worker")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(configPath())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load pipeline configuration")
	}

	tracingRuntime, err := tracing.Init(ctx, "pipeline-worker", log.Logger)
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

	runner, err := docker.NewRunner(docker.Settings{
		Host:            cfg.Docker.Host,
		NetworkName:     cfg.Docker.NetworkName,
		ContainerPrefix: cfg.Docker.ContainerPrefix,
		AutoRemove:      cfg.Docker.AutoRemove,
		ConfigRoot:      cfg.ConfigRoot(),
	}, log.Logger)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create Docker runner")
	}
	defer func() {
		if err := runner.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close Docker runner")
		}
	}()

	if cfg.Temporal.ScheduleEnabled {
		if err := ensureSchedule(ctx, temporalClient, cfg); err != nil {
			log.Fatal().Err(err).Msg("Failed to ensure Temporal schedule")
		}
	}

	containerActivity := activities.NewRunContainerActivity(runner, metricsServer.Collector(), log.Logger)
	temporalWorker := worker.New(temporalClient, cfg.Temporal.TaskQueue, worker.Options{})
	temporalWorker.RegisterWorkflow(workflows.CatalogPipelineWorkflow)
	temporalWorker.RegisterActivityWithOptions(containerActivity.RunContainer, activity.RegisterOptions{
		Name: "RunContainer",
	})

	log.Info().
		Str("task_queue", cfg.Temporal.TaskQueue).
		Str("schedule_id", cfg.Temporal.ScheduleID).
		Msg("Pipeline worker started and waiting for tasks")

	if err := temporalWorker.Run(worker.InterruptCh()); err != nil {
		log.Fatal().Err(err).Msg("Pipeline worker stopped with error")
	}
}

func ensureSchedule(ctx context.Context, temporalClient client.Client, cfg *config.Config) error {
	scheduleOptions := client.ScheduleOptions{
		ID: cfg.Temporal.ScheduleID,
		Spec: client.ScheduleSpec{
			CronExpressions: []string{cfg.Temporal.ScheduleCron},
			TimeZoneName:    cfg.Temporal.ScheduleTimezone,
		},
		Action: &client.ScheduleWorkflowAction{
			ID:        fmt.Sprintf("%s-scheduled", cfg.Temporal.WorkflowIDPrefix),
			Workflow:  workflows.CatalogPipelineWorkflow,
			TaskQueue: cfg.Temporal.TaskQueue,
			Args:      []interface{}{cfg.WorkflowInput()},
		},
		Overlap: enumspb.SCHEDULE_OVERLAP_POLICY_SKIP,
	}

	_, err := temporalClient.ScheduleClient().Create(ctx, scheduleOptions)
	if err == nil {
		log.Info().
			Str("schedule_id", cfg.Temporal.ScheduleID).
			Str("cron", cfg.Temporal.ScheduleCron).
			Str("timezone", cfg.Temporal.ScheduleTimezone).
			Msg("Temporal schedule created")
		return nil
	}

	var alreadyExists *serviceerror.AlreadyExists
	if errors.As(err, &alreadyExists) {
		log.Info().
			Str("schedule_id", cfg.Temporal.ScheduleID).
			Msg("Temporal schedule already exists")
		return nil
	}

	return err
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
