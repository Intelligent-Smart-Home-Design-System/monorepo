package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

type DeviceInfo struct {
	ID               int                    `json:"id"`
	Brand            string                 `json:"brand"`
	Model            string                 `json:"model"`
	Category         string                 `json:"category"`
	Quality          float64                `json:"quality"`
	DeviceAttributes map[string]interface{} `json:"device_attributes"`
}

func enrichDevices(ctx context.Context, db *sql.DB, deviceSelection map[string]interface{}) error {
	ids := extractDeviceIDs(deviceSelection)
	if len(ids) == 0 {
		return nil
	}

	devices, err := loadDevicesByID(ctx, db, ids)
	if err != nil {
		return fmt.Errorf("load devices: %w", err)
	}

	applyDeviceInfo(deviceSelection, devices)
	return nil
}

func extractDeviceIDs(ds map[string]interface{}) []int {
	seen := make(map[int]struct{})
	var ids []int

	paretoFront, ok := ds["pareto_front"].([]interface{})
	if !ok {
		return nil
	}
	for _, rawPoint := range paretoFront {
		point, ok := rawPoint.(map[string]interface{})
		if !ok {
			continue
		}
		items, ok := point["items"].([]interface{})
		if !ok {
			continue
		}
		for _, rawItem := range items {
			item, ok := rawItem.(map[string]interface{})
			if !ok {
				continue
			}
			id := toInt(item["device_id"])
			if id > 0 {
				if _, exists := seen[id]; !exists {
					seen[id] = struct{}{}
					ids = append(ids, id)
				}
			}
		}
	}
	return ids
}

func loadDevicesByID(ctx context.Context, db *sql.DB, ids []int) (map[int]DeviceInfo, error) {
	query := `
		SELECT
			d.id,
			COALESCE(d.brand, ''),
			COALESCE(d.model, ''),
			d.category,
			COALESCE(d.quality, 0),
			COALESCE(d.device_attributes, '{}'::jsonb)
		FROM devices d
		WHERE d.id = ANY($1)
	`
	rows, err := db.QueryContext(ctx, query, intSliceToInterface(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	devices := make(map[int]DeviceInfo, len(ids))
	for rows.Next() {
		var d DeviceInfo
		var attrsJSON []byte
		if err := rows.Scan(&d.ID, &d.Brand, &d.Model, &d.Category, &d.Quality, &attrsJSON); err != nil {
			return nil, err
		}
		if len(attrsJSON) > 0 {
			_ = json.Unmarshal(attrsJSON, &d.DeviceAttributes)
		}
		if d.DeviceAttributes == nil {
			d.DeviceAttributes = map[string]interface{}{}
		}
		devices[d.ID] = d
	}
	return devices, rows.Err()
}

func applyDeviceInfo(ds map[string]interface{}, devices map[int]DeviceInfo) {
	paretoFront, ok := ds["pareto_front"].([]interface{})
	if !ok {
		return
	}
	for _, rawPoint := range paretoFront {
		point, ok := rawPoint.(map[string]interface{})
		if !ok {
			continue
		}
		items, ok := point["items"].([]interface{})
		if !ok {
			continue
		}
		for _, rawItem := range items {
			item, ok := rawItem.(map[string]interface{})
			if !ok {
				continue
			}
			id := toInt(item["device_id"])
			if d, exists := devices[id]; exists {
				item["device"] = d
			}
		}
	}
}

func toInt(v interface{}) int {
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case int64:
		return int(val)
	case json.Number:
		n, _ := val.Int64()
		return int(n)
	default:
		return 0
	}
}

func intSliceToInterface(ids []int) []interface{} {
	out := make([]interface{}, len(ids))
	for i, id := range ids {
		out[i] = id
	}
	return out
}

func enrichLayoutFromSelection(layout map[string]interface{}, deviceSelection map[string]interface{}) {
	if layout == nil || deviceSelection == nil {
		return
	}

	categoryInfo := buildCategoryInfoMap(deviceSelection)
	if len(categoryInfo) == 0 {
		return
	}

	placements, ok := layout["placements"].(map[string]interface{})
	if !ok {
		return
	}

	for _, rawRoom := range placements {
		items, ok := rawRoom.([]interface{})
		if !ok {
			continue
		}
		for _, rawPlacement := range items {
			placement, ok := rawPlacement.(map[string]interface{})
			if !ok {
				continue
			}
			dev, ok := placement["device"].(map[string]interface{})
			if !ok {
				continue
			}
			deviceType, _ := dev["type"].(string)
			if info, exists := categoryInfo[deviceType]; exists {
				dev["price"] = info.price
				dev["ecosystem"] = info.ecosystem
			}
		}
	}
}

type categoryInfo struct {
	price     float64
	ecosystem string
}

func buildCategoryInfoMap(ds map[string]interface{}) map[string]categoryInfo {
	result := make(map[string]categoryInfo)

	paretoFront, ok := ds["pareto_front"].([]interface{})
	if !ok || len(paretoFront) == 0 {
		return nil
	}

	bestPoint := findBestParetoPoint(paretoFront)
	if bestPoint == nil {
		return nil
	}

	items, ok := bestPoint["items"].([]interface{})
	if !ok {
		return nil
	}

	for _, rawItem := range items {
		item, ok := rawItem.(map[string]interface{})
		if !ok {
			continue
		}

		d, ok := item["device"].(map[string]interface{})
		if !ok {
			continue
		}
		category, _ := d["category"].(string)
		if category == "" {
			continue
		}

		price := toFloat(item["price"])

		ecosystem := ""
		connection, ok := item["connection"].(map[string]interface{})
		if ok {
			direct, ok := connection["direct"].(map[string]interface{})
			if ok {
				ecosystem, _ = direct["ecosystem"].(string)
			}
		}

		result[category] = categoryInfo{price: price, ecosystem: ecosystem}
	}
	return result
}

func findBestParetoPoint(points []interface{}) map[string]interface{} {
	for _, raw := range points {
		point, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		if isRecommended, _ := point["is_recommended"].(bool); isRecommended {
			return point
		}
	}
	if len(points) > 0 {
		if point, ok := points[0].(map[string]interface{}); ok {
			return point
		}
	}
	return nil
}

func toFloat(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return 0
	}
}
