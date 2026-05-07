package repository

import (
    "archive/tar"
    "bytes"
    "compress/gzip"
    "database/sql"
    "fmt"
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
    fmt.Printf("[DEBUG] SaveResult: called for task %d, resources count = %d\n", trackedPageID, len(result.Resources))

    buf := new(bytes.Buffer)
    gzipWriter := gzip.NewWriter(buf)
    tarWriter := tar.NewWriter(gzipWriter)

    for i, res := range result.Resources {
        fmt.Printf("[DEBUG] SaveResult: writing resource %d, name=%s, body len=%d\n", i, res.Name, len(res.ResponseBody))
        header := &tar.Header{
            Name:   res.Name,
            Size:   int64(len(res.ResponseBody)),
            Mode:   0600,
            ModTime: time.Now(),
        }
        if err := tarWriter.WriteHeader(header); err != nil {
            return fmt.Errorf("write header for %s: %w", res.Name, err)
        }
        if _, err := tarWriter.Write(res.ResponseBody); err != nil {
            return fmt.Errorf("write body for %s: %w", res.Name, err)
        }
    }

    if err := tarWriter.Close(); err != nil {
        return fmt.Errorf("close tar: %w", err)
    }
    if err := gzipWriter.Close(); err != nil {
        return fmt.Errorf("close gzip: %w", err)
    }

    fmt.Printf("[DEBUG] SaveResult: archive size = %d bytes\n", buf.Len())

    _, err := r.db.Exec(`
        INSERT INTO page_snapshots (tracked_page, scraped_at, warc_bundle_archive, scrape_duration_ms)
        VALUES ($1, $2, $3, $4)
    `, trackedPageID, time.Now(), buf.Bytes(), durationMs)
    if err != nil {
        return fmt.Errorf("db insert: %w", err)
    }
    fmt.Printf("[DEBUG] SaveResult: successfully inserted\n")
    return nil
}