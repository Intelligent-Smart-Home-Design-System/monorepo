package pipeline

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/rs/zerolog"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/filters"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/metrics"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/repository"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scraper"
)

// ScrapePhase loads tasks from DB, runs the worker pool, and persists snapshots.
func ScrapePhase(
	ctx context.Context,
	logger zerolog.Logger,
	m *metrics.Collector,
	taskRepo *repository.TrackedPageRepo,
	snapshotRepo *repository.SnapshotRepo,
	sourceToScraper map[string]scraper.Scraper,
	cfg config.Config,
	sources, pageTypes []string,
	discoveryOnly bool,
	sqlPageType string,
	retryFailed bool,
	retrySince time.Time,
) error {
	sourceFilter, pageTypeFilter := scrapeTaskFilters(sources, pageTypes, discoveryOnly, sqlPageType)

	var allTasks []domain.ScrapeTask
	var err error
	if retryFailed {
		allTasks, err = taskRepo.GetFailedTasks(sourceFilter, pageTypeFilter, retrySince, 0)
	} else {
		allTasks, err = taskRepo.GetTasks(sourceFilter, pageTypeFilter, 0)
	}
	if err != nil {
		return fmt.Errorf("get tasks: %w", err)
	}

	var sourceSet map[string]bool
	if len(sources) > 0 {
		sourceSet = make(map[string]bool, len(sources))
		for _, s := range sources {
			sourceSet[s] = true
		}
	}

	var allowedPageTypes []string
	if sqlPageType != "" {
		allowedPageTypes = []string{sqlPageType}
	} else if discoveryOnly {
		allowedPageTypes = []string{domain.PageTypeDiscovery.String()}
	} else if len(pageTypes) > 0 {
		allowedPageTypes = pageTypes
	}

	var tasks []domain.ScrapeTask
	for _, t := range allTasks {
		if sourceSet != nil && !sourceSet[t.Source] {
			continue
		}
		if len(allowedPageTypes) > 0 && !slices.Contains(allowedPageTypes, t.PageType.String()) {
			continue
		}
		tasks = append(tasks, t)
	}

	scrapeJob := config.JobScrape
	if discoveryOnly {
		scrapeJob = config.JobScrapeDiscovery
	}
	beforeFilter := len(tasks)
	for _, t := range tasks {
		m.AddTasksSelected(ctx, t.Source, t.PageType.String(), scrapeJob, metrics.FilterStageBefore, 1)
	}
	tasks = filters.ScrapeTasks(tasks, scrapeJob, cfg.Jobs)
	for _, t := range tasks {
		m.AddTasksSelected(ctx, t.Source, t.PageType.String(), scrapeJob, metrics.FilterStageAfter, 1)
	}
	taskByID := make(map[int]domain.ScrapeTask, len(tasks))
	for _, t := range tasks {
		taskByID[t.ID] = t
	}
	logger.Info().
		Str("job", scrapeJob).
		Str("page_type", pageTypeFilter).
		Strs("sources", uniqueTaskSources(tasks)).
		Int("tasks_before_filter", beforeFilter).
		Int("tasks_matched", len(tasks)).
		Msg("scrape tasks after job filters")

	if len(tasks) == 0 {
		logger.Info().Str("page_type", pageTypeFilter).Msg("no active tasks after filtering for phase")
		return nil
	}

	tasksCh := make(chan domain.ScrapeTask)
	resultsCh := make(chan domain.ScrapeResult)
	worker := scraper.NewWorker(logger, sourceToScraper, resultsCh, cfg.Scraping.MaxConcurrency)

	go func() {
		defer close(tasksCh)
		for _, task := range tasks {
			select {
			case <-ctx.Done():
				return
			case tasksCh <- task:
			}
		}
	}()

	go worker.Run(ctx, tasksCh)

	for result := range resultsCh {
		task := taskByID[result.TrackedPageID]
		pageType := task.PageType.String()
		source := task.Source
		taskLog := logger.With().Str("source", source).Str("page_type", pageType).Logger()

		taskLog.Debug().Int("task_id", result.TrackedPageID).Int("resources", len(result.Resources)).Err(result.Err).Msg("run: received result for task")

		if result.Err != nil {
			taskLog.Error().
				Err(result.Err).
				Int("task_id", result.TrackedPageID).
				Int64("listings_scraped", m.SuccessCount(source, domain.PageTypeListing.String())).
				Int64("categories_scraped", m.SuccessCount(source, domain.PageTypeCategory.String())).
				Msg("scrape error")
			m.AddTaskFinished(ctx, source, pageType, metrics.StatusFailure, 1)
			m.RecordTaskDuration(ctx, source, pageType, result.DurationMs)
			if err := taskRepo.SetStatus(result.TrackedPageID, false, result.DurationMs); err != nil {
				taskLog.Error().Err(err).Msg("update status error")
			}
			continue
		}
		if err := snapshotRepo.SaveResult(result.TrackedPageID, result, result.DurationMs); err != nil {
			taskLog.Error().
				Err(err).
				Int64("listings_scraped", m.SuccessCount(source, domain.PageTypeListing.String())).
				Int64("categories_scraped", m.SuccessCount(source, domain.PageTypeCategory.String())).
				Msg("save snapshot")
			m.AddTaskFinished(ctx, source, pageType, metrics.StatusFailure, 1)
		} else {
			taskLog.Info().Msg("snapshot saved successfully")
			m.AddTaskFinished(ctx, source, pageType, metrics.StatusSuccess, 1)
			if err := taskRepo.SetStatus(result.TrackedPageID, true, result.DurationMs); err != nil {
				taskLog.Error().Err(err).Msg("update status")
			}
		}
		m.RecordTaskDuration(ctx, source, pageType, result.DurationMs)
		taskLog.Debug().Int("task_id", result.TrackedPageID).Msg("run: finished processing task")
	}

	return nil
}

func scrapeTaskFilters(sources, pageTypes []string, discoveryOnly bool, sqlPageType string) (source, pageType string) {
	if len(sources) == 1 {
		source = sources[0]
	}
	switch {
	case sqlPageType != "":
		pageType = sqlPageType
	case discoveryOnly:
		pageType = domain.PageTypeDiscovery.String()
	case len(pageTypes) == 1:
		pageType = pageTypes[0]
	}
	return source, pageType
}

func uniqueTaskSources(tasks []domain.ScrapeTask) []string {
	seen := make(map[string]struct{}, len(tasks))
	var out []string
	for _, t := range tasks {
		if t.Source == "" {
			continue
		}
		if _, ok := seen[t.Source]; ok {
			continue
		}
		seen[t.Source] = struct{}{}
		out = append(out, t.Source)
	}
	slices.Sort(out)
	return out
}

func discoveryPageType(pageTypes []string, discoveryOnly bool) string {
	if discoveryOnly {
		return domain.PageTypeDiscovery.String()
	}
	if len(pageTypes) == 1 {
		return pageTypes[0]
	}
	return ""
}
