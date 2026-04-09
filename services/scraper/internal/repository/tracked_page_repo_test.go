//go:build integration

package repository

import (
	"testing"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrackedPageRepo_GetTasks(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := NewTrackedPageRepo(db)

	_, err := db.Exec(`
        INSERT INTO tracked_pages (source_name, page_type, url, is_active)
        VALUES ($1, $2, $3, $4)
    `, "sprut", "listing", "https://test.com", true)
	require.NoError(t, err)

	tasks, err := repo.GetTasks()
	require.NoError(t, err)
	require.Len(t, tasks, 1)

	assert.Equal(t, "sprut", tasks[0].Source)
	assert.Equal(t, domain.PageTypeListing, tasks[0].PageType)
	assert.Equal(t, "https://test.com", tasks[0].URL)
}

func TestTrackedPageRepo_SetStatus(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := NewTrackedPageRepo(db)

	var taskID int
	err := db.QueryRow(`
        INSERT INTO tracked_pages (source_name, page_type, url, is_active)
        VALUES ($1, $2, $3, $4)
        RETURNING id
    `, "sprut", "listing", "https://test.com", true).Scan(&taskID)
	require.NoError(t, err)

	err = repo.SetStatus(taskID, true, 100)
	require.NoError(t, err)

	var lastScrapedAt time.Time
	var scrapeCount int
	var consecutiveFailures int
	err = db.QueryRow(`
        SELECT last_scraped_at, scrape_count, consecutive_failures
        FROM tracked_pages WHERE id = $1
    `, taskID).Scan(&lastScrapedAt, &scrapeCount, &consecutiveFailures)
	require.NoError(t, err)

	assert.WithinDuration(t, time.Now(), lastScrapedAt, 2*time.Second)
	assert.Equal(t, 1, scrapeCount)
	assert.Equal(t, 0, consecutiveFailures)

	err = repo.SetStatus(taskID, false, 100)
	require.NoError(t, err)

	err = db.QueryRow(`
        SELECT consecutive_failures
        FROM tracked_pages WHERE id = $1
    `, taskID).Scan(&consecutiveFailures)
	require.NoError(t, err)
	assert.Equal(t, 1, consecutiveFailures)
}
