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

// Worker processes pages using the appropriate scraper
type Worker struct {
	logger          zerolog.Logger
	sourceToScraper map[string]Scraper
	results         chan<- domain.ScrapeResult
	wg              sync.WaitGroup
}

func NewWorker(
	logger zerolog.Logger,
	sourceToScraper map[string]Scraper,
	results chan<- domain.ScrapeResult,
) *Worker {
	return &Worker{
		logger:          logger,
		sourceToScraper: sourceToScraper,
		results:         results,
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
			w.wg.Go(func() {
				result, err := w.processTask(ctx, task)
				if err != nil {
					w.logger.Error().Msgf("scraping %s failed: %v", task.URL, err)
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

	start := time.Now()
	result, err := scraper.Scrape(ctx, task)
	durationMs := int(time.Since(start).Milliseconds())

	if err != nil {
		w.logger.Error().Err(err).Str("url", task.URL).Msg("scraping failed")
		fmt.Printf("[DEBUG] worker: error scraping task %d: %v\n", task.ID, err)
		return &domain.ScrapeResult{
			TrackedPageID: task.ID,
			DurationMs:    durationMs,
			Err:           err,
		}, nil
	}

	result.TrackedPageID = task.ID
	result.DurationMs = durationMs
	fmt.Printf("[DEBUG] worker: success for task %d, resources=%d\n", task.ID, len(result.Resources))
	return result, nil
}
