package metrics

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

type Collector struct {
	jobRunsTotal      *prometheus.CounterVec
	jobDurationSecond *prometheus.HistogramVec
}

type Server struct {
	address   string
	logger    zerolog.Logger
	registry  *prometheus.Registry
	collector *Collector
	server    *http.Server
}

func NewServer(address string, logger zerolog.Logger) *Server {
	registry := prometheus.NewRegistry()
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	collector := &Collector{
		jobRunsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "pipeline_job_runs_total",
				Help: "Total number of job container runs grouped by job and status.",
			},
			[]string{"job", "status"},
		),
		jobDurationSecond: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "pipeline_job_duration_seconds",
				Help:    "Duration of pipeline jobs executed through the Temporal worker.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"job"},
		),
	}

	registry.MustRegister(collector.jobRunsTotal, collector.jobDurationSecond)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	mux.HandleFunc("/healthz", func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("ok"))
	})

	return &Server{
		address:   address,
		logger:    logger,
		registry:  registry,
		collector: collector,
		server: &http.Server{
			Addr:              address,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

func (s *Server) Start() {
	go func() {
		s.logger.Info().Str("listen_address", s.address).Msg("Metrics server listening")
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error().Err(err).Msg("Metrics server stopped unexpectedly")
		}
	}()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s == nil || s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

func (s *Server) Collector() *Collector {
	if s == nil {
		return nil
	}
	return s.collector
}

func (c *Collector) RecordJob(jobName string, duration time.Duration, err error) {
	if c == nil {
		return
	}

	status := "success"
	if err != nil {
		status = "failure"
	}

	c.jobRunsTotal.WithLabelValues(jobName, status).Inc()
	c.jobDurationSecond.WithLabelValues(jobName).Observe(duration.Seconds())
}
