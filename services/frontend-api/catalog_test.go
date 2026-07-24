package main

import (
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestValidateCreateManualPlan(t *testing.T) {
	valid := CreateManualPlanRequest{
		Budget:    10_000,
		FloorPlan: json.RawMessage(`{"rooms":[],"walls":[]}`),
		Selections: []ManualSelectionRequest{
			{DeviceID: 1, ListingID: 10, DeviceType: "smart_lamp", Quantity: 2},
		},
	}
	if err := validateCreateManualPlan(valid); err != nil {
		t.Fatalf("expected valid manual plan request: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*CreateManualPlanRequest)
	}{
		{name: "budget", mutate: func(req *CreateManualPlanRequest) { req.Budget = 0 }},
		{name: "floor", mutate: func(req *CreateManualPlanRequest) { req.FloorPlan = nil }},
		{name: "selections", mutate: func(req *CreateManualPlanRequest) { req.Selections = nil }},
		{name: "device id", mutate: func(req *CreateManualPlanRequest) { req.Selections[0].DeviceID = 0 }},
		{name: "listing id", mutate: func(req *CreateManualPlanRequest) { req.Selections[0].ListingID = 0 }},
		{name: "device type", mutate: func(req *CreateManualPlanRequest) { req.Selections[0].DeviceType = "" }},
		{name: "quantity", mutate: func(req *CreateManualPlanRequest) { req.Selections[0].Quantity = 0 }},
		{
			name: "duplicate device type",
			mutate: func(req *CreateManualPlanRequest) {
				req.Selections = append(req.Selections, ManualSelectionRequest{
					DeviceID: 2, ListingID: 20, DeviceType: "smart_lamp", Quantity: 1,
				})
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := valid
			req.Selections = append([]ManualSelectionRequest(nil), valid.Selections...)
			test.mutate(&req)
			if err := validateCreateManualPlan(req); err == nil {
				t.Fatal("expected invalid manual plan request to be rejected")
			}
		})
	}
}

func TestCountManualHubs(t *testing.T) {
	requirements := []Requirement{
		{DeviceType: "smart_lamp", Quantity: 4},
		{DeviceType: "smart_hub", Quantity: 2},
	}
	if got := countManualHubs(requirements); got != 2 {
		t.Fatalf("expected two hubs, got %d", got)
	}
}

func TestParseCatalogProductsQueryRequiresDeviceType(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/catalog/products", nil)

	if _, err := parseCatalogProductsQuery(request); err == nil {
		t.Fatal("expected missing device_type to be rejected")
	}
}

func TestParseCatalogProductsQueryUsesDefaults(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/catalog/products?device_type=smart_lamp", nil)

	query, err := parseCatalogProductsQuery(request)
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}
	if query.DeviceType != "smart_lamp" {
		t.Fatalf("expected smart_lamp device type, got %q", query.DeviceType)
	}
	if query.Page != 1 || query.PageSize != 20 || query.Sort != "price_asc" {
		t.Fatalf("unexpected defaults: page=%d page_size=%d sort=%q", query.Page, query.PageSize, query.Sort)
	}
}

func TestParseCatalogProductsQueryReadsFilters(t *testing.T) {
	request := httptest.NewRequest(
		"GET",
		"/api/v1/catalog/products?device_type=smart_lamp&q=alice&brand=Aqara&ecosystem=yandex&protocol=zigbee&min_price=1000&max_price=5000&sort=rating_desc&page=2&page_size=30",
		nil,
	)

	query, err := parseCatalogProductsQuery(request)
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}
	if query.Search != "alice" || query.Brand != "Aqara" || query.Ecosystem != "yandex" || query.Protocol != "zigbee" {
		t.Fatalf("unexpected string filters: %+v", query)
	}
	if query.MinPrice == nil || *query.MinPrice != 1000 || query.MaxPrice == nil || *query.MaxPrice != 5000 {
		t.Fatalf("unexpected price filters: min=%v max=%v", query.MinPrice, query.MaxPrice)
	}
	if query.Sort != "rating_desc" || query.Page != 2 || query.PageSize != 30 {
		t.Fatalf("unexpected pagination or sort: %+v", query)
	}
}

func TestParseCatalogProductsQueryRejectsInvalidValues(t *testing.T) {
	tests := []string{
		"?device_type=smart_lamp&min_price=-1",
		"?device_type=smart_lamp&min_price=500&max_price=100",
		"?device_type=smart_lamp&page=0",
		"?device_type=smart_lamp&page_size=101",
		"?device_type=smart_lamp&sort=unknown",
	}

	for _, queryString := range tests {
		request := httptest.NewRequest("GET", "/api/v1/catalog/products"+queryString, nil)
		if _, err := parseCatalogProductsQuery(request); err == nil {
			t.Errorf("expected invalid query %q to be rejected", queryString)
		}
	}
}

func TestCatalogProductConditionsBuildsStableArguments(t *testing.T) {
	minPrice := 1000.0
	maxPrice := 5000.0
	query := catalogProductsQuery{
		DeviceType: "smart_lamp",
		Search:     "alice",
		Brand:      "Aqara",
		Ecosystem:  "yandex",
		Protocol:   "zigbee",
		MinPrice:   &minPrice,
		MaxPrice:   &maxPrice,
	}

	whereSQL, args := catalogProductConditions(query)
	expectedArgs := []interface{}{"smart_lamp", "%alice%", "Aqara", "yandex", "zigbee", 1000.0, 5000.0}
	if !reflect.DeepEqual(args, expectedArgs) {
		t.Fatalf("unexpected arguments: %#v", args)
	}
	expectedSQL := " WHERE LOWER(name || ' ' || brand || ' ' || model) LIKE $2 AND LOWER(brand) = LOWER($3) AND $4 = ANY(ecosystems) AND $5 = ANY(protocols) AND price >= $6 AND price <= $7"
	if whereSQL != expectedSQL {
		t.Fatalf("unexpected SQL:\n%s", whereSQL)
	}
}
