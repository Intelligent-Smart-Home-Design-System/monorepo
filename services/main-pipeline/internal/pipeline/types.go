package pipeline

import "encoding/json"

type PipelineRequest struct {
	RequestID       string                 `json:"request_id,omitempty"`
	FloorPlan       map[string]interface{} `json:"floor_plan"`
	SelectedLevels  map[string]string      `json:"selected_levels"`
	DeviceSelection map[string]interface{} `json:"device_selection"`
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
	SelectedLevels map[string]string      `json:"selected_levels"`
}

type LayoutOutput struct {
	Layout map[string]interface{} `json:"layout"`
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
	DeviceSelection map[string]interface{} `json:"device_selection"`
}

func FromRaw(data []byte) (PipelineRequest, error) {
	var req PipelineRequest
	err := json.Unmarshal(data, &req)
	return req, err
}
