//go:build integration

package repository_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/catalog-builder/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/catalog-builder/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/catalog-builder/internal/repository"
)

func setupTestRepo(t *testing.T) (*repository.PostgresRepository, *sql.DB) {
	t.Helper()
	ctx := context.Background()

	ctr, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("catalog"),
		tcpostgres.WithUsername("catalog-builder"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		),
	)

	require.NoError(t, err)
	t.Cleanup(func() { _ = ctr.Terminate(ctx) })

	host, err := ctr.Host(ctx)
	require.NoError(t, err)
	port, err := ctr.MappedPort(ctx, "5432")
	require.NoError(t, err)

	dsn := fmt.Sprintf("postgres://catalog-builder:test@%s:%s/catalog?sslmode=disable", host, port.Port())

	_, thisFile, _, _ := runtime.Caller(0)
	migrationsDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "db", "catalog", "migrations")

	m, err := migrate.New("file://"+migrationsDir, dsn)
	require.NoError(t, err)
	defer m.Close()
	err = m.Up()
	require.True(t, err == nil || err == migrate.ErrNoChange)

	repo, err := repository.NewPostgresRepository(config.DatabaseConfig{
		Host:     host,
		Port:     int(port.Num()),
		User:     "catalog-builder",
		Password: "test",
		DBName:   "catalog",
		SSLMode:  "disable",
	}, zerolog.Nop())
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	return repo, db
}

func seedListing(t *testing.T, db *sql.DB, url, brand string, attrs map[string]any) int {
	t.Helper()
	ctx := context.Background()

	var trackedPageID int
	require.NoError(t, db.QueryRowContext(ctx,
		`INSERT INTO tracked_pages (source_name, page_type, url) VALUES ('amazon_us', 'listing', $1) RETURNING id`, url,
	).Scan(&trackedPageID))

	var snapID int
	require.NoError(t, db.QueryRowContext(ctx,
		`INSERT INTO page_snapshots (tracked_page) VALUES ($1) RETURNING id`, trackedPageID,
	).Scan(&snapID))

	var parsedID int
	require.NoError(t, db.QueryRowContext(ctx, `
		INSERT INTO parsed_listing_snapshots
		    (page_snapshot_id, extracted_in_stock, extracted_text, extracted_name,
		     extracted_brand, extracted_rating, extracted_review_count)
		VALUES ($1, true, 'desc', 'Product Name', $2, 4.5, 100) RETURNING id`,
		snapID, brand,
	).Scan(&parsedID))

	attrsJSON, err := json.Marshal(attrs)
	require.NoError(t, err)

	var listingID int
	require.NoError(t, db.QueryRowContext(ctx, `
		INSERT INTO llm_extracted_listings
		    (parsed_listing_snapshot_id, brand, model, category, category_confidence,
		     device_attributes, llm_model, taxonomy_version)
		VALUES ($1, $2, 'YNDX-00558', 'smart_lamp', 0.95, $3, 'gpt-4', 'v1')
		RETURNING id`,
		parsedID, brand, attrsJSON,
	).Scan(&listingID))

	return listingID
}

func TestGetLatestExtractedListings(t *testing.T) {
	repo, db := setupTestRepo(t)
	ctx := context.Background()

	attrs := map[string]any{
		"color":     "white",
		"wattage":   9.5,
		"dimmable":  true,
		"ecosystem": []any{"apple", "google"},
	}
	seedListing(t, db, "https://example.com/lamp1", "yandex", attrs)

	listings, err := repo.GetLatestExtractedListings(ctx)
	require.NoError(t, err)
	require.Len(t, listings, 1)

	l := listings[0]
	assert.Equal(t, "yandex", l.Brand)
	assert.Equal(t, "gpt-4", l.LLM)
	assert.Equal(t, "v1", l.TaxonomyVersion)
	require.NotNil(t, l.Model)
	assert.Equal(t, "YNDX-00558", *l.Model)

	assert.Equal(t, "white", l.DeviceAttributes["color"])
	assert.Equal(t, float64(9.5), l.DeviceAttributes["wattage"])
	assert.Equal(t, true, l.DeviceAttributes["dimmable"])

	temps, ok := l.DeviceAttributes["ecosystem"].([]any)
	require.True(t, ok, "ecosystem should unmarshal as []any")
	assert.Equal(t, []any{"apple", "google"}, temps)
}

