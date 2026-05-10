package repository

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
)

type SnapshotRepo struct {
	db  *sql.DB
	log zerolog.Logger
}

func NewSnapshotRepo(db *sql.DB, log zerolog.Logger) *SnapshotRepo {
	return &SnapshotRepo{db: db, log: log}
}

func (r *SnapshotRepo) SaveResult(trackedPageID int, result domain.ScrapeResult, durationMs int) error {
	fmt.Printf("[DEBUG] SaveResult: called for task %d, resources count = %d\n", trackedPageID, len(result.Resources))

	buf := new(bytes.Buffer)
	gzipWriter := gzip.NewWriter(buf)
	tarWriter := tar.NewWriter(gzipWriter)

	for i, res := range result.Resources {
		fmt.Printf("[DEBUG] SaveResult: writing resource %d, name=%s, body len=%d\n", i, res.Name, len(res.ResponseBody))
		header := &tar.Header{
			Name:    res.Name,
			Size:    int64(len(res.ResponseBody)),
			Mode:    0600,
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

func (r *SnapshotRepo) GetUnprocessedSnapshots(ctx context.Context, pageType string, sourceName string) ([]*domain.PageSnapshot, error) {
	query := `
		SELECT
			ps.id,
			ps.tracked_page,
			ps.scraped_at,
			ps.warc_bundle_archive,
			tp.page_type,
			tp.source_name
		FROM page_snapshots ps
		JOIN tracked_pages tp ON tp.id = ps.tracked_page
		WHERE ps.processed = FALSE
		  AND tp.is_active = TRUE
		  AND ($1 = '' OR tp.page_type = $1)
		  AND ($2 = '' OR tp.source_name = $2)
		ORDER BY ps.scraped_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, pageType, sourceName)
	if err != nil {
		return nil, fmt.Errorf("query unprocessed snapshots: %w", err)
	}
	defer rows.Close()

	var snapshots []*domain.PageSnapshot
	for rows.Next() {
		var s domain.PageSnapshot

		if err := rows.Scan(
			&s.ID,
			&s.TrackedPageID,
			&s.ScrapedAt,
			&s.WARCBundle,
			&s.PageType,
			&s.SourceName,
		); err != nil {
			return nil, fmt.Errorf("scan snapshot row: %w", err)
		}
		snapshots = append(snapshots, &s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate snapshot rows: %w", err)
	}

	r.log.Debug().
		Int("count", len(snapshots)).
		Str("page_type", pageType).
		Str("source_name", sourceName).
		Msg("fetched unprocessed snapshots")

	return snapshots, nil
}

func (r *SnapshotRepo) SaveListingParseResult(result *domain.ListingParseResult) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO parsed_listing_snapshots (
			page_snapshot_id,
			parsed_at,
			extracted_in_stock,
			extracted_text,
			extracted_name,
			extracted_brand,
			extracted_image_url,
			extracted_price,
			extracted_currency,
			extracted_model_number,
			extracted_category,
			extracted_quantity,
			extracted_quantity_raw,
			extracted_rating,
			extracted_review_count,
			content_hash,
			extractor_version
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
	`,
		result.PageSnapshotID,
		result.ParsedAt,
		result.InStock,
		result.Text,
		result.Name,
		result.Brand,
		result.ImageURL,
		result.Price,
		result.Currency,
		result.ModelNumber,
		result.Category,
		result.Quantity,
		result.QuantityRaw,
		result.Rating,
		result.ReviewCount,
		result.ContentHash,
		result.ExtractorVer,
	)
	if err != nil {
		return fmt.Errorf("insert parsed listing snapshot: %w", err)
	}

	_, err = tx.Exec(
		`UPDATE page_snapshots SET processed = TRUE WHERE id = $1`,
		result.PageSnapshotID,
	)
	if err != nil {
		return fmt.Errorf("mark snapshot as processed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	r.log.Debug().
		Int("page_snapshot_id", result.PageSnapshotID).
		Str("content_hash", result.ContentHash).
		Msg("saved listing parse result")

	return nil
}

func (r *SnapshotRepo) SaveDirectCompatibilityRecord(rec *domain.DirectCompatibilityRecord) error {
    _, err := r.db.Exec(`
        INSERT INTO direct_compatibility (brand, model, ecosystem, tracked_page_id, discovered_at)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (brand, model, ecosystem) DO UPDATE SET last_confirmed_at = NOW()
    `, rec.Brand, rec.Model, "yandex", rec.PageSnapshotID, time.Now())
    return err
}
