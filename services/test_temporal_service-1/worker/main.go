package main

import (
	"os"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"temporal-go-project/activities"
	"temporal-go-project/internal/logging"
	"temporal-go-project/workflows"
)

const (
	TaskQueueName = "greeting-task-queue"
)

func main() {
	log.Logger = logging.New("worker")

	temporalHost := os.Getenv("TEMPORAL_HOST")
	if temporalHost == "" {
		temporalHost = "localhost:7233"
	}

	log.Info().
		Str("temporal_host", temporalHost).
		Msg("Connecting to Temporal server")

	c, err := connectTemporalClient(temporalHost)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create Temporal client")
	}
	defer c.Close()

	w := worker.New(c, TaskQueueName, worker.Options{})

	w.RegisterWorkflow(workflows.GreetingWorkflow)
	w.RegisterWorkflow(workflows.SimpleWorkflow)

	greetingActivity := &activities.GreetingActivity{}
	w.RegisterActivity(greetingActivity.SayHello)
	w.RegisterActivity(greetingActivity.ProcessData)
	w.RegisterActivity(greetingActivity.SendNotification)

	log.Info().
		Str("task_queue", TaskQueueName).
		Msg("Worker started and waiting for tasks")

	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start worker")
	}

	log.Info().Msg("Worker stopped")
}

func connectTemporalClient(temporalHost string) (client.Client, error) {
	attempts := 30
	if configured := os.Getenv("TEMPORAL_CONNECT_ATTEMPTS"); configured != "" {
		if parsed, err := strconv.Atoi(configured); err == nil && parsed > 0 {
			attempts = parsed
		}
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		c, err := client.Dial(client.Options{
			HostPort: temporalHost,
			Logger:   logging.NewTemporalLogger(log.Logger),
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
