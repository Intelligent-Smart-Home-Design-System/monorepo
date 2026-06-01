package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open("postgres", env("DATABASE_DSN", "postgres://catalog:catalog@localhost:5432/smart_home?sslmode=disable"))
	if err != nil {
		panic(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		panic(err)
	}

	api := &apiServer{db: db}
	server := &http.Server{
		Addr:              env("HTTP_ADDRESS", ":8080"),
		Handler:           api.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}

type apiServer struct {
	db *sql.DB
}

func (s *apiServer) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/v1/device-types", s.listDeviceTypes)
	mux.HandleFunc("GET /api/v1/ecosystems", s.listEcosystems)
	mux.HandleFunc("GET /api/v1/presets", s.listPresets)
	mux.HandleFunc("GET /api/v1/plans", s.listPlans)
	mux.HandleFunc("POST /api/v1/plans", s.createPlan)
	mux.HandleFunc("GET /api/v1/plans/{plan_id}", s.getPlan)
	mux.HandleFunc("GET /api/v1/plans/{plan_id}/status", s.getPlanStatus)
	return withCORS(mux)
}

func (s *apiServer) listDeviceTypes(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.QueryContext(r.Context(), `SELECT id, name, filters FROM frontend_device_types ORDER BY name`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	defer rows.Close()

	var out []DeviceType
	for rows.Next() {
		var item DeviceType
		var filters []byte
		if err := rows.Scan(&item.ID, &item.Name, &filters); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		if err := json.Unmarshal(filters, &item.Filters); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *apiServer) listEcosystems(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.QueryContext(r.Context(), `SELECT id, name, description, may_be_main, image_url FROM frontend_ecosystems ORDER BY name`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	defer rows.Close()

	var out []Ecosystem
	for rows.Next() {
		var item Ecosystem
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.MayBeMain, &item.ImageURL); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *apiServer) listPresets(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.QueryContext(r.Context(), `SELECT id, name, description, requirements FROM frontend_presets ORDER BY name`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	defer rows.Close()

	var out []Preset
	for rows.Next() {
		var item Preset
		var requirements []byte
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &requirements); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		if err := json.Unmarshal(requirements, &item.Requirements); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *apiServer) listPlans(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.QueryContext(r.Context(), `SELECT id, created_at, budget, status FROM frontend_plans ORDER BY created_at DESC`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	defer rows.Close()

	var out []PlanSummary
	for rows.Next() {
		var item PlanSummary
		if err := rows.Scan(&item.PlanID, &item.CreatedAt, &item.Budget, &item.Status); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *apiServer) createPlan(w http.ResponseWriter, r *http.Request) {
	var req CreatePlanRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := validateCreatePlan(req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	requirements := normalizeRequirements(req.Requirements)
	bundles, err := s.buildBundles(r.Context(), req.Budget, req.MainEcosystemID, req.AllowedEcosystems, req.ExcludedEcosystems, requirements)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	requirementsJSON, _ := json.Marshal(requirements)
	bundlesJSON, _ := json.Marshal(bundles)

	var planID int
	err = s.db.QueryRowContext(
		r.Context(),
		`INSERT INTO frontend_plans (budget, main_ecosystem_id, allowed_ecosystems, excluded_ecosystems, requirements, status, progress, bundles)
		 VALUES ($1, $2, $3, $4, $5, 'completed', 1.0, $6)
		 RETURNING id`,
		req.Budget,
		req.MainEcosystemID,
		pq.Array(req.AllowedEcosystems),
		pq.Array(req.ExcludedEcosystems),
		requirementsJSON,
		bundlesJSON,
	).Scan(&planID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	writeJSON(w, http.StatusAccepted, CreatePlanResponse{
		PlanID:  planID,
		Status:  "accepted",
		Message: "Plan generation has started.",
	})
}

func (s *apiServer) getPlan(w http.ResponseWriter, r *http.Request) {
	planID, ok := parsePlanID(w, r)
	if !ok {
		return
	}

	var plan HomePlan
	var requirements, bundles []byte
	err := s.db.QueryRowContext(
		r.Context(),
		`SELECT id, budget, main_ecosystem_id, allowed_ecosystems, excluded_ecosystems, requirements, bundles
		 FROM frontend_plans WHERE id = $1`,
		planID,
	).Scan(
		&plan.PlanID,
		&plan.Budget,
		&plan.MainEcosystemID,
		pq.Array(&plan.AllowedEcosystems),
		pq.Array(&plan.ExcludedEcosystems),
		&requirements,
		&bundles,
	)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "not_found", "plan not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if err := json.Unmarshal(requirements, &plan.Requirements); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if err := json.Unmarshal(bundles, &plan.Bundles); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (s *apiServer) getPlanStatus(w http.ResponseWriter, r *http.Request) {
	planID, ok := parsePlanID(w, r)
	if !ok {
		return
	}

	var status PlanStatus
	var errorRaw []byte
	err := s.db.QueryRowContext(
		r.Context(),
		`SELECT id, status, progress, error FROM frontend_plans WHERE id = $1`,
		planID,
	).Scan(&status.PlanID, &status.Status, &status.Progress, &errorRaw)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "not_found", "plan not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if len(errorRaw) > 0 {
		var errorResponse ErrorResponse
		if err := json.Unmarshal(errorRaw, &errorResponse); err == nil {
			status.Error = &errorResponse
		}
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *apiServer) buildBundles(ctx context.Context, budget float64, mainEcosystem string, allowedEcosystems, excludedEcosystems []string, requirements []Requirement) ([]Bundle, error) {
	var listings []Listing
	var total float64
	ecosystems := map[string]struct{}{mainEcosystem: {}}
	policy := newEcosystemPolicy(mainEcosystem, allowedEcosystems, excludedEcosystems)

	for _, requirement := range requirements {
		rows, err := s.db.QueryContext(ctx, `
			SELECT
				best.id,
				COALESCE(pls.extracted_name, d.brand || ' ' || COALESCE(d.model, d.category)) AS name,
				d.brand,
				COALESCE(d.model, d.category) AS model,
				COALESCE(d.quality, 0.0) AS quality,
				COALESCE(best.extracted_price, 0)::float AS price,
				COALESCE(tp.url, 'https://example.com/listing/' || best.id::text) AS url,
				pls.extracted_image_url,
				COALESCE(pls.extracted_quantity, 1) AS devices_per_listing,
				COALESCE(dc.ecosystem, $2) AS direct_ecosystem,
				COALESCE(dc.protocol, 'unknown') AS direct_protocol,
				d.device_attributes
			FROM devices d
			JOIN LATERAL (
				SELECT l.id, l.parsed_listing_snapshot_id, ps.extracted_price
				FROM listing_device_links ldl
				JOIN llm_extracted_listings l ON l.id = ldl.llm_extracted_listing_id
				JOIN parsed_listing_snapshots ps ON ps.id = l.parsed_listing_snapshot_id
				WHERE ldl.device_id = d.id
				  AND ps.extracted_price IS NOT NULL
				  AND ps.extracted_in_stock = TRUE
				ORDER BY ps.extracted_price ASC
				LIMIT 1
			) best ON TRUE
			JOIN parsed_listing_snapshots pls ON pls.id = best.parsed_listing_snapshot_id
			LEFT JOIN page_snapshots pgs ON pgs.id = pls.page_snapshot_id
			LEFT JOIN tracked_pages tp ON tp.id = pgs.tracked_page
			LEFT JOIN LATERAL (
				SELECT ecosystem, protocol
				FROM direct_compatibility
				WHERE device_id = d.id
				ORDER BY (ecosystem = $2) DESC, ecosystem
				LIMIT 1
			) dc ON TRUE
			WHERE d.category = $1
			ORDER BY best.extracted_price ASC
		`, requirement.DeviceType, mainEcosystem)
		if err != nil {
			return nil, err
		}

		var selected *Listing
		for rows.Next() {
			var listing Listing
			var imageURL sql.NullString
			var attrs []byte
			var devicesPerListing int
			if err := rows.Scan(
				&listing.ID,
				&listing.Name,
				&listing.DeviceBrand,
				&listing.DeviceModel,
				&listing.DeviceQualityScore,
				&listing.Price,
				&listing.URL,
				&imageURL,
				&devicesPerListing,
				&listing.ConnectionInfo.DirectEcosystem,
				&listing.ConnectionInfo.DirectProtocol,
				&attrs,
			); err != nil {
				rows.Close()
				return nil, err
			}

			if !policy.allows(listing.ConnectionInfo.DirectEcosystem) {
				continue
			}
			if len(attrs) > 0 {
				_ = json.Unmarshal(attrs, &listing.DeviceAttributes)
			}
			if !matchesFilters(listing.DeviceAttributes, requirement.Filters) {
				continue
			}

			listing.RequirementID = requirement.ID
			listing.DevicesPerListing = max(devicesPerListing, 1)
			listing.UnitsToBuy = (requirement.Quantity + listing.DevicesPerListing - 1) / listing.DevicesPerListing
			if imageURL.Valid {
				listing.ImageURL = &imageURL.String
			}
			listing.ConnectionInfo.FinalEcosystem = mainEcosystem
			listing.ConnectionInfo.FinalProtocol = listing.ConnectionInfo.DirectProtocol
			selected = &listing
			break
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()

		if selected == nil {
			continue
		}

		total += selected.Price * float64(selected.UnitsToBuy)
		ecosystems[selected.ConnectionInfo.DirectEcosystem] = struct{}{}
		listings = append(listings, *selected)
	}

	if len(listings) == 0 {
		return []Bundle{}, nil
	}

	ecosystemsUsed := make([]string, 0, len(ecosystems))
	for ecosystem := range ecosystems {
		ecosystemsUsed = append(ecosystemsUsed, ecosystem)
	}
	sort.Strings(ecosystemsUsed)

	return []Bundle{{
		ID:                  1,
		TotalCost:           total,
		QualityScore:        averageQuality(listings),
		ExtraEcosystemsUsed: max(len(ecosystemsUsed)-1, 0),
		HubsUsed:            0,
		IsRecommended:       total <= budget,
		EcosystemsUsed:      ecosystemsUsed,
		Listings:            listings,
	}}, nil
}

func normalizeRequirements(input []CreateRequirement) []Requirement {
	out := make([]Requirement, 0, len(input))
	for index, item := range input {
		id := item.ID
		if id <= 0 {
			id = index + 1
		}
		out = append(out, Requirement{
			ID:         id,
			DeviceType: item.DeviceType,
			Quantity:   item.Quantity,
			Filters:    item.Filters,
		})
	}
	return out
}

func averageQuality(listings []Listing) float64 {
	if len(listings) == 0 {
		return 0
	}
	var sum float64
	for _, listing := range listings {
		sum += listing.DeviceQualityScore
	}
	return sum / float64(len(listings))
}

func parsePlanID(w http.ResponseWriter, r *http.Request) (int, bool) {
	planID, err := strconv.Atoi(r.PathValue("plan_id"))
	if err != nil || planID <= 0 {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid plan_id")
		return 0, false
	}
	return planID, true
}

func validateCreatePlan(req CreatePlanRequest) error {
	if req.Budget <= 0 {
		return errors.New("budget must be positive")
	}
	if strings.TrimSpace(req.MainEcosystemID) == "" {
		return errors.New("main_ecosystem_id is required")
	}
	if len(req.Requirements) == 0 {
		return errors.New("requirements must not be empty")
	}
	for _, requirement := range req.Requirements {
		if strings.TrimSpace(requirement.DeviceType) == "" {
			return errors.New("requirement device_type is required")
		}
		if requirement.Quantity <= 0 {
			return errors.New("requirement quantity must be positive")
		}
	}
	return nil
}

type ecosystemPolicy struct {
	allowed  map[string]struct{}
	excluded map[string]struct{}
}

func newEcosystemPolicy(mainEcosystem string, allowed, excluded []string) ecosystemPolicy {
	policy := ecosystemPolicy{
		allowed:  make(map[string]struct{}, len(allowed)+1),
		excluded: make(map[string]struct{}, len(excluded)),
	}
	for _, ecosystem := range allowed {
		ecosystem = strings.TrimSpace(ecosystem)
		if ecosystem != "" {
			policy.allowed[ecosystem] = struct{}{}
		}
	}
	if len(policy.allowed) > 0 && mainEcosystem != "" {
		policy.allowed[mainEcosystem] = struct{}{}
	}
	for _, ecosystem := range excluded {
		ecosystem = strings.TrimSpace(ecosystem)
		if ecosystem != "" {
			policy.excluded[ecosystem] = struct{}{}
		}
	}
	return policy
}

func (p ecosystemPolicy) allows(ecosystem string) bool {
	if _, excluded := p.excluded[ecosystem]; excluded {
		return false
	}
	if len(p.allowed) == 0 {
		return true
	}
	_, allowed := p.allowed[ecosystem]
	return allowed
}

func matchesFilters(attrs map[string]interface{}, filters []RequirementFilter) bool {
	if len(filters) == 0 {
		return true
	}
	for _, filter := range filters {
		value, exists := attrs[filter.Field]
		if !matchesFilter(value, exists, filter) {
			return false
		}
	}
	return true
}

func matchesFilter(attr interface{}, exists bool, filter RequirementFilter) bool {
	switch filter.Operation {
	case "exists":
		return exists && attr != nil
	}
	if !exists || attr == nil {
		return false
	}

	switch filter.Operation {
	case "eq":
		comparison, ok := compareValues(attr, filter.Value)
		return ok && comparison == 0
	case "neq":
		comparison, ok := compareValues(attr, filter.Value)
		return ok && comparison != 0
	case "gt":
		comparison, ok := compareValues(attr, filter.Value)
		return ok && comparison > 0
	case "gte":
		comparison, ok := compareValues(attr, filter.Value)
		return ok && comparison >= 0
	case "lt":
		comparison, ok := compareValues(attr, filter.Value)
		return ok && comparison < 0
	case "lte":
		comparison, ok := compareValues(attr, filter.Value)
		return ok && comparison <= 0
	case "contains":
		values, ok := attr.([]interface{})
		if !ok {
			return false
		}
		for _, item := range values {
			comparison, ok := compareValues(item, filter.Value)
			if ok && comparison == 0 {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func compareValues(left, right interface{}) (int, bool) {
	leftNumber, leftIsNumber := numericValue(left)
	rightNumber, rightIsNumber := numericValue(right)
	if leftIsNumber && rightIsNumber {
		switch {
		case leftNumber < rightNumber:
			return -1, true
		case leftNumber > rightNumber:
			return 1, true
		default:
			return 0, true
		}
	}

	leftString, leftOK := left.(string)
	rightString, rightOK := right.(string)
	if leftOK && rightOK {
		return strings.Compare(leftString, rightString), true
	}

	leftBool, leftOK := left.(bool)
	rightBool, rightOK := right.(bool)
	if leftOK && rightOK {
		switch {
		case leftBool == rightBool:
			return 0, true
		case !leftBool && rightBool:
			return -1, true
		default:
			return 1, true
		}
	}

	return 0, false
}

func numericValue(value interface{}) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case json.Number:
		value, err := typed.Float64()
		return value, err == nil
	default:
		return 0, false
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, out interface{}) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 10<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, ErrorResponse{Message: message, Code: &code})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

type DeviceType struct {
	ID      string                  `json:"id"`
	Name    string                  `json:"name"`
	Filters []DeviceTypeFilterField `json:"filters"`
}

type DeviceTypeFilterField struct {
	Name       string    `json:"name"`
	Field      string    `json:"field"`
	ValueType  string    `json:"value_type"`
	EnumValues *[]string `json:"enum_values"`
	Operations []string  `json:"operations"`
}

type Ecosystem struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	MayBeMain   bool    `json:"may_be_main"`
	ImageURL    *string `json:"image_url"`
}

type Preset struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Description  *string       `json:"description"`
	Requirements []Requirement `json:"requirements"`
}

type Requirement struct {
	ID         int                 `json:"id"`
	DeviceType string              `json:"device_type"`
	Quantity   int                 `json:"quantity"`
	Filters    []RequirementFilter `json:"filters"`
}

type CreateRequirement struct {
	ID         int                 `json:"id,omitempty"`
	DeviceType string              `json:"device_type"`
	Quantity   int                 `json:"quantity"`
	Filters    []RequirementFilter `json:"filters"`
}

type RequirementFilter struct {
	Field     string      `json:"field"`
	Operation string      `json:"operation"`
	Value     interface{} `json:"value"`
}

type CreatePlanRequest struct {
	Budget             float64             `json:"budget"`
	MainEcosystemID    string              `json:"main_ecosystem_id"`
	AllowedEcosystems  []string            `json:"allowed_ecosystems"`
	ExcludedEcosystems []string            `json:"excluded_ecosystems"`
	Requirements       []CreateRequirement `json:"requirements"`
}

type CreatePlanResponse struct {
	PlanID  int    `json:"plan_id"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type PlanSummary struct {
	PlanID    int       `json:"plan_id"`
	CreatedAt time.Time `json:"created_at"`
	Budget    float64   `json:"budget"`
	Status    string    `json:"status"`
}

type PlanStatus struct {
	PlanID   int            `json:"plan_id"`
	Status   string         `json:"status"`
	Progress *float64       `json:"progress"`
	Error    *ErrorResponse `json:"error,omitempty"`
}

type HomePlan struct {
	PlanID             int           `json:"plan_id"`
	Budget             float64       `json:"budget"`
	MainEcosystemID    string        `json:"main_ecosystem_id"`
	AllowedEcosystems  []string      `json:"allowed_ecosystems"`
	ExcludedEcosystems []string      `json:"excluded_ecosystems"`
	Requirements       []Requirement `json:"requirements"`
	Bundles            []Bundle      `json:"bundles"`
}

type Bundle struct {
	ID                  int       `json:"id"`
	TotalCost           float64   `json:"total_cost"`
	QualityScore        float64   `json:"quality_score"`
	ExtraEcosystemsUsed int       `json:"extra_ecosystems_used"`
	HubsUsed            int       `json:"hubs_used"`
	IsRecommended       bool      `json:"is_recommended"`
	EcosystemsUsed      []string  `json:"ecosystems_used"`
	Listings            []Listing `json:"listings"`
}

type Listing struct {
	ID                 int                    `json:"id"`
	Name               string                 `json:"name"`
	DeviceBrand        string                 `json:"device_brand"`
	DeviceModel        string                 `json:"device_model"`
	DeviceQualityScore float64                `json:"device_quality_score"`
	Price              float64                `json:"price"`
	URL                string                 `json:"url"`
	ImageURL           *string                `json:"image_url"`
	DevicesPerListing  int                    `json:"devices_per_listing"`
	UnitsToBuy         int                    `json:"units_to_buy"`
	RequirementID      int                    `json:"requirement_id"`
	DeviceAttributes   map[string]interface{} `json:"device_attributes"`
	ConnectionInfo     ConnectionInfo         `json:"connection_info"`
}

type ConnectionInfo struct {
	DirectEcosystem            string  `json:"direct_ecosystem"`
	DirectProtocol             string  `json:"direct_protocol"`
	DirectHubSelectedListingID *int    `json:"direct_hub_selected_listing_id"`
	DirectDescription          *string `json:"direct_description"`
	FinalEcosystem             string  `json:"final_ecosystem"`
	FinalProtocol              string  `json:"final_protocol"`
	FinalHubSelectedListingID  *int    `json:"final_hub_selected_listing_id"`
	FinalDescription           *string `json:"final_description"`
}

type ErrorResponse struct {
	Message string  `json:"message"`
	Code    *string `json:"code"`
	Details *string `json:"details"`
}
