package repository

import (
	"database/sql"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
)

type ListingParseResultRepo struct {
	db *sql.DB
}

func NewListingParseResultRepo(db *sql.DB) *ListingParseResultRepo {
	return &ListingParseResultRepo{db: db}
}

func (r *ListingParseResultRepo) Save(res *domain.ListingParseResult) error {
	_, err := r.db.Exec(`
		INSERT INTO parsed_listing_snapshots
		(page_snapshot_id, parsed_at, processed, extractor_version,
		 extracted_in_stock, extracted_text, extracted_name, extracted_brand,
		 extracted_image_url, extracted_price, extracted_currency,
		 extracted_model_number, extracted_category, extracted_quantity,
		 extracted_quantity_raw, extracted_rating, extracted_review_count,
		 content_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`,
		res.PageSnapshotID, res.ParsedAt, res.Processed, res.ExtractorVer,
		res.InStock, res.Text, res.Name, res.Brand, res.ImageURL,
		res.Price, res.Currency, res.ModelNumber, res.Category, res.Quantity,
		res.QuantityRaw, res.Rating, res.ReviewCount, res.ContentHash,
	)
	return err
}