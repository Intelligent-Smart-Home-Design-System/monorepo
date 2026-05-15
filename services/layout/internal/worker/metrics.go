package worker

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
	runsTotal      *prometheus.CounterVec
	durationSecond prometheus.Histogram
	concurrentRuns prometheus.Gauge
}

type MetricsServer struct {
	address   string
	logger    zerolog.Logger
	collector *Collector
	server    *http.Server
}

func NewMetricsServer(address string, logger zerolog.Logger) *MetricsServer {
	registry := prometheus.NewRegistry()
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	collector := &Collector{
		runsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "layout_activity_runs_total",
				Help: "Total number of layout activity runs grouped by status.",
			},
			[]string{"status"},
		),
		durationSecond: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "layout_activity_duration_seconds",
				Help:    "Duration of layout activities.",
				Buckets: prometheus.DefBuckets,
			},
		),
		concurrentRuns: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "layout_activity_concurrent_runs",
				Help: "Number of currently running layout activities.",
			},
		),
	}

	registry.MustRegister(collector.runsTotal, collector.durationSecond, collector.concurrentRuns)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	mux.HandleFunc("/healthz", func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("ok"))
	})

	return &MetricsServer{
		address:   address,
		logger:    logger,
		collector: collector,
		server: &http.Server{
			Addr:              address,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

func (s *MetricsServer) Start() {
	go func() {
		s.logger.Info().Str("listen_address", s.address).Msg("Metrics server listening")
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error().Err(err).Msg("Metrics server stopped unexpectedly")
		}
	}()
}

func (s *MetricsServer) Shutdown(ctx context.Context) error {
	if s == nil || s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

func (s *MetricsServer) Collector() *Collector {
	if s == nil {
		return nil
	}
	return s.collector
}

func (c *Collector) record(status string, duration time.Duration) {
	c.runsTotal.WithLabelValues(status).Inc()
	c.durationSecond.Observe(duration.Seconds())
}
