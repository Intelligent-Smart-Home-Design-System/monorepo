package activities

import (
	"context"
	"strings"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/pipeline-worker/internal/docker"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/pipeline-worker/internal/metrics"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/pipeline-worker/internal/pipeline"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/pipeline-worker/internal/tracing"
	"github.com/rs/zerolog"
)

type RunContainerActivity struct {
	runner  *docker.Runner
	metrics *metrics.Collector
	logger  zerolog.Logger
}

func NewRunContainerActivity(runner *docker.Runner, collector *metrics.Collector, logger zerolog.Logger) *RunContainerActivity {
	return &RunContainerActivity{
		runner:  runner,
		metrics: collector,
		logger:  logger,
	}
}

func (a *RunContainerActivity) RunContainer(ctx context.Context, params pipeline.RunContainerParams) (*pipeline.RunContainerResult, error) {
	ctx, span := tracing.StartSpan(ctx, "activity.run_container")
	defer span.End()

	logger := tracing.ContextLogger(ctx, a.logger).With().
		Str("job", params.Name).
		Str("image", params.Image).
		Logger()

	startedAt := time.Now()
	result, err := a.runner.Run(ctx, params)
	a.metrics.RecordJob(params.Name, time.Since(startedAt), err)

	if err != nil {
		if result != nil && strings.TrimSpace(result.Logs) != "" {
			logger.Error().
				Err(err).
				Str("logs", result.Logs).
				Msg("Job container failed")
		} else {
			logger.Error().Err(err).Msg("Job container failed")
		}
		return result, err
	}

	if result != nil && strings.TrimSpace(result.Logs) != "" {
		logger.Info().Str("logs", result.Logs).Msg("Job container succeeded")
	} else {
		logger.Info().Msg("Job container succeeded")
	}

	return result, nil
}
