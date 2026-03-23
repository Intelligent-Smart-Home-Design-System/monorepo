package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestScrapeTaskDispatch(t *testing.T) {
	firstScraper := NewMockScraper(t)
	secondScraper := NewMockScraper(t)
	sourceToScraper := map[string]Scraper{
		"first_source":  firstScraper,
		"second_source": secondScraper,
	}

	resultsCh := make(chan domain.ScrapeResult)

	tasks := []domain.ScrapeTask{
		{
			Source:   "second_source",
			PageType: "type_1",
			URL:      "http://example.com",
		},
		{
			Source:   "first_source",
			PageType: "type_1",
			URL:      "http://example.com/3",
		},
		{
			Source:   "second_source",
			PageType: "type_2",
			URL:      "http://example.com/1",
		},
		{
			Source:   "first_source",
			PageType: "type_3",
			URL:      "http://example.com/4",
		},
		{
			Source:   "unknown_source",
			PageType: "type_3",
			URL:      "http://example.com/4",
		},
	}

	expectedResults := []domain.ScrapeResult{makeResult("0"), makeResult("1"), makeResult("2"), makeResult("3")}

	firstScraper.EXPECT().Scrape(mock.Anything, tasks[1]).Return(expectedResults[0], nil)
	firstScraper.EXPECT().Scrape(mock.Anything, tasks[3]).Return(expectedResults[1], nil)
	secondScraper.EXPECT().Scrape(mock.Anything, tasks[0]).Return(expectedResults[2], nil)
	secondScraper.EXPECT().Scrape(mock.Anything, tasks[2]).Return(expectedResults[3], nil)

	tasksCh := make(chan domain.ScrapeTask, len(tasks))
	for _, task := range tasks {
		tasksCh <- task
	}
	close(tasksCh)

	worker := NewWorker(zap.NewNop(), sourceToScraper, resultsCh)

	go worker.Run(t.Context(), tasksCh)

	var results []domain.ScrapeResult
	for result := range resultsCh {
		results = append(results, result)
	}

	require.ElementsMatch(t, expectedResults, results)
}

func TestScrapeFailure(t *testing.T) {
	scraper := NewMockScraper(t)
	sourceToScraper := map[string]Scraper{
		"source": scraper,
	}

	resultsCh := make(chan domain.ScrapeResult)

	tasks := []domain.ScrapeTask{
		{
			Source:   "source",
			PageType: "type_1",
			URL:      "http://example.com",
		},
		{
			Source:   "source",
			PageType: "type_1",
			URL:      "http://example.com/1",
		},
		{
			Source:   "source",
			PageType: "type_1",
			URL:      "http://example.com/2",
		},
	}

	expectedResults := []domain.ScrapeResult{makeResult("0"), makeResult("1")}

	scraper.EXPECT().Scrape(mock.Anything, tasks[0]).Return(expectedResults[0], nil)
	scraper.EXPECT().Scrape(mock.Anything, tasks[1]).Return(domain.ScrapeResult{}, errors.New("Scrape failure"))
	scraper.EXPECT().Scrape(mock.Anything, tasks[2]).Return(expectedResults[1], nil)

	tasksCh := make(chan domain.ScrapeTask, len(tasks))
	for _, task := range tasks {
		tasksCh <- task
	}
	close(tasksCh)

	worker := NewWorker(zap.NewNop(), sourceToScraper, resultsCh)

	go worker.Run(t.Context(), tasksCh)

	var results []domain.ScrapeResult
	for result := range resultsCh {
		results = append(results, result)
	}

	require.ElementsMatch(t, expectedResults, results)
}

func TestContextCancelled(t *testing.T) {
	scraper := NewMockScraper(t)
	sourceToScraper := map[string]Scraper{
		"source": scraper,
	}
	resultsCh := make(chan domain.ScrapeResult)
	tasksCh := make(chan domain.ScrapeTask)

	worker := NewWorker(zap.NewNop(), sourceToScraper, resultsCh)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	var workerExited bool
	go func() {
		worker.Run(ctx, tasksCh)
		workerExited = true
	}()
	time.Sleep(200 * time.Millisecond)

	require.True(t, workerExited)

	_, ok := <-resultsCh
	require.False(t, ok) // channel closed
}

func makeResult(name string) domain.ScrapeResult {
	return domain.ScrapeResult{
		Resources: []domain.Resource{
			{
				Name: name,
			},
		},
	}
}
