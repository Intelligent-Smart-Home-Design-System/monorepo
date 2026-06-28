package pipeline

import (
	"encoding/json"
	"fmt"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
)

type pythonBytesDataConverter struct {
	parent converter.DataConverter
}

type deviceSelectionInputPayload struct {
	RequestProtoBytes []int `json:"request_proto_bytes"`
}

// NewDataConverter keeps the default Temporal converter for all payloads except
// DeviceSelectionInput, whose byte field must be JSON-compatible with Python.
func NewDataConverter() converter.DataConverter {
	return pythonBytesDataConverter{parent: converter.GetDefaultDataConverter()}
}

func (c pythonBytesDataConverter) ToPayload(value interface{}) (*commonpb.Payload, error) {
	switch input := value.(type) {
	case DeviceSelectionInput:
		return c.deviceSelectionInputToPayload(input)
	case *DeviceSelectionInput:
		if input == nil {
			return c.parent.ToPayload(value)
		}
		return c.deviceSelectionInputToPayload(*input)
	default:
		return c.parent.ToPayload(value)
	}
}

func (c pythonBytesDataConverter) FromPayload(payload *commonpb.Payload, valuePtr interface{}) error {
	return c.parent.FromPayload(payload, valuePtr)
}

func (c pythonBytesDataConverter) ToPayloads(values ...interface{}) (*commonpb.Payloads, error) {
	if len(values) == 0 {
		return nil, nil
	}
	payloads := &commonpb.Payloads{}
	for i, value := range values {
		payload, err := c.ToPayload(value)
		if err != nil {
			return nil, fmt.Errorf("values[%d]: %w", i, err)
		}
		payloads.Payloads = append(payloads.Payloads, payload)
	}
	return payloads, nil
}

func (c pythonBytesDataConverter) FromPayloads(payloads *commonpb.Payloads, valuePtrs ...interface{}) error {
	return c.parent.FromPayloads(payloads, valuePtrs...)
}

func (c pythonBytesDataConverter) ToString(input *commonpb.Payload) string {
	return c.parent.ToString(input)
}

func (c pythonBytesDataConverter) ToStrings(input *commonpb.Payloads) []string {
	return c.parent.ToStrings(input)
}

func (c pythonBytesDataConverter) deviceSelectionInputToPayload(input DeviceSelectionInput) (*commonpb.Payload, error) {
	data, err := json.Marshal(deviceSelectionInputPayload{
		RequestProtoBytes: bytesAsInts(input.RequestProtoBytes),
	})
	if err != nil {
		return nil, fmt.Errorf("encode device selection input: %w", err)
	}
	return &commonpb.Payload{
		Metadata: map[string][]byte{
			converter.MetadataEncoding: []byte(converter.MetadataEncodingJSON),
		},
		Data: data,
	}, nil
}

func bytesAsInts(data []byte) []int {
	out := make([]int, len(data))
	for i, value := range data {
		out[i] = int(value)
	}
	return out
}
