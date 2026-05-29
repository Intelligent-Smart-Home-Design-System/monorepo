package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/main-pipeline/internal/pipeline"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/main-pipeline/workflows"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

func main() {
	log := zerolog.New(os.Stdout).With().Timestamp().Str("service", "api-gateway").Logger()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	temporalClient, err := client.Dial(client.Options{
		HostPort:  env("TEMPORAL_ADDRESS", "localhost:7233"),
		Namespace: env("TEMPORAL_NAMESPACE", "default"),
		Logger:    temporalLogger{log: log},
	})
	if err != nil {
		log.Fatal().Err(err).Msg("connect temporal")
	}
	defer temporalClient.Close()

	registry := prometheus.NewRegistry()
	startedTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "api_gateway_workflows_started_total", Help: "Total workflow start requests grouped by status."},
		[]string{"status"},
	)
	registry.MustRegister(prometheus.NewGoCollector(), prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}), startedTotal)

	apiServer := &http.Server{
		Addr:              env("HTTP_ADDRESS", ":8080"),
		ReadHeaderTimeout: 5 * time.Second,
		Handler:           buildAPI(log, temporalClient, startedTotal, os.Getenv("API_GATEWAY_TOKEN")),
	}
	metricsServer := &http.Server{
		Addr:              env("METRICS_ADDRESS", ":2116"),
		ReadHeaderTimeout: 5 * time.Second,
		Handler:           promhttp.HandlerFor(registry, promhttp.HandlerOpts{}),
	}

	go serve(log, "api", apiServer)
	go serve(log, "metrics", metricsServer)

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = apiServer.Shutdown(shutdownCtx)
	_ = metricsServer.Shutdown(shutdownCtx)
}

func buildAPI(log zerolog.Logger, temporalClient client.Client, startedTotal *prometheus.CounterVec, token string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("POST /start", func(w http.ResponseWriter, r *http.Request) {
		if !authorized(r, token) {
			startedTotal.WithLabelValues("unauthorized").Inc()
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var req pipeline.PipelineRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 10<<20))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			startedTotal.WithLabelValues("invalid").Inc()
			http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := validate(req); err != nil {
			startedTotal.WithLabelValues("invalid").Inc()
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.RequestID == "" {
			req.RequestID = fmt.Sprintf("manual-%s", time.Now().UTC().Format("20060102-150405"))
		}

		run, err := temporalClient.ExecuteWorkflow(r.Context(), client.StartWorkflowOptions{
			ID:        "main-pipeline-" + req.RequestID,
			TaskQueue: workflows.MainPipelineTaskQueue,
		}, workflows.MainPipelineWorkflow, req)
		if err != nil {
			startedTotal.WithLabelValues("failure").Inc()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		startedTotal.WithLabelValues("success").Inc()
		log.Info().Str("workflow_id", run.GetID()).Str("run_id", run.GetRunID()).Msg("workflow started")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{"workflow_id": run.GetID(), "run_id": run.GetRunID()})
	})
	resultHandler := func(w http.ResponseWriter, r *http.Request) {
		if !authorized(r, token) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		workflowID := r.PathValue("workflow_id")
		if workflowID == "" {
			workflowID = r.URL.Query().Get("workflow_id")
		}
		runID := r.URL.Query().Get("run_id")
		if workflowID == "" {
			http.Error(w, "workflow_id is required; use /result/{workflow_id} or /result?workflow_id=...", http.StatusBadRequest)
			return
		}

		description, err := temporalClient.DescribeWorkflowExecution(r.Context(), workflowID, runID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		status := description.WorkflowExecutionInfo.Status
		if status != enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"workflow_id": workflowID,
				"run_id":      description.WorkflowExecutionInfo.Execution.RunId,
				"status":      status.String(),
			})
			return
		}

		var result pipeline.PipelineResult
		if err := temporalClient.GetWorkflow(r.Context(), workflowID, runID).Get(r.Context(), &result); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	}
	mux.HandleFunc("GET /result", resultHandler)
	mux.HandleFunc("GET /result/{workflow_id}", resultHandler)
	return mux
}

func validate(req pipeline.PipelineRequest) error {
	if req.FloorPlan == nil {
		return errors.New("floor_plan is required")
	}
	if req.SelectedLevels == nil || len(req.SelectedLevels) == 0 {
		return errors.New("selected_levels is required")
	}
	if req.DeviceSelection == nil {
		return errors.New("device_selection is required")
	}
	return nil
}

func authorized(r *http.Request, token string) bool {
	if token == "" {
		return true
	}
	if r.Header.Get("X-API-Key") == token {
		return true
	}
	auth := r.Header.Get("Authorization")
	return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer ")) == token
}

func serve(log zerolog.Logger, name string, server *http.Server) {
	log.Info().Str("server", name).Str("address", server.Addr).Msg("listening")
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal().Err(err).Str("server", name).Msg("server stopped")
	}
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
