package parser

import (
	"context"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/rs/zerolog"
)

type SourceParser[T any] interface {
	Source() string
	Parse(pageSnapshotId int, files []*ArchiveFile) (T, error)
}

type SnapshotRepository interface {
	GetUnprocessedSnapshots(ctx context.Context, source, pageType string) ([]*domain.PageSnapshot, error)
	SetProcessed(snapshotId int) error
}

type Worker[T any] struct {
	logger         zerolog.Logger
	pageType       domain.PageType
	repo           SnapshotRepository
	sourceToParser map[string]SourceParser[T]
	metrics        ParseMetrics
	job            string
}

func NewWorker[T any](
	logger zerolog.Logger,
	pageType domain.PageType,
	repo SnapshotRepository,
	parsers []SourceParser[T],
) *Worker[T] {
	sourceToParser := make(map[string]SourceParser[T], len(parsers))
	for _, p := range parsers {
		sourceToParser[p.Source()] = p
	}
	return &Worker[T]{
		logger:         logger,
		pageType:       pageType,
		repo:           repo,
		sourceToParser: sourceToParser,
	}
}

func (w *Worker[T]) UseMetrics(metrics ParseMetrics, job string) {
	w.metrics = metrics
	w.job = job
}

func (w *Worker[T]) Parse(ctx context.Context) []T {
	var all []*domain.PageSnapshot
	for source := range w.sourceToParser {
		if ctx.Err() != nil {
			break
		}
		snapshots, err := w.repo.GetUnprocessedSnapshots(ctx, w.pageType.String(), source)
		if err != nil {
			w.logger.Error().Err(err).Str("source", source).Msg("failed to get unprocessed snapshots")
			continue
		}
		all = append(all, snapshots...)
	}
	return w.ParseSnapshots(ctx, all)
}

func (w *Worker[T]) ParseSnapshots(ctx context.Context, snapshots []*domain.PageSnapshot) []T {
	var results []T

	for _, snapshot := range snapshots {
		if ctx.Err() != nil {
			return results
		}
		parser, ok := w.sourceToParser[snapshot.SourceName]
		if !ok {
			continue
		}
		snapshotLog := w.logger.With().
			Str("source", snapshot.SourceName).
			Str("page_type", w.pageType.String()).
			Logger()

		files, err := ExtractArchive(snapshot.WARCBundle)
		if err != nil {
			snapshotLog.Error().Err(err).Int("snapshot_id", snapshot.ID).Msg("failed to extract archive")
			if w.metrics != nil {
				w.metrics.AddParseSnapshots(ctx, snapshot.SourceName, w.pageType.String(), w.job, "parse_error", "", 1)
			}
			continue
		}

		result, parseErr := parser.Parse(snapshot.ID, files)
		if err = w.repo.SetProcessed(snapshot.ID); err != nil {
			snapshotLog.Error().Err(err).Int("snapshot_id", snapshot.ID).Msg("failed to set snapshot as processed")
		}
		if parseErr != nil {
			snapshotLog.Error().Err(parseErr).Int("snapshot_id", snapshot.ID).Msg("failed to parse snapshot")
			if w.metrics != nil {
				w.metrics.AddParseSnapshots(ctx, parser.Source(), w.pageType.String(), w.job, "parse_error", "", 1)
			}
			continue
		}

		if w.metrics != nil {
			w.metrics.AddParseSnapshots(ctx, parser.Source(), w.pageType.String(), w.job, "parsed", "", 1)
		}

		results = append(results, result)
	}

	return results
}