func TestGetLatestExtractedListings_LatestPerTrackedPage(t *testing.T) {
	repo, db := setupTestRepo(t)
	ctx := context.Background()

	var trackedPageID int
	require.NoError(t, db.QueryRowContext(ctx,
		`INSERT INTO tracked_pages (source_name, page_type, url) VALUES ('amazon_us', 'listing', 'https://example.com/page') RETURNING id`,
	).Scan(&trackedPageID))

	insertSnapshot := func(brand string) {
		var snapID, parsedID int
		require.NoError(t, db.QueryRowContext(ctx,
			`INSERT INTO page_snapshots (tracked_page) VALUES ($1) RETURNING id`, trackedPageID,
		).Scan(&snapID))
		require.NoError(t, db.QueryRowContext(ctx, `
			INSERT INTO parsed_listing_snapshots
			    (page_snapshot_id, extracted_in_stock, extracted_text, extracted_name, extracted_brand, extracted_rating, extracted_review_count)
			VALUES ($1, true, 'text', 'name', $2, 4.5, 0) RETURNING id`,
			snapID, brand,
		).Scan(&parsedID))
		_, err := db.ExecContext(ctx, `
			INSERT INTO llm_extracted_listings
			    (parsed_listing_snapshot_id, brand, category, category_confidence, device_attributes, llm_model, taxonomy_version)
			VALUES ($1, $2, 'smart_lamp', 0.9, '{}', 'gpt-4', 'v1')`,
			parsedID, brand)
		require.NoError(t, err)
	}

	insertSnapshot("old-brand")
	insertSnapshot("new-brand") // latest snapshot

	listings, err := repo.GetLatestExtractedListings(ctx)
	require.NoError(t, err)
	require.Len(t, listings, 1)
	assert.Equal(t, "new-brand", listings[0].Brand)
}

func TestGetLatestDirectCompatibility(t *testing.T) {
	repo, db := setupTestRepo(t)
	ctx := context.Background()

	// old snapshot (should be ignored
	var oldSnapID int
	require.NoError(t, db.QueryRowContext(ctx,
		`INSERT INTO parsed_direct_compatibility_snapshot DEFAULT VALUES RETURNING id`,
	).Scan(&oldSnapID))
	_, err := db.ExecContext(ctx,
		`INSERT INTO parsed_direct_compatibility_record (snapshot_id, ecosystem, brand, model, protocol) VALUES ($1, 'stale', 'brand', 'model', 'zwave')`,
		oldSnapID)
	require.NoError(t, err)

	// latest snapshot
	var snapID int
	require.NoError(t, db.QueryRowContext(ctx,
		`INSERT INTO parsed_direct_compatibility_snapshot DEFAULT VALUES RETURNING id`,
	).Scan(&snapID))
	_, err = db.ExecContext(ctx, `
		INSERT INTO parsed_direct_compatibility_record (snapshot_id, ecosystem, brand, model, protocol)
		VALUES ($1, 'apple', 'yandex', 'YNDX-00558', 'matter-over-wifi'),
		       ($1, 'google', 'yandex', 'YNDX-00558', 'matter-over-thread')`,
		snapID)
	require.NoError(t, err)

	records, err := repo.GetLatestDirectCompatibility(ctx)
	require.NoError(t, err)
	require.Len(t, records, 2)

	var ecosystems []string
	for _, r := range records {
		ecosystems = append(ecosystems, r.Ecosystem)
	}
	assert.ElementsMatch(t, []string{"apple", "google"}, ecosystems)
}

func TestWriteCatalog(t *testing.T) {
	repo, db := setupTestRepo(t)
	ctx := context.Background()

	listingID := seedListing(t, db, "https://example.com/lamp2", "yandex", map[string]any{"dimmable": true})

	model := "YNDX-00558"
	catalog := &domain.Catalog{
		Devices: []*domain.Device{
			{
				Brand:            "yandex",
				Model:            &model,
				Category:         "smart_lamp",
				DeviceAttributes: map[string]any{"dimmable": true, "wattage": float64(9)},
				TaxonomyVersion:  "v1",
				Listings:         []*domain.ExtractedListing{{Id: listingID}},
				DirectCompatibility: []*domain.DirectCompatibility{
					{Ecosystem: "apple", Protocol: "matter-over-wifi"},
				},
				BridgeCompatibility: []*domain.BridgeCompatibility{
					{SourceEcosystem: "aqara", TargetEcosystem: "apple", Protocol: "matter"},
				},
			},
		},
	}

	require.NoError(t, repo.WriteCatalog(ctx, catalog))

	var count int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM devices`).Scan(&count))
	assert.Equal(t, 1, count)

	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM listing_device_links`).Scan(&count))
	assert.Equal(t, 1, count)

	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM direct_compatibility`).Scan(&count))
	assert.Equal(t, 1, count)

	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM bridge_ecosystem_compatibility`).Scan(&count))
	assert.Equal(t, 1, count)

	// check re running
	require.NoError(t, repo.WriteCatalog(ctx, catalog))
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM devices`).Scan(&count))
	assert.Equal(t, 1, count)
}
