package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/lib/pq"
)

const manualPlanEcosystem = "manual"

var errManualSelectionUnavailable = errors.New("selected catalog product is unavailable")

type CreateManualPlanRequest struct {
	Budget     float64                  `json:"budget"`
	FloorPlan  json.RawMessage          `json:"floor_plan"`
	Selections []ManualSelectionRequest `json:"selections"`
}

type ManualSelectionRequest struct {
	DeviceID   int    `json:"device_id"`
	ListingID  int    `json:"listing_id"`
	DeviceType string `json:"device_type"`
	Quantity   int    `json:"quantity"`
}

func (s *apiServer) createManualPlan(w http.ResponseWriter, r *http.Request) {
	var req CreateManualPlanRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := validateCreateManualPlan(req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	defer tx.Rollback()

	requirements, bundle, err := buildManualPlan(r.Context(), tx, req.Selections, req.Budget)
	if errors.Is(err, errManualSelectionUnavailable) {
		writeError(w, http.StatusUnprocessableEntity, "catalog_product_unavailable", err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if bundle.TotalCost > req.Budget {
		writeError(
			w,
			http.StatusBadRequest,
			"budget_exceeded",
			fmt.Sprintf("selected devices cost %.2f but budget is %.2f", bundle.TotalCost, req.Budget),
		)
		return
	}

	requirementsJSON, err := json.Marshal(requirements)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	bundlesJSON, err := json.Marshal([]Bundle{bundle})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	var planID int
	err = tx.QueryRowContext(
		r.Context(),
		`INSERT INTO frontend_plans (
			budget, main_ecosystem_id, allowed_ecosystems, excluded_ecosystems,
			requirements, status, progress, bundles, floor_plan
		)
		VALUES ($1, $2, $3, $4, $5, 'completed', 1.0, $6, $7)
		RETURNING id`,
		req.Budget,
		manualPlanEcosystem,
		pq.Array([]string{}),
		pq.Array([]string{}),
		requirementsJSON,
		bundlesJSON,
		req.FloorPlan,
	).Scan(&planID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, CreatePlanResponse{
		PlanID:  planID,
		Status:  "completed",
		Message: "Manual plan created.",
	})
}

type manualPlanQuerier interface {
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

func buildManualPlan(
	ctx context.Context,
	db manualPlanQuerier,
	selections []ManualSelectionRequest,
	budget float64,
) ([]Requirement, Bundle, error) {
	requirements := make([]Requirement, 0, len(selections))
	listings := make([]Listing, 0, len(selections))
	ecosystems := make(map[string]struct{})
	total := 0.0

	for index, selection := range selections {
		requirementID := index + 1
		listing, err := loadManualListing(ctx, db, selection, requirementID)
		if err != nil {
			return nil, Bundle{}, err
		}

		requirements = append(requirements, Requirement{
			ID:         requirementID,
			DeviceType: selection.DeviceType,
			Quantity:   selection.Quantity,
			Filters:    []RequirementFilter{},
		})
		listings = append(listings, listing)
		total += listing.Price * float64(listing.UnitsToBuy)
		if listing.ConnectionInfo.DirectEcosystem != "" && listing.ConnectionInfo.DirectEcosystem != "unknown" {
			ecosystems[listing.ConnectionInfo.DirectEcosystem] = struct{}{}
		}
	}

	ecosystemsUsed := make([]string, 0, len(ecosystems))
	for ecosystem := range ecosystems {
		ecosystemsUsed = append(ecosystemsUsed, ecosystem)
	}
	sort.Strings(ecosystemsUsed)

	return requirements, Bundle{
		ID:                  1,
		TotalCost:           total,
		QualityScore:        averageQuality(listings),
		ExtraEcosystemsUsed: max(len(ecosystemsUsed)-1, 0),
		HubsUsed:            countManualHubs(requirements),
		IsRecommended:       total <= budget,
		EcosystemsUsed:      ecosystemsUsed,
		Listings:            listings,
	}, nil
}

func loadManualListing(
	ctx context.Context,
	db manualPlanQuerier,
	selection ManualSelectionRequest,
	requirementID int,
) (Listing, error) {
	var listing Listing
	var imageURL sql.NullString
	var attributes []byte
	var devicesPerListing int

	err := db.QueryRowContext(
		ctx,
		`
			SELECT
				l.id,
				COALESCE(ps.extracted_name, d.brand || ' ' || COALESCE(d.model, d.category)),
				d.brand,
				COALESCE(d.model, d.category),
				COALESCE(d.quality, 0.0)::float,
				ps.extracted_price::float,
				COALESCE(tp.url, 'https://example.com/listing/' || l.id::text),
				ps.extracted_image_url,
				COALESCE(ps.extracted_quantity, 1),
				d.device_attributes,
				COALESCE(dc.ecosystem, 'unknown'),
				COALESCE(dc.protocol, 'unknown')
			FROM listing_device_links ldl
			JOIN devices d ON d.id = ldl.device_id
			JOIN llm_extracted_listings l ON l.id = ldl.llm_extracted_listing_id
			JOIN parsed_listing_snapshots ps ON ps.id = l.parsed_listing_snapshot_id
			LEFT JOIN page_snapshots pgs ON pgs.id = ps.page_snapshot_id
			LEFT JOIN tracked_pages tp ON tp.id = pgs.tracked_page
			LEFT JOIN LATERAL (
				SELECT ecosystem, protocol
				FROM direct_compatibility
				WHERE device_id = d.id
				ORDER BY ecosystem, protocol
				LIMIT 1
			) dc ON TRUE
			WHERE d.id = $1
			  AND l.id = $2
			  AND d.category = $3
			  AND ps.extracted_in_stock = TRUE
			  AND ps.extracted_price IS NOT NULL
		`,
		selection.DeviceID,
		selection.ListingID,
		selection.DeviceType,
	).Scan(
		&listing.ID,
		&listing.Name,
		&listing.DeviceBrand,
		&listing.DeviceModel,
		&listing.DeviceQualityScore,
		&listing.Price,
		&listing.URL,
		&imageURL,
		&devicesPerListing,
		&attributes,
		&listing.ConnectionInfo.DirectEcosystem,
		&listing.ConnectionInfo.DirectProtocol,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Listing{}, fmt.Errorf(
			"%w: device_id=%d listing_id=%d",
			errManualSelectionUnavailable,
			selection.DeviceID,
			selection.ListingID,
		)
	}
	if err != nil {
		return Listing{}, err
	}

	if len(attributes) > 0 {
		if err := json.Unmarshal(attributes, &listing.DeviceAttributes); err != nil {
			return Listing{}, err
		}
	}
	if listing.DeviceAttributes == nil {
		listing.DeviceAttributes = map[string]interface{}{}
	}
	if imageURL.Valid {
		listing.ImageURL = &imageURL.String
	}
	listing.RequirementID = requirementID
	listing.DevicesPerListing = max(devicesPerListing, 1)
	listing.UnitsToBuy = (selection.Quantity + listing.DevicesPerListing - 1) / listing.DevicesPerListing
	listing.ConnectionInfo.FinalEcosystem = listing.ConnectionInfo.DirectEcosystem
	listing.ConnectionInfo.FinalProtocol = listing.ConnectionInfo.DirectProtocol
	return listing, nil
}

func validateCreateManualPlan(req CreateManualPlanRequest) error {
	if req.Budget <= 0 {
		return errors.New("budget must be positive")
	}

	var floor map[string]interface{}
	if len(req.FloorPlan) == 0 || string(req.FloorPlan) == "null" || json.Unmarshal(req.FloorPlan, &floor) != nil || floor == nil {
		return errors.New("floor_plan must be a JSON object")
	}
	if len(req.Selections) == 0 {
		return errors.New("selections must not be empty")
	}

	deviceTypes := make(map[string]struct{}, len(req.Selections))
	for _, selection := range req.Selections {
		selection.DeviceType = strings.TrimSpace(selection.DeviceType)
		if selection.DeviceID <= 0 || selection.ListingID <= 0 {
			return errors.New("selection device_id and listing_id must be positive")
		}
		if selection.DeviceType == "" {
			return errors.New("selection device_type is required")
		}
		if selection.Quantity <= 0 {
			return errors.New("selection quantity must be positive")
		}
		if _, exists := deviceTypes[selection.DeviceType]; exists {
			return fmt.Errorf("only one selection is allowed for device_type %q", selection.DeviceType)
		}
		deviceTypes[selection.DeviceType] = struct{}{}
	}
	return nil
}

func countManualHubs(requirements []Requirement) int {
	count := 0
	for _, requirement := range requirements {
		if requirement.DeviceType == "smart_hub" {
			count += requirement.Quantity
		}
	}
	return count
}
