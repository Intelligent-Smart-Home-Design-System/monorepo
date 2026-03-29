package repository

import (
    "database/sql"
    "time"

    "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
)

type SnapshotRepo struct {
    db *sql.DB
}

func NewSnapshotRepo(db *sql.DB) *SnapshotRepo {
    return &SnapshotRepo{db: db}
}

func (r *SnapshotRepo) SaveResult(trackedPageID int, result domain.ScrapeResult, durationMs int) error {
    var body []byte
    for _, res := range result.Resources {
        if res.Name == "html" {
            body = res.ResponseBody
            break
        }
    }
    if len(body) == 0 && len(result.Resources) > 0 {
        body = result.Resources[0].ResponseBody
    }

    _, err := r.db.Exec(`
        INSERT INTO page_snapshots (tracked_page, scraped_at, warc_bundle_archive, scrape_duration_ms)
        VALUES ($1, $2, $3, $4)
    `, trackedPageID, time.Now(), body, durationMs)
    return err
}