package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/rs/zerolog"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/catalog-builder/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/catalog-builder/internal/domain"
)

type PostgresRepository struct {
	db  *sql.DB
	log zerolog.Logger
}

func NewPostgresRepository(cfg config.DatabaseConfig, log zerolog.Logger) (*PostgresRepository, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return &PostgresRepository{db: db, log: log}, nil
}

func (r *PostgresRepository) Close() error {
	return r.db.Close()
}

// GetLatestExtractedListings returns the latest llm_extracted_listing per tracked_page
func (r *PostgresRepository) GetLatestExtractedListings(ctx context.Context) ([]*domain.ExtractedListing, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT ON (tp.id)
			l.id, l.brand, l.model, l.category, l.category_confidence,
			l.device_attributes, l.llm_model, l.taxonomy_version
		FROM tracked_pages tp
		JOIN page_snapshots ps ON ps.tracked_page = tp.id
		JOIN parsed_listing_snapshots pls ON pls.page_snapshot_id = ps.id
		JOIN llm_extracted_listings l ON l.parsed_listing_snapshot_id = pls.id
		ORDER BY tp.id, l.id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query listings: %w", err)
	}
	defer rows.Close()

	var listings []*domain.ExtractedListing
	for rows.Next() {
		l := &domain.ExtractedListing{}
		var attrJSON []byte
		if err := rows.Scan(&l.Id, &l.Brand, &l.Model, &l.Category, &l.CategoryConfidence,
			&attrJSON, &l.LLM, &l.TaxonomyVersion); err != nil {
			return nil, fmt.Errorf("scan listing: %w", err)
		}
		if err := json.Unmarshal(attrJSON, &l.DeviceAttributes); err != nil {
			return nil, fmt.Errorf("unmarshal device_attributes id=%d: %w", l.Id, err)
		}
		listings = append(listings, l)
	}
	return listings, rows.Err()
}

// GetLatestDirectCompatibility returns all records from the latest direct compat snapshot.
func (r *PostgresRepository) GetLatestDirectCompatibility(ctx context.Context) ([]*domain.ScrapedDirectCompatibility, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT r.ecosystem, r.brand, r.model, r.protocol
		FROM parsed_direct_compatibility_record r
		WHERE r.snapshot_id = (SELECT MAX(id) FROM parsed_direct_compatibility_snapshot)
	`)
	if err != nil {
		return nil, fmt.Errorf("query direct compat: %w", err)
	}
	defer rows.Close()

	var records []*domain.ScrapedDirectCompatibility
	for rows.Next() {
		c := &domain.ScrapedDirectCompatibility{}
		if err := rows.Scan(&c.Ecosystem, &c.Brand, &c.Model, &c.Protocol); err != nil {
			return nil, fmt.Errorf("scan direct compat: %w", err)
		}
		records = append(records, c)
	}
	return records, rows.Err()
}

func (r *PostgresRepository) WriteCatalog(ctx context.Context, catalog *domain.Catalog) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// cascade wipes listing_device_links, direct_compatibility, bridge_ecosystem_compatibility
	if _, err := tx.ExecContext(ctx, `TRUNCATE devices CASCADE`); err != nil {
		return fmt.Errorf("truncate devices: %w", err)
	}

	for _, d := range catalog.Devices {
		attrJSON, err := json.Marshal(d.DeviceAttributes)
		if err != nil {
			return fmt.Errorf("marshal device_attributes brand=%s: %w", d.Brand, err)
		}

		if err := tx.QueryRowContext(ctx, `
			INSERT INTO devices (brand, model, category, device_attributes, taxonomy_version)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`, d.Brand, d.Model, d.Category, attrJSON, d.TaxonomyVersion).Scan(&d.Id); err != nil {
			return fmt.Errorf("insert device brand=%s: %w", d.Brand, err)
		}

		for _, l := range d.Listings {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO listing_device_links (llm_extracted_listing_id, device_id)
				VALUES ($1, $2)
			`, l.Id, d.Id); err != nil {
				return fmt.Errorf("insert listing link listing_id=%d: %w", l.Id, err)
			}
		}

		for _, dc := range d.DirectCompatibility {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO direct_compatibility (device_id, ecosystem, protocol)
				VALUES ($1, $2, $3)
			`, d.Id, dc.Ecosystem, dc.Protocol); err != nil {
				return fmt.Errorf("insert direct compat device_id=%d: %w", d.Id, err)
			}
		}

		for _, bc := range d.BridgeCompatibility {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO bridge_ecosystem_compatibility (device_id, ecosystem_source, ecosystem_target, protocol)
				VALUES ($1, $2, $3, $4)
			`, d.Id, bc.SourceEcosystem, bc.TargetEcosystem, bc.Protocol); err != nil {
				return fmt.Errorf("insert bridge compat device_id=%d: %w", d.Id, err)
			}
		}
	}

	return tx.Commit()
}
