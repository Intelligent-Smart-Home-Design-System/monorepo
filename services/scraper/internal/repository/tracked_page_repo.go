package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
)

type TrackedPageRepo struct {
	db *sql.DB
}

func NewTrackedPageRepo(db *sql.DB) *TrackedPageRepo {
	return &TrackedPageRepo{db: db}
}

func (r *TrackedPageRepo) GetTasks(source, pageType string, limit int) ([]domain.ScrapeTask, error) {
	query := `
        SELECT id, source_name, page_type, url, first_seen_at, last_scraped_at
        FROM tracked_pages
        WHERE is_active = true`
	args := make([]any, 0, 3)
	argN := 1

	if source != "" {
		query += fmt.Sprintf(" AND source_name = $%d", argN)
		args = append(args, source)
		argN++
	}
	if pageType != "" {
		query += fmt.Sprintf(" AND page_type = $%d", argN)
		args = append(args, pageType)
		argN++
	}

	query += " ORDER BY last_scraped_at NULLS FIRST"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argN)
		args = append(args, limit)
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []domain.ScrapeTask
	for rows.Next() {
		var t domain.ScrapeTask
		var pageType string
		var lastScraped sql.NullTime
		if err := rows.Scan(&t.ID, &t.Source, &pageType, &t.URL, &t.FirstSeenAt, &lastScraped); err != nil {
			return nil, err
		}
		t.PageType = domain.PageTypeFromString(pageType)
		if lastScraped.Valid {
			ts := lastScraped.Time
			t.LastScrapedAt = &ts
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// GetFailedTasks selects pages deactivated after repeated failures
// (is_active = false, see SetStatus) whose last attempt falls within the
// given window — old, long-dead pages aren't retried forever by default.
func (r *TrackedPageRepo) GetFailedTasks(source, pageType string, since time.Time, limit int) ([]domain.ScrapeTask, error) {
	query := `
        SELECT id, source_name, page_type, url, first_seen_at, last_scraped_at
        FROM tracked_pages
        WHERE is_active = false AND last_scraped_at >= $1`
	args := []any{since}
	argN := 2

	if source != "" {
		query += fmt.Sprintf(" AND source_name = $%d", argN)
		args = append(args, source)
		argN++
	}
	if pageType != "" {
		query += fmt.Sprintf(" AND page_type = $%d", argN)
		args = append(args, pageType)
		argN++
	}

	query += " ORDER BY last_scraped_at ASC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argN)
		args = append(args, limit)
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []domain.ScrapeTask
	for rows.Next() {
		var t domain.ScrapeTask
		var pt string
		var lastScraped sql.NullTime
		if err := rows.Scan(&t.ID, &t.Source, &pt, &t.URL, &t.FirstSeenAt, &lastScraped); err != nil {
			return nil, err
		}
		t.PageType = domain.PageTypeFromString(pt)
		if lastScraped.Valid {
			ts := lastScraped.Time
			t.LastScrapedAt = &ts
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (r *TrackedPageRepo) SetStatus(taskID int, success bool, durationMs int) error {
	now := time.Now()
	if success {
		_, err := r.db.Exec(`
            UPDATE tracked_pages
            SET last_scraped_at = $1,
                last_successful_scrape_at = $1,
                scrape_count = scrape_count + 1,
                consecutive_failures = 0,
                is_active = true
            WHERE id = $2
        `, now, taskID)
		return err
	} else {
		_, err := r.db.Exec(`
            UPDATE tracked_pages
            SET last_scraped_at = $1,
                consecutive_failures = consecutive_failures + 1,
                is_active = CASE WHEN consecutive_failures + 1 >= 5 THEN false ELSE true END
            WHERE id = $2
        `, now, taskID)
		return err
	}
}

func (r *TrackedPageRepo) CreateTask(source, pageType, url string) error {
    _, err := r.db.Exec(`
        INSERT INTO tracked_pages (source_name, page_type, url, is_active)
        VALUES ($1, $2, $3, true)
        ON CONFLICT (url_hash) DO NOTHING
    `, source, pageType, url)
    return err
}

func (r *TrackedPageRepo) DeleteTaskByID(id int) error {
	_, err := r.db.Exec(`DELETE FROM tracked_pages WHERE id = $1`, id)
	return err
}
