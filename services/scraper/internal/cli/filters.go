package cli

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/filters"
)

func filterScrapeTasks(tasks []domain.ScrapeTask, job string, cfg config.JobsConfig) []domain.ScrapeTask {
	return filters.ScrapeTasks(tasks, job, cfg)
}

func filterSnapshots(snapshots []*domain.PageSnapshot, job string, cfg config.JobsConfig) []*domain.PageSnapshot {
	return filters.Snapshots(snapshots, job, cfg)
}
