package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/main-pipeline/workflows"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	log := zerolog.New(os.Stdout).With().Timestamp().Str("service", "main-pipeline").Logger()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	temporalAddress := env("TEMPORAL_ADDRESS", "localhost:7233")
	namespace := env("TEMPORAL_NAMESPACE", "default")
	metricsAddress := env("METRICS_ADDRESS", ":2112")

	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewGoCollector(), prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	temporalClient, err := client.Dial(client.Options{
		HostPort:  temporalAddress,
		Namespace: namespace,
		Logger:    temporalLogger{log: log},
	})
	if err != nil {
		log.Fatal().Err(err).Msg("connect temporal")
	}
	defer temporalClient.Close()

	workflowWorker := worker.New(temporalClient, workflows.MainPipelineTaskQueue, worker.Options{})
	workflowWorker.RegisterWorkflow(workflows.MainPipelineWorkflow)
	go func() {
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

type temporalLogger struct{ log zerolog.Logger }

func (l temporalLogger) Debug(msg string, keyvals ...interface{}) { l.log.Debug().Fields(fields(keyvals...)).Msg(msg) }
func (l temporalLogger) Info(msg string, keyvals ...interface{})  { l.log.Info().Fields(fields(keyvals...)).Msg(msg) }
func (l temporalLogger) Warn(msg string, keyvals ...interface{})  { l.log.Warn().Fields(fields(keyvals...)).Msg(msg) }
func (l temporalLogger) Error(msg string, keyvals ...interface{}) { l.log.Error().Fields(fields(keyvals...)).Msg(msg) }

func fields(keyvals ...interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(keyvals)/2)
	for i := 0; i+1 < len(keyvals); i += 2 {
		if key, ok := keyvals[i].(string); ok {
			out[key] = keyvals[i+1]
		}
	}
	return out
}
