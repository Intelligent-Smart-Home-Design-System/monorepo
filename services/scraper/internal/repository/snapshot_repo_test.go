//go:build integration
package repository_test

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tc "github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/repository"
)

func setupTestDB(t *testing.T) (*repository.SnapshotRepo, *sql.DB) {
	t.Helper()
	ctx := context.Background()

	ctr, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("scraper"),
		tcpostgres.WithUsername("scraper"),
		tcpostgres.WithPassword("test"),
		tc.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ctr.Terminate(ctx) })

	host, err := ctr.Host(ctx)
	require.NoError(t, err)
	port, err := ctr.MappedPort(ctx, "5432")
	require.NoError(t, err)

	dsn := fmt.Sprintf("postgres://scraper:test@%s:%s/scraper?sslmode=disable", host, port.Port())

	_, thisFile, _, _ := runtime.Caller(0)
	migrationsDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "db", "catalog", "migrations")

	m, err := migrate.New("file://"+migrationsDir, dsn)
	require.NoError(t, err)
	defer m.Close()
	err = m.Up()
	require.True(t, err == nil || err == migrate.ErrNoChange)

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	repo := repository.NewSnapshotRepo(db, zerolog.Nop())
	return repo, db
}

// seeds a tracked_page and a page_snapshot, returns their IDs
func seedSnapshot(t *testing.T, db *sql.DB, sourceName, pageType string, processed bool) (trackedPageID, snapshotID int) {
	t.Helper()

	url := fmt.Sprintf("https://example.com/%s/%s/%d", sourceName, pageType, time.Now().UnixNano())
	err := db.QueryRow(`
		INSERT INTO tracked_pages (source_name, page_type, url)
		VALUES ($1, $2, $3)
		RETURNING id
	`, sourceName, pageType, url).Scan(&trackedPageID)
	require.NoError(t, err)

	err = db.QueryRow(`
		INSERT INTO page_snapshots (tracked_page, processed, scrape_duration_ms)
		VALUES ($1, $2, $3)
		RETURNING id
	`, trackedPageID, processed, 500).Scan(&snapshotID)
	require.NoError(t, err)

	return trackedPageID, snapshotID
}

