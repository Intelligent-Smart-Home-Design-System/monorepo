package worker

import (
	"context"
	"sync"

	"go.uber.org/zap"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
)

type Scraper interface {
	Scrape(ctx context.Context, task domain.ScrapeTask) (domain.ScrapeResult, error)
}

// Worker processes pages using the appropriate scraper
type Worker struct {
	logger          *zap.Logger
	sourceToScraper map[string]Scraper
	results         chan<- domain.ScrapeResult
	wg              sync.WaitGroup
}

func NewWorker(
	logger *zap.Logger,
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
				w.processPage(ctx, task)
			})
		}
	}
}

func (w *Worker) processPage(ctx context.Context, task domain.ScrapeTask) {
	scraper, ok := w.sourceToScraper[task.Source]
	if !ok {
		w.logger.Sugar().Errorf("scraper for source %s not found", task.Source)
		return
	}

	result, err := scraper.Scrape(ctx, task)
	if err != nil {
		w.logger.Sugar().Errorf("scraping %s failed: %s", err.Error())
		return
	}

	w.results <- result
}
