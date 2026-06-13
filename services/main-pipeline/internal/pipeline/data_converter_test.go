package pipeline

import (
	"encoding/json"
	"testing"
)

func TestDataConverterEncodesDeviceSelectionBytesAsJSONList(t *testing.T) {
	payload, err := NewDataConverter().ToPayload(DeviceSelectionInput{
		RequestProtoBytes: []byte{1, 2, 255},
	})
	if err != nil {
		t.Fatalf("ToPayload() error = %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(payload.GetData(), &got); err != nil {
		t.Fatalf("payload JSON unmarshal error = %v", err)
	}

	raw, ok := got["request_proto_bytes"].([]interface{})
	if !ok {
		t.Fatalf("request_proto_bytes type = %T, want JSON array", got["request_proto_bytes"])
	}
	if len(raw) != 3 || raw[0] != float64(1) || raw[1] != float64(2) || raw[2] != float64(255) {
		t.Fatalf("request_proto_bytes = %v, want [1 2 255]", raw)
	}
}
