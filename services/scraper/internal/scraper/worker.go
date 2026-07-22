package scraper

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/rs/zerolog"
)

type Scraper interface {
	Scrape(ctx context.Context, task domain.ScrapeTask) (*domain.ScrapeResult, error)
}

// defaultMaxConcurrency caps in-flight scrape goroutines when the caller
// doesn't configure one. Sources like dns share a single browser tab/proxy
// session per Scraper instance, so unbounded concurrency just piles up
// contention (and, during a proxy rotation, races on the shared client) rather
// than doing real parallel work.
const defaultMaxConcurrency = 3

// Worker processes pages using the appropriate scraper
type Worker struct {
	logger          zerolog.Logger
	sourceToScraper map[string]Scraper
	results         chan<- domain.ScrapeResult
	wg              sync.WaitGroup
	sem             chan struct{}
}

func NewWorker(
	logger zerolog.Logger,
	sourceToScraper map[string]Scraper,
	results chan<- domain.ScrapeResult,
	maxConcurrency int,
) *Worker {
	if maxConcurrency <= 0 {
		maxConcurrency = defaultMaxConcurrency
	}
	return &Worker{
		logger:          logger,
		sourceToScraper: sourceToScraper,
		results:         results,
		sem:             make(chan struct{}, maxConcurrency),
	}
}

func (w *Worker) Run(ctx context.Context, tasks <-chan domain.ScrapeTask) {
	for {
		select {
		case <-ctx.Done():
			w.wg.Wait()
			close(w.results)
			return
		case task, ok := <-tasks:
			if !ok {
				w.wg.Wait()
				close(w.results)
				return
			}
			select {
			case w.sem <- struct{}{}:
			case <-ctx.Done():
				w.wg.Wait()
				close(w.results)
				return
			}
			w.wg.Go(func() {
				defer func() { <-w.sem }()
				result, err := w.processTask(ctx, task)
				if err != nil {
					w.logger.Error().
						Str("source", task.Source).
						Str("page_type", task.PageType.String()).
						Str("url", task.URL).
						Err(err).
						Msg("scraping failed")
					w.results <- domain.ScrapeResult{Err: err}
					return
				}
				w.results <- *result
			})
		}
	}
}

func (w *Worker) processTask(ctx context.Context, task domain.ScrapeTask) (*domain.ScrapeResult, error) {
	scraper, ok := w.sourceToScraper[task.Source]
	if !ok {
		return nil, fmt.Errorf("scraper for source %s not found", task.Source)
	}

	taskLog := w.logger.With().
		Str("source", task.Source).
		Str("page_type", task.PageType.String()).
		Logger()

	start := time.Now()
	result, err := scraper.Scrape(ctx, task)
	durationMs := int(time.Since(start).Milliseconds())

	if err != nil {
		taskLog.Error().Err(err).Str("url", task.URL).Msg("scraping failed")
		taskLog.Debug().Int("task_id", task.ID).Err(err).Msg("worker: error scraping task")
		return &domain.ScrapeResult{
			TrackedPageID: task.ID,
			DurationMs:    durationMs,
			Err:           err,
		}, nil
	}

	result.TrackedPageID = task.ID
	result.DurationMs = durationMs
	taskLog.Debug().Int("task_id", task.ID).Int("resources", len(result.Resources)).Msg("worker: success for task")
	return result, nil
}
