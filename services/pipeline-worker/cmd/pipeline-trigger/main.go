package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/pipeline-worker/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/pipeline-worker/internal/logging"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/pipeline-worker/internal/pipeline"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/pipeline-worker/internal/temporalx"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/pipeline-worker/internal/tracing"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/pipeline-worker/workflows"
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/client"
)

func main() {
	log.Logger = logging.New("pipeline-trigger")
	ctx := context.Background()

	cfg, err := config.Load(configPath())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load pipeline configuration")
	}

	tracingRuntime, err := tracing.Init(ctx, "pipeline-trigger", log.Logger)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize tracing")
	}
	defer shutdownTracing(tracingRuntime)

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

	workflowID := fmt.Sprintf("%s-manual-%s", cfg.Temporal.WorkflowIDPrefix, time.Now().UTC().Format("20060102-150405"))
	run, err := temporalClient.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: cfg.Temporal.TaskQueue,
	}, workflows.CatalogPipelineWorkflow, cfg.WorkflowInput())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start catalog pipeline workflow")
	}

	log.Info().
		Str("workflow_id", run.GetID()).
		Str("run_id", run.GetRunID()).
		Msg("Catalog pipeline workflow started")

	if !waitForCompletion() {
		return
	}

	var result pipeline.WorkflowResult
	if err := run.Get(ctx, &result); err != nil {
		log.Fatal().Err(err).Msg("Catalog pipeline workflow failed")
	}

	log.Info().
		Int("jobs", len(result.Jobs)).
		Time("completed_at", result.CompletedAt).
		Msg("Catalog pipeline workflow completed")
}

func configPath() string {
	if configured := strings.TrimSpace(os.Getenv("PIPELINE_CONFIG")); configured != "" {
		return configured
	}
	return "config/pipeline.toml"
}

func waitForCompletion() bool {
	value := strings.TrimSpace(os.Getenv("WAIT_FOR_COMPLETION"))
	if value == "" {
		return true
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return true
	}
	return parsed
}

func shutdownTracing(runtime *tracing.Runtime) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := runtime.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to flush tracing provider")
	}
}
