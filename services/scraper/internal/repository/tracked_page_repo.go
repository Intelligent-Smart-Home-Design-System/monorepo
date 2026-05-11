package repository

import (
	"database/sql"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
)

type TrackedPageRepo struct {
	db *sql.DB
}

func NewTrackedPageRepo(db *sql.DB) *TrackedPageRepo {
	return &TrackedPageRepo{db: db}
}

func (r *TrackedPageRepo) GetTasks() ([]domain.ScrapeTask, error) {
	rows, err := r.db.Query(`
        SELECT id, source_name, page_type, url
        FROM tracked_pages
        WHERE is_active = true
        ORDER BY last_scraped_at NULLS FIRST
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []domain.ScrapeTask
	for rows.Next() {
		var t domain.ScrapeTask
		var pageType string
		if err := rows.Scan(&t.ID, &t.Source, &pageType, &t.URL); err != nil {
			return nil, err
		}
		t.PageType = domain.PageTypeFromString(pageType)
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
                consecutive_failures = 0
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
