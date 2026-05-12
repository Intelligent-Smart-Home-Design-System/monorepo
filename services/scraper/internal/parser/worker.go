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
}

type Worker[T any] struct {
	logger         zerolog.Logger
	pageType       domain.PageType
	repo           SnapshotRepository
	sourceToParser map[string]SourceParser[T]
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

func (w *Worker[T]) Parse(ctx context.Context) []T {
	var results []T

	for source, parser := range w.sourceToParser {
		if ctx.Err() != nil {
			return results
		}
		snapshots, err := w.repo.GetUnprocessedSnapshots(ctx, w.pageType.String(), source)
		if err != nil {
			w.logger.Error().Err(err).Str("source", source).Msg("failed to get unprocessed snapshots")
			continue
		}

		for _, snapshot := range snapshots {
			if ctx.Err() != nil {
				return results
			}
			files, err := extractArchive(snapshot.WARCBundle)
			if err != nil {
				w.logger.Error().Err(err).Int("snapshot_id", snapshot.ID).Msg("failed to extract archive")
				continue
			}

			result, err := parser.Parse(snapshot.ID, files)
			if err != nil {
				w.logger.Error().Err(err).Int("snapshot_id", snapshot.ID).Str("source", parser.Source()).Msg("failed to parse snapshot")
				continue
			}

			results = append(results, result)
		}
	}

	return results
}
