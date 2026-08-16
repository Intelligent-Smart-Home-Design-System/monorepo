package config

import (
	"strings"
	"time"
)

// JobsConfig holds per-job filters keyed by source name (e.g. jobs.scrape_discovery.dns).
// Multiple sources: add [jobs.scrape_discovery.wildberries], [jobs.scrape.dns.sprut], etc.
type JobsConfig struct {
	ScrapeDiscovery map[string]SourceJobFilter `mapstructure:"scrape_discovery"`
	ParseDiscovery  map[string]SourceJobFilter `mapstructure:"parse_discovery"`
	Scrape          map[string]SourceJobFilter `mapstructure:"scrape"`
	Parse           map[string]SourceJobFilter `mapstructure:"parse"`
}

// SourceJobFilter limits which tracked_pages / snapshots a job processes for one source.
type SourceJobFilter struct {
	Limit              int       `mapstructure:"limit"`
	TrackedPageIDs     []int     `mapstructure:"tracked_page_ids"`  // scrape + parse: tracked_pages.id
	PageSnapshotIDs    []int     `mapstructure:"page_snapshot_ids"` // parse only: page_snapshots.id
	URLContains        []string  `mapstructure:"url_contains"`
	DiscoveryBootstrap []string  `mapstructure:"discovery_bootstrap"` // scrape_discovery only: "seed", "db"
	// scrape: tracked_pages.last_scraped_at; parse: page_snapshots.scraped_at
	ScrapedAfter  time.Time `mapstructure:"scraped_after"`
	ScrapedBefore time.Time `mapstructure:"scraped_before"`
	// scrape only: tracked_pages.first_seen_at — когда задача появилась в пайплайне (BFS, parse, bootstrap)
	CreatedAfter  time.Time `mapstructure:"created_after"`
	CreatedBefore time.Time `mapstructure:"created_before"`
}

// MatchesTask filters scrape tasks by id, URL, first_seen_at and last_scraped_at.
func (f SourceJobFilter) MatchesTask(trackedPageID int, pageURL string, firstSeenAt time.Time, lastScrapedAt *time.Time) bool {
	if !f.matchesTrackedPageAndURL(trackedPageID, pageURL) {
		return false
	}
	if !f.matchesCreatedTime(firstSeenAt) {
		return false
	}
	unscraped := lastScrapedAt == nil
	at := time.Time{}
	if lastScrapedAt != nil {
		at = *lastScrapedAt
	}
	return f.matchesScrapedTime(at, unscraped)
}

// Matches filters scrape tasks without timestamps (legacy; prefer MatchesTask).
func (f SourceJobFilter) Matches(trackedPageID int, pageURL string) bool {
	return f.MatchesTask(trackedPageID, pageURL, time.Time{}, nil)
}

// MatchesSnapshot filters page_snapshots by id, tracked page, URL and scraped_at.
func (f SourceJobFilter) MatchesSnapshot(snapshotID, trackedPageID int, pageURL string, scrapedAt time.Time) bool {
	if len(f.PageSnapshotIDs) > 0 {
		found := false
		for _, id := range f.PageSnapshotIDs {
			if id == snapshotID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if !f.matchesTrackedPageAndURL(trackedPageID, pageURL) {
		return false
	}
	return f.matchesScrapedTime(scrapedAt, false)
}

func (f SourceJobFilter) matchesScrapedTime(at time.Time, unscraped bool) bool {
	if f.ScrapedAfter.IsZero() && f.ScrapedBefore.IsZero() {
		return true
	}
	if unscraped {
		// Ни разу не скрапили: проходят только без нижней границы (только scraped_before или без фильтра).
		return f.ScrapedAfter.IsZero()
	}
	if !f.ScrapedAfter.IsZero() && at.Before(f.ScrapedAfter) {
		return false
	}
	if !f.ScrapedBefore.IsZero() && at.After(f.ScrapedBefore) {
		return false
	}
	return true
}

func (f SourceJobFilter) matchesCreatedTime(firstSeenAt time.Time) bool {
	if f.CreatedAfter.IsZero() && f.CreatedBefore.IsZero() {
		return true
	}
	if !f.CreatedAfter.IsZero() && firstSeenAt.Before(f.CreatedAfter) {
		return false
	}
	if !f.CreatedBefore.IsZero() && firstSeenAt.After(f.CreatedBefore) {
		return false
	}
	return true
}

func (f SourceJobFilter) matchesTrackedPageAndURL(trackedPageID int, pageURL string) bool {
	if len(f.TrackedPageIDs) > 0 {
		found := false
		for _, id := range f.TrackedPageIDs {
			if id == trackedPageID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(f.URLContains) > 0 {
		lower := strings.ToLower(pageURL)
		found := false
		for _, sub := range f.URLContains {
			sub = strings.ToLower(strings.TrimSpace(sub))
			if sub == "" {
				continue
			}
			if strings.Contains(lower, sub) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// BootstrapMode parses discovery_bootstrap. Empty means both seed and db.
func (f SourceJobFilter) BootstrapMode() (seed, db bool) {
	if len(f.DiscoveryBootstrap) == 0 {
		return true, true
	}
	for _, mode := range f.DiscoveryBootstrap {
		switch strings.ToLower(strings.TrimSpace(mode)) {
		case "seed":
			seed = true
		case "db":
			db = true
		}
	}
	return seed, db
}

func (j JobsConfig) ForJob(job string) map[string]SourceJobFilter {
	switch job {
	case JobScrapeDiscovery:
		return j.ScrapeDiscovery
	case JobParseDiscovery:
		return j.ParseDiscovery
	case JobScrape:
		return j.Scrape
	case JobParse:
		return j.Parse
	default:
		return nil
	}
}

const (
	JobScrapeDiscovery = "scrape_discovery"
	JobParseDiscovery  = "parse_discovery"
	JobScrape          = "scrape"
	JobParse           = "parse"
)
