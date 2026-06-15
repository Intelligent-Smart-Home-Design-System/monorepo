package main

import "testing"

func TestNormalizeRequirementsKeepsProvidedIDs(t *testing.T) {
	requirements := normalizeRequirements([]CreateRequirement{
		{ID: 42, DeviceType: "smart_lamp", Quantity: 2},
		{DeviceType: "gas_leak_sensor", Quantity: 1},
	})

	if requirements[0].ID != 42 {
		t.Fatalf("expected provided id to be preserved, got %d", requirements[0].ID)
	}
	if requirements[1].ID != 2 {
		t.Fatalf("expected missing id to fall back to position, got %d", requirements[1].ID)
	}
}

func TestMatchesFilters(t *testing.T) {
	attrs := map[string]interface{}{
		"socket_type":        "E27",
		"battery_life_years": float64(2),
		"gas_types":          []interface{}{"methane", "natural_gas"},
	}

	filters := []RequirementFilter{
		{Field: "socket_type", Operation: "eq", Value: "E27"},
		{Field: "battery_life_years", Operation: "gte", Value: float64(2)},
		{Field: "gas_types", Operation: "contains", Value: "methane"},
	}
	if !matchesFilters(attrs, filters) {
		t.Fatal("expected filters to match device attributes")
	}

	filters[0].Value = "E14"
	if matchesFilters(attrs, filters) {
		t.Fatal("expected mismatched enum filter to reject device attributes")
	}
}

func TestEcosystemPolicyAllowsAllWhenAllowedListIsEmpty(t *testing.T) {
	policy := newEcosystemPolicy("aqara", nil, nil)
	if !policy.allows("yandex") {
		t.Fatal("expected empty allowed_ecosystems to allow non-main ecosystems")
	}

	policy = newEcosystemPolicy("aqara", []string{"tuya"}, []string{"yandex"})
	if !policy.allows("tuya") {
		t.Fatal("expected explicitly allowed ecosystem to pass")
	}
	if !policy.allows("aqara") {
		t.Fatal("expected main ecosystem to pass when allowed_ecosystems is constrained")
	}
	if policy.allows("yandex") {
		t.Fatal("expected excluded ecosystem to be rejected")
	}
}
