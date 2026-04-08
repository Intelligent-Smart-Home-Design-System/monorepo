package repository

import (
    "archive/tar"
    "bytes"
    "compress/gzip"
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
    buf := new(bytes.Buffer)

    gzipWriter := gzip.NewWriter(buf)
    defer gzipWriter.Close()

    tarWriter := tar.NewWriter(gzipWriter)
    defer tarWriter.Close()

    for _, res := range result.Resources {
        header := &tar.Header{
            Name:       res.Name,
            Size:       int64(len(res.ResponseBody)),
            Mode:       0600,
            ModTime:    time.Now(),
        }

        if err := tarWriter.WriteHeader(header); err != nil {
            return err
        }

        if _, err := tarWriter.Write(res.ResponseBody); err != nil {
            return err
        }
    }

    _, err := r.db.Exec(`
        INSERT INTO page_snapshots (tracked_page, scraped_at, warc_bundle_archive, scrape_duration_ms)
        VALUES ($1, $2, $3, $4)
    `, trackedPageID, time.Now(), buf.Bytes(), durationMs)
    return err
}
