package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
	"temporal-go-project/internal/logging"
	"temporal-go-project/internal/tracing"
	"temporal-go-project/workflows"
)

const (
	TaskQueueName = "greeting-task-queue"
)

func main() {
	log.Logger = logging.New("client")
	ctx := context.Background()

	tracingRuntime, err := tracing.Init(ctx, "temporal-client", log.Logger)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize tracing")
	}
	defer shutdownTracing(tracingRuntime)

	temporalHost := os.Getenv("TEMPORAL_HOST")
	if temporalHost == "" {
		temporalHost = "localhost:7233"
	}

	c, err := connectTemporalClient(temporalHost, tracingRuntime.ClientInterceptors())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create Temporal client")
	}
	defer c.Close()

	workflowInput := workflows.GreetingWorkflowInput{
		Name: "Temporal User",
		Data: "Test data payload for Temporal Workflow processing",
	}

	workflowIDSuffix := workflowIDSuffix()
	workflowOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("greeting-workflow-%s", workflowIDSuffix),
		TaskQueue: TaskQueueName,
	}

	log.Info().Msg("Starting GreetingWorkflow")
	we, err := c.ExecuteWorkflow(ctx, workflowOptions, workflows.GreetingWorkflow, workflowInput)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to execute GreetingWorkflow")
	}

	log.Info().
		Str("workflow_id", we.GetID()).
		Str("run_id", we.GetRunID()).
		Msg("GreetingWorkflow started")

	var result workflows.GreetingWorkflowResult
	err = we.Get(ctx, &result)
	if err != nil {
		log.Fatal().Err(err).Msg("GreetingWorkflow failed")
	}

	log.Info().
		Str("greeting_message", result.GreetingMessage).
		Str("processed_data", result.ProcessedData).
		Time("completed_at", result.CompletedAt).
		Msg("GreetingWorkflow completed")

	log.Info().Msg("Starting SimpleWorkflow")
	simpleOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("simple-workflow-%s", workflowIDSuffix),
		TaskQueue: TaskQueueName,
	}

	we2, err := c.ExecuteWorkflow(ctx, simpleOptions, workflows.SimpleWorkflow, "Simple User")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to execute SimpleWorkflow")
	}

	var simpleResult string
	err = we2.Get(ctx, &simpleResult)
	if err != nil {
		log.Fatal().Err(err).Msg("SimpleWorkflow failed")
	}

	log.Info().Str("result", simpleResult).Msg("SimpleWorkflow completed")
}

func connectTemporalClient(temporalHost string, clientInterceptors []interceptor.ClientInterceptor) (client.Client, error) {
	attempts := 30
	if configured := os.Getenv("TEMPORAL_CONNECT_ATTEMPTS"); configured != "" {
		if parsed, err := strconv.Atoi(configured); err == nil && parsed > 0 {
			attempts = parsed
		}
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		c, err := client.Dial(client.Options{
			HostPort:     temporalHost,
			Logger:       logging.NewTemporalLogger(log.Logger),
			Interceptors: clientInterceptors,
		})
		if err == nil {
			return c, nil
		}

		lastErr = err
		log.Warn().
			Int("attempt", attempt).
			Int("max_attempts", attempts).
			Err(err).
			Msg("Temporal is not ready yet, retrying connection")
		time.Sleep(2 * time.Second)
	}

	return nil, lastErr
}

func workflowIDSuffix() string {
	if configured := strings.TrimSpace(os.Getenv("WORKFLOW_ID_SUFFIX")); configured != "" {
		return configured
	}
	return time.Now().UTC().Format("20060102-150405")
}

func shutdownTracing(runtime *tracing.Runtime) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := runtime.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to flush tracing provider")
	}
}
