//go:build integration

package repository

import (
	"testing"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnapshotRepo_SaveResult(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	snapshotRepo := NewSnapshotRepo(db)

	var taskID int
	err := db.QueryRow(`
        INSERT INTO tracked_pages (source_name, page_type, url, is_active)
        VALUES ($1, $2, $3, $4)
        RETURNING id
    `, "sprut", "article", "https://test.com", true).Scan(&taskID)
	require.NoError(t, err)

	resources := []domain.Resource{
		{
			Name:         "html",
			URL:          "https://test.com",
			ResponseBody: []byte("<html><body>test</body></html>"),
		},
	}
	result := domain.ScrapeResult{
		TrackedPageID: taskID,
		Resources:     resources,
	}

	err = snapshotRepo.SaveResult(taskID, result, 42)
	require.NoError(t, err)

	var snapshotID int
	var archived []byte
	err = db.QueryRow(`
        SELECT id, warc_bundle_archive
        FROM page_snapshots
        WHERE tracked_page = $1
    `, taskID).Scan(&snapshotID, &archived)
	require.NoError(t, err)
	assert.NotEmpty(t, archived)
}
