package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/lib/pq"
)

const catalogProductsCTE = `
WITH products AS (
	SELECT
		d.id AS device_id,
		d.category AS device_type,
		COALESCE(best.extracted_name, d.brand || ' ' || COALESCE(d.model, d.category)) AS name,
		d.brand,
		COALESCE(d.model, '') AS model,
		COALESCE(d.quality, 0)::float AS quality,
		d.device_attributes,
		best.listing_id,
		best.extracted_price::float AS price,
		COALESCE(best.extracted_currency, '') AS currency,
		best.extracted_image_url,
		best.extracted_rating::float AS rating,
		best.extracted_review_count,
		COALESCE(best.extracted_quantity, 1) AS devices_per_listing,
		COALESCE(tp.source_name, '') AS merchant,
		COALESCE(tp.url, '') AS url,
		ARRAY(
			SELECT DISTINCT dc.ecosystem
			FROM direct_compatibility dc
			WHERE dc.device_id = d.id
			ORDER BY dc.ecosystem
		) AS ecosystems,
		ARRAY(
			SELECT DISTINCT dc.protocol
			FROM direct_compatibility dc
			WHERE dc.device_id = d.id
			ORDER BY dc.protocol
		) AS protocols
	FROM devices d
	JOIN LATERAL (
		SELECT
			l.id AS listing_id,
			ps.page_snapshot_id,
			ps.extracted_name,
			ps.extracted_price,
			ps.extracted_currency,
			ps.extracted_image_url,
			ps.extracted_rating,
			ps.extracted_review_count,
			ps.extracted_quantity
		FROM listing_device_links ldl
		JOIN llm_extracted_listings l ON l.id = ldl.llm_extracted_listing_id
		JOIN parsed_listing_snapshots ps ON ps.id = l.parsed_listing_snapshot_id
		WHERE ldl.device_id = d.id
		  AND ps.extracted_in_stock = TRUE
		  AND ps.extracted_price IS NOT NULL
		ORDER BY ps.extracted_price ASC, ps.parsed_at DESC, l.id
		LIMIT 1
	) best ON TRUE
	LEFT JOIN page_snapshots pgs ON pgs.id = best.page_snapshot_id
	LEFT JOIN tracked_pages tp ON tp.id = pgs.tracked_page
	WHERE d.category = $1
	  AND d.taxonomy_version = 'test'
)
`

type catalogProductsQuery struct {
	DeviceType string
	Search     string
	Brand      string
	Ecosystem  string
	Protocol   string
	MinPrice   *float64
	MaxPrice   *float64
	Sort       string
	Page       int
	PageSize   int
}

type CatalogCategory struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	ProductCount int     `json:"product_count"`
	MinPrice     float64 `json:"min_price"`
}

type CatalogProduct struct {
	DeviceID          int                    `json:"device_id"`
	ListingID         int                    `json:"listing_id"`
	DeviceType        string                 `json:"device_type"`
	Name              string                 `json:"name"`
	Brand             string                 `json:"brand"`
	Model             string                 `json:"model"`
	Quality           float64                `json:"quality"`
	Price             float64                `json:"price"`
	Currency          string                 `json:"currency"`
	ImageURL          *string                `json:"image_url"`
	Rating            float64                `json:"rating"`
	ReviewCount       int                    `json:"review_count"`
	DevicesPerListing int                    `json:"devices_per_listing"`
	Merchant          string                 `json:"merchant"`
	URL               string                 `json:"url"`
	Available         bool                   `json:"available"`
	DeviceAttributes  map[string]interface{} `json:"device_attributes"`
	Ecosystems        []string               `json:"ecosystems"`
	Protocols         []string               `json:"protocols"`
}

type CatalogProductFilters struct {
	Brands     []string `json:"brands"`
	Ecosystems []string `json:"ecosystems"`
	Protocols  []string `json:"protocols"`
	MinPrice   float64  `json:"min_price"`
	MaxPrice   float64  `json:"max_price"`
}

type CatalogProductsResponse struct {
	Items      []CatalogProduct      `json:"items"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"page_size"`
	Total      int                   `json:"total"`
	TotalPages int                   `json:"total_pages"`
	Filters    CatalogProductFilters `json:"filters"`
}

