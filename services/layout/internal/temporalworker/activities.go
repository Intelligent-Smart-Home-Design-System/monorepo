package temporalworker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/events/engine"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/point"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/rules/storage"
	"go.temporal.io/sdk/activity"
)

type PlaceDevicesInput struct {
	RequestID      string                 `json:"request_id,omitempty"`
	FloorPlan      map[string]interface{} `json:"floor_plan"`
	SelectedLevels map[string]string      `json:"selected_levels"`
}

type PlaceDevicesOutput struct {
	Layout map[string]interface{} `json:"layout"`
}

type Activities struct {
	engine *engine.Engine
}

func NewActivities(tracksPath, devicesPath string) (*Activities, error) {
	if err := configs.LoadTracksConfig(tracksPath); err != nil {
		return nil, fmt.Errorf("load tracks config: %w", err)
	}
	if err := configs.LoadDevicesConfig(devicesPath); err != nil {
		return nil, fmt.Errorf("load devices config: %w", err)
	}
	rules := storage.NewStorage()
	rules.LoadAllRules()
	return &Activities{engine: engine.NewEngine(rules)}, nil
}

func (a *Activities) PlaceDevices(ctx context.Context, input PlaceDevicesInput) (PlaceDevicesOutput, error) {
	activity.GetLogger(ctx).Info("layout request received", "request_id", input.RequestID)
	apartmentModel, err := toApartment(input.FloorPlan)
	if err != nil {
		return PlaceDevicesOutput{}, err
	}
	apartmentModel.Index()

	layout, err := a.engine.PlaceDevices(apartmentModel, input.SelectedLevels)
	if err != nil {
		return PlaceDevicesOutput{}, err
	}

	raw, err := json.Marshal(layout)
	if err != nil {
		return PlaceDevicesOutput{}, err
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return PlaceDevicesOutput{}, err
	}
	return PlaceDevicesOutput{Layout: out}, nil
}

func toApartment(plan map[string]interface{}) (*apartment.Apartment, error) {
	raw, err := json.Marshal(normalizePlan(plan))
	if err != nil {
		return nil, err
	}
	var ap apartment.Apartment
	if err := json.Unmarshal(raw, &ap); err != nil {
		return nil, err
	}
	return &ap, nil
}

func normalizePlan(plan map[string]interface{}) map[string]interface{} {
	out := copyMap(plan)
	if _, ok := out["door"]; !ok {
		if doors, ok := out["doors"]; ok {
			out["door"] = doors
		}
	}
	for _, key := range []string{"walls", "doors", "windows", "rooms", "furniture", "plumbing", "appliances"} {
		if items, ok := out[key].([]interface{}); ok {
			for _, item := range items {
				if entry, ok := item.(map[string]interface{}); ok {
					if pts, ok := entry["points"]; ok {
						entry["points"] = normalizePoints(pts)
					}
					if area, ok := entry["area"]; ok {
						entry["area"] = normalizePoints(area)
					}
				}
			}
		}
	}
	if _, ok := out["furniture"]; !ok {
		out["furniture"] = []interface{}{}
	}
	if _, ok := out["plumbing"]; !ok {
		out["plumbing"] = []interface{}{}
	}
	if _, ok := out["appliances"]; !ok {
		out["appliances"] = []interface{}{}
	}
	return out
}

func copyMap(input map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func normalizePoints(value interface{}) interface{} {
	points, ok := value.([]interface{})
	if !ok {
		return value
	}
	out := make([]point.Point, 0, len(points))
	for _, rawPoint := range points {
		switch p := rawPoint.(type) {
		case []interface{}:
			if len(p) >= 2 {
				out = append(out, point.Point{X: toFloat(p[0]), Y: toFloat(p[1])})
			}
		case map[string]interface{}:
			out = append(out, point.Point{X: toFloat(first(p, "x", "X")), Y: toFloat(first(p, "y", "Y"))})
		}
	}
	return out
}

func first(input map[string]interface{}, keys ...string) interface{} {
	for _, key := range keys {
		if value, ok := input[key]; ok {
			return value
		}
	}
	return 0
}

func toFloat(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	default:
		return 0
	}
}