func TestSnapshotRepo_SaveResult(t *testing.T) {
	repo, db := setupTestDB(t)

	var taskID int
	err := db.QueryRow(`
        INSERT INTO tracked_pages (source_name, page_type, url, is_active)
        VALUES ($1, $2, $3, $4)
        RETURNING id
    `, "sprut", "listing", "https://test.com", true).Scan(&taskID)
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

	err = repo.SaveResult(taskID, result, 42)
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

func TestGetUnprocessedSnapshots_ReturnsOnlyUnprocessed(t *testing.T) {
	repo, db := setupTestDB(t)

	_, unprocessedID := seedSnapshot(t, db, "amazon_us", "listing", false)
	seedSnapshot(t, db, "amazon_us", "listing", true) // should not appear

	snapshots, err := repo.GetUnprocessedSnapshots(t.Context(), "", "")
	require.NoError(t, err)

	ids := make([]int, len(snapshots))
	for i, s := range snapshots {
		ids[i] = s.ID
	}
	assert.Contains(t, ids, unprocessedID)
	for _, s := range snapshots {
		assert.False(t, s.ID == 0)
	}
}

func TestGetUnprocessedSnapshots_FilterByPageType(t *testing.T) {
	repo, db := setupTestDB(t)

	_, listingID := seedSnapshot(t, db, "amazon_us", "listing", false)
	_, compatID := seedSnapshot(t, db, "amazon_us", "compatibility", false)

	snapshots, err := repo.GetUnprocessedSnapshots(t.Context(), "listing", "")
	require.NoError(t, err)

	ids := make([]int, len(snapshots))
	for i, s := range snapshots {
		ids[i] = s.ID
	}
	assert.Contains(t, ids, listingID)
	assert.NotContains(t, ids, compatID)

	for _, s := range snapshots {
		assert.Equal(t, "listing", s.PageType)
	}
}

func TestGetUnprocessedSnapshots_FilterBySource(t *testing.T) {
	repo, db := setupTestDB(t)

	_, amazonID := seedSnapshot(t, db, "amazon_us", "listing", false)
	_, wbID := seedSnapshot(t, db, "wildberries", "listing", false)

	snapshots, err := repo.GetUnprocessedSnapshots(t.Context(), "", "amazon_us")
	require.NoError(t, err)

	ids := make([]int, len(snapshots))
	for i, s := range snapshots {
		ids[i] = s.ID
	}
	assert.Contains(t, ids, amazonID)
	assert.NotContains(t, ids, wbID)
}

func TestSaveListingParseResult(t *testing.T) {
	repo, db := setupTestDB(t)

	_, snapshotID := seedSnapshot(t, db, "amazon_us", "listing", false)

	price := 4999
	currency := "RUB"
	modelNum := "WLD-01"
	category := "water_leak_detector"
	qty := 2
	qtyRaw := "2-Pack"

	result := domain.ListingParseResult{
		PageSnapshotID: snapshotID,
		InStock:        true,
		Text:           "Some extracted description text",
		Name:           "Smart Water Leak Detector",
		Brand:          "Sprut",
		ImageURL:       "https://example.com/img.jpg",
		Price:          &price,
		Currency:       &currency,
		ModelNumber:    &modelNum,
		Category:       &category,
		Quantity:       &qty,
		QuantityRaw:    &qtyRaw,
		Rating:         4.5,
		ReviewCount:    128,
		ContentHash:    "abc123def456",
		ExtractorVer:   "1.0.0",
		ParsedAt:       time.Now().UTC(),
	}

	err := repo.SaveListingParseResult(&result)
	require.NoError(t, err)

	// verify parsed record was inserted
	var (
		dbSnapshotID int
		dbInStock    bool
		dbName       string
		dbRating     float64
		dbPrice      sql.NullInt64
		dbProcessed  bool
	)

	err = db.QueryRow(`
		SELECT page_snapshot_id, extracted_in_stock, extracted_name, extracted_rating, extracted_price
		FROM parsed_listing_snapshots
		WHERE page_snapshot_id = $1
	`, snapshotID).Scan(&dbSnapshotID, &dbInStock, &dbName, &dbRating, &dbPrice)
	require.NoError(t, err)

	assert.Equal(t, snapshotID, dbSnapshotID)
	assert.True(t, dbInStock)
	assert.Equal(t, "Smart Water Leak Detector", dbName)
	assert.InDelta(t, 4.5, dbRating, 0.01)
	assert.True(t, dbPrice.Valid)
	assert.Equal(t, int64(4999), dbPrice.Int64)

	// verify the snapshot was marked as processed
	err = db.QueryRow(`SELECT processed FROM page_snapshots WHERE id = $1`, snapshotID).Scan(&dbProcessed)
	require.NoError(t, err)
	assert.True(t, dbProcessed)
}

func TestSaveListingParseResult_MarksSnapshotProcessed(t *testing.T) {
	repo, db := setupTestDB(t)

	_, snapshotID := seedSnapshot(t, db, "wildberries", "listing", false)

	result := domain.ListingParseResult{
		PageSnapshotID: snapshotID,
		InStock:        false,
		Text:           "out of stock item",
		Name:           "Smart Plug",
		Brand:          "Yandex",
		Rating:         3.8,
		ReviewCount:    42,
		ContentHash:    "xyz789",
		ExtractorVer:   "1.0.0",
		ParsedAt:       time.Now().UTC(),
	}

	err := repo.SaveListingParseResult(&result)
	require.NoError(t, err)

	// processed snapshot should no longer appear in unprocessed results
	snapshots, err := repo.GetUnprocessedSnapshots(t.Context(), "", "wildberries")
	require.NoError(t, err)

	for _, s := range snapshots {
		assert.NotEqual(t, snapshotID, s.ID, "processed snapshot should not be returned")
	}
}

func TestSaveListingParseResult_NullableFieldsOmitted(t *testing.T) {
	repo, db := setupTestDB(t)

	_, snapshotID := seedSnapshot(t, db, "sprut_ai", "listing", false)

	// only required fields, all pointers nil
	result := domain.ListingParseResult{
		PageSnapshotID: snapshotID,
		InStock:        true,
		Text:           "minimal listing",
		Name:           "Sensor X",
		Brand:          "Acme",
		Rating:         4.0,
		ReviewCount:    10,
		ContentHash:    "minimal123",
		ExtractorVer:   "1.0.0",
		ParsedAt:       time.Now().UTC(),
	}

	err := repo.SaveListingParseResult(&result)
	require.NoError(t, err)

	var price sql.NullInt64
	var currency sql.NullString
	err = db.QueryRow(`
		SELECT extracted_price, extracted_currency
		FROM parsed_listing_snapshots
		WHERE page_snapshot_id = $1
	`, snapshotID).Scan(&price, &currency)
	require.NoError(t, err)

	assert.False(t, price.Valid)
	assert.False(t, currency.Valid)
}