func (s *apiServer) listCatalogCategories(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.QueryContext(r.Context(), `
		SELECT
			d.category,
			COALESCE(fdt.name, INITCAP(REPLACE(d.category, '_', ' '))) AS name,
			COUNT(DISTINCT d.id) AS product_count,
			MIN(ps.extracted_price)::float AS min_price
		FROM devices d
		JOIN listing_device_links ldl ON ldl.device_id = d.id
		JOIN llm_extracted_listings l ON l.id = ldl.llm_extracted_listing_id
		JOIN parsed_listing_snapshots ps ON ps.id = l.parsed_listing_snapshot_id
		LEFT JOIN frontend_device_types fdt ON fdt.id = d.category
		WHERE ps.extracted_in_stock = TRUE
		  AND ps.extracted_price IS NOT NULL
		  AND d.taxonomy_version = 'test'
		GROUP BY d.category, fdt.name
		ORDER BY name
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	defer rows.Close()

	out := make([]CatalogCategory, 0)
	for rows.Next() {
		var category CatalogCategory
		if err := rows.Scan(&category.ID, &category.Name, &category.ProductCount, &category.MinPrice); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		out = append(out, category)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *apiServer) listCatalogProducts(w http.ResponseWriter, r *http.Request) {
	query, err := parseCatalogProductsQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	whereSQL, args := catalogProductConditions(query)
	var total int
	if err := s.db.QueryRowContext(
		r.Context(),
		catalogProductsCTE+"SELECT COUNT(*) FROM products"+whereSQL,
		args...,
	).Scan(&total); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	orderSQL := catalogProductOrder(query.Sort)
	dataArgs := append(append([]interface{}{}, args...), query.PageSize, (query.Page-1)*query.PageSize)
	limitPosition := len(args) + 1
	rows, err := s.db.QueryContext(
		r.Context(),
		catalogProductsCTE+`
			SELECT
				device_id, listing_id, device_type, name, brand, model, quality,
				price, currency, extracted_image_url, rating, extracted_review_count,
				devices_per_listing, merchant, url, device_attributes, ecosystems, protocols
			FROM products`+
			whereSQL+orderSQL+
			fmt.Sprintf(" LIMIT $%d OFFSET $%d", limitPosition, limitPosition+1),
		dataArgs...,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	defer rows.Close()

	items := make([]CatalogProduct, 0)
	for rows.Next() {
		var product CatalogProduct
		var imageURL sql.NullString
		var attributes []byte
		if err := rows.Scan(
			&product.DeviceID,
			&product.ListingID,
			&product.DeviceType,
			&product.Name,
			&product.Brand,
			&product.Model,
			&product.Quality,
			&product.Price,
			&product.Currency,
			&imageURL,
			&product.Rating,
			&product.ReviewCount,
			&product.DevicesPerListing,
			&product.Merchant,
			&product.URL,
			&attributes,
			pq.Array(&product.Ecosystems),
			pq.Array(&product.Protocols),
		); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		if imageURL.Valid {
			product.ImageURL = &imageURL.String
		}
		if len(attributes) > 0 {
			if err := json.Unmarshal(attributes, &product.DeviceAttributes); err != nil {
				writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
				return
			}
		}
		if product.DeviceAttributes == nil {
			product.DeviceAttributes = map[string]interface{}{}
		}
		product.Available = true
		items = append(items, product)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	filters, err := s.catalogProductFilters(r, query.DeviceType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + query.PageSize - 1) / query.PageSize
	}
	writeJSON(w, http.StatusOK, CatalogProductsResponse{
		Items:      items,
		Page:       query.Page,
		PageSize:   query.PageSize,
		Total:      total,
		TotalPages: totalPages,
		Filters:    filters,
	})
}

func parseCatalogProductsQuery(r *http.Request) (catalogProductsQuery, error) {
	values := r.URL.Query()
	query := catalogProductsQuery{
		DeviceType: strings.TrimSpace(values.Get("device_type")),
		Search:     strings.TrimSpace(values.Get("q")),
		Brand:      strings.TrimSpace(values.Get("brand")),
		Ecosystem:  strings.TrimSpace(values.Get("ecosystem")),
		Protocol:   strings.TrimSpace(values.Get("protocol")),
		Sort:       strings.TrimSpace(values.Get("sort")),
		Page:       1,
		PageSize:   20,
	}
	if query.DeviceType == "" {
		return query, errors.New("device_type is required")
	}
	if len(query.Search) > 200 {
		return query, errors.New("q must be at most 200 characters")
	}

	var err error
	if query.MinPrice, err = optionalNonNegativeFloat(values.Get("min_price"), "min_price"); err != nil {
		return query, err
	}
	if query.MaxPrice, err = optionalNonNegativeFloat(values.Get("max_price"), "max_price"); err != nil {
		return query, err
	}
	if query.MinPrice != nil && query.MaxPrice != nil && *query.MaxPrice < *query.MinPrice {
		return query, errors.New("max_price must be greater than or equal to min_price")
	}
	if value := values.Get("page"); value != "" {
		query.Page, err = positiveInt(value, "page")
		if err != nil {
			return query, err
		}
	}
	if value := values.Get("page_size"); value != "" {
		query.PageSize, err = positiveInt(value, "page_size")
		if err != nil {
			return query, err
		}
		if query.PageSize > 100 {
			return query, errors.New("page_size must not exceed 100")
		}
	}
	if query.Sort == "" {
		query.Sort = "price_asc"
	}
	switch query.Sort {
	case "price_asc", "price_desc", "rating_desc", "name_asc":
	default:
		return query, errors.New("unsupported sort value")
	}
	return query, nil
}

func optionalNonNegativeFloat(raw, name string) (*float64, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || value < 0 {
		return nil, fmt.Errorf("%s must be a non-negative number", name)
	}
	return &value, nil
}

func positiveInt(raw, name string) (int, error) {
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return value, nil
}

func catalogProductConditions(query catalogProductsQuery) (string, []interface{}) {
	conditions := make([]string, 0, 6)
	args := []interface{}{query.DeviceType}
	add := func(condition string, value interface{}) {
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf(condition, len(args)))
	}

	if query.Search != "" {
		add("LOWER(name || ' ' || brand || ' ' || model) LIKE $%d", "%"+strings.ToLower(query.Search)+"%")
	}
	if query.Brand != "" {
		add("LOWER(brand) = LOWER($%d)", query.Brand)
	}
	if query.Ecosystem != "" {
		add("$%d = ANY(ecosystems)", query.Ecosystem)
	}
	if query.Protocol != "" {
		add("$%d = ANY(protocols)", query.Protocol)
	}
	if query.MinPrice != nil {
		add("price >= $%d", *query.MinPrice)
	}
	if query.MaxPrice != nil {
		add("price <= $%d", *query.MaxPrice)
	}
	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func catalogProductOrder(sortValue string) string {
	switch sortValue {
	case "price_desc":
		return " ORDER BY price DESC, name, device_id"
	case "rating_desc":
		return " ORDER BY rating DESC, extracted_review_count DESC, name, device_id"
	case "name_asc":
		return " ORDER BY name, device_id"
	default:
		return " ORDER BY price, name, device_id"
	}
}

func (s *apiServer) catalogProductFilters(r *http.Request, deviceType string) (CatalogProductFilters, error) {
	var filters CatalogProductFilters
	err := s.db.QueryRowContext(
		r.Context(),
		catalogProductsCTE+`
			SELECT
				COALESCE(ARRAY_AGG(DISTINCT brand ORDER BY brand), '{}'),
				COALESCE(ARRAY_AGG(DISTINCT ecosystem ORDER BY ecosystem)
					FILTER (WHERE ecosystem IS NOT NULL), '{}'),
				COALESCE(ARRAY_AGG(DISTINCT protocol ORDER BY protocol)
					FILTER (WHERE protocol IS NOT NULL), '{}'),
				COALESCE(MIN(price), 0),
				COALESCE(MAX(price), 0)
			FROM products
			LEFT JOIN LATERAL UNNEST(ecosystems) AS ecosystem_values(ecosystem) ON TRUE
			LEFT JOIN LATERAL UNNEST(protocols) AS protocol_values(protocol) ON TRUE
		`,
		deviceType,
	).Scan(
		pq.Array(&filters.Brands),
		pq.Array(&filters.Ecosystems),
		pq.Array(&filters.Protocols),
		&filters.MinPrice,
		&filters.MaxPrice,
	)
	return filters, err
}
