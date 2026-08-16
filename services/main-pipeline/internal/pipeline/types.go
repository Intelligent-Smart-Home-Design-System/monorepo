package pipeline

import "encoding/json"

type CustomDeviceInput struct {
    Device string `json:"device"`
    Count  int    `json:"count"`
}

type PipelineRequest struct {
    RequestID       string                 `json:"request_id,omitempty"`
    FloorPlan       map[string]interface{} `json:"floor_plan"`
    SelectedLevels  map[string]string      `json:"selected_levels,omitempty"`
    CustomDevices   []CustomDeviceInput    `json:"custom_devices,omitempty"`
    DeviceSelection map[string]interface{} `json:"device_selection,omitempty"`
}

type FloorParserInput struct {
	RequestID string                 `json:"request_id,omitempty"`
	FloorPlan map[string]interface{} `json:"floor_plan"`
}

type FloorParserOutput struct {
	FloorPlan map[string]interface{} `json:"floor_plan"`
}

type LayoutInput struct {
    RequestID      string                 `json:"request_id,omitempty"`
    FloorPlan      map[string]interface{} `json:"floor_plan"`
    SelectedLevels map[string]string      `json:"selected_levels,omitempty"`
    CustomDevices  []CustomDeviceInput    `json:"custom_devices,omitempty"`
}

type LayoutOutput struct {
	Layout       map[string]interface{} `json:"layout"`
	Dependencies map[string][]string    `json:"dependencies"`
}

type DeviceSelectionInput struct {
	RequestProtoBytes []byte `json:"request_proto_bytes"`
}

type DeviceSelectionOutput struct {
	ResponseProtoBytes []byte `json:"response_proto_bytes"`
}

type PipelineResult struct {
	RequestID       string                 `json:"request_id,omitempty"`
	ParsedFloorPlan map[string]interface{} `json:"parsed_floor_plan"`
	Layout          map[string]interface{} `json:"layout"`
	Dependencies    map[string][]string    `json:"dependencies"`
	DeviceSelection map[string]interface{} `json:"device_selection"`
}

func FromRaw(data []byte) (PipelineRequest, error) {
	var req PipelineRequest
	err := json.Unmarshal(data, &req)
	return req, err
}
