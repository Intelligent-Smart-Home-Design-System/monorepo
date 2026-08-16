package filters

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
)

func JobSourceFilter(cfg config.JobsConfig, job, source string) config.SourceJobFilter {
	filters := cfg.ForJob(job)
	if filters == nil {
		return config.SourceJobFilter{}
	}
	return filters[source]
}

func ScrapeTasks(tasks []domain.ScrapeTask, job string, cfg config.JobsConfig) []domain.ScrapeTask {
	var out []domain.ScrapeTask
	perSourceCount := make(map[string]int)

	for _, t := range tasks {
		f := JobSourceFilter(cfg, job, t.Source)
		if !f.MatchesTask(t.ID, t.URL, t.FirstSeenAt, t.LastScrapedAt) {
			continue
		}
		if f.Limit > 0 && perSourceCount[t.Source] >= f.Limit {
			continue
		}
		out = append(out, t)
		perSourceCount[t.Source]++
	}
	return out
}

func Snapshots(snapshots []*domain.PageSnapshot, job string, cfg config.JobsConfig) []*domain.PageSnapshot {
	var out []*domain.PageSnapshot
	perSourceCount := make(map[string]int)

	for _, s := range snapshots {
		f := JobSourceFilter(cfg, job, s.SourceName)
		if !f.MatchesSnapshot(s.ID, s.TrackedPageID, s.PageURL, s.ScrapedAt) {
			continue
		}
		if f.Limit > 0 && perSourceCount[s.SourceName] >= f.Limit {
			continue
		}
		out = append(out, s)
		perSourceCount[s.SourceName]++
	}
	return out
}
