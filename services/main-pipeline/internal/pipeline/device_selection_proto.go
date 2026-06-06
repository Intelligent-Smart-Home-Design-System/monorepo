package pipeline

import (
	"fmt"
	"math"
	"strings"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	opUnspecified = 0
	opEQ          = 1
	opNEQ         = 2
	opGT          = 3
	opGTE         = 4
	opLT          = 5
	opLTE         = 6
	opContains    = 7
	opExists      = 8
)

func DeviceSelectionInputFromJSON(request map[string]interface{}) (DeviceSelectionInput, error) {
	payload, err := encodeDeviceSelectionRequest(request)
	if err != nil {
		return DeviceSelectionInput{}, err
	}
	return DeviceSelectionInput{RequestProtoBytes: payload}, nil
}

func DeviceSelectionOutputToJSON(output DeviceSelectionOutput) (map[string]interface{}, error) {
	points, err := decodeDeviceSelectionResponse(output.ResponseProtoBytes)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"num_solutions": len(points),
		"pareto_front":  points,
	}, nil
}

func encodeDeviceSelectionRequest(request map[string]interface{}) ([]byte, error) {
	var out []byte
	out = appendString(out, 1, stringValue(request, "main_ecosystem"))
	out = appendDouble(out, 2, numberValue(request, "budget"))
	for _, raw := range sliceValue(request, "requirements") {
		requirement, ok := raw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("device_selection.requirements must contain objects")
		}
		encodedRequirement, err := encodeDeviceRequirement(requirement)
		if err != nil {
			return nil, err
		}
		out = protowire.AppendTag(out, 3, protowire.BytesType)
		out = protowire.AppendBytes(out, encodedRequirement)
	}
	for _, ecosystem := range stringSliceValue(request, "include_ecosystems") {
		out = appendString(out, 4, ecosystem)
	}
	for _, ecosystem := range stringSliceValue(request, "exclude_ecosystems") {
		out = appendString(out, 5, ecosystem)
	}
	if value, ok := request["max_solutions"]; ok {
		out = appendVarint(out, 6, uint64(intValue(value)))
	}
	if value, ok := request["random_seed"]; ok {
		out = appendVarint(out, 7, uint64(int64Value(value)))
	}
	if value, ok := request["time_budget_seconds"]; ok {
		out = appendDoubleValue(out, 8, floatValue(value))
	}
	return out, nil
}

func encodeDeviceRequirement(requirement map[string]interface{}) ([]byte, error) {
	var out []byte
	out = appendVarint(out, 1, uint64(intValue(requirement["requirement_id"])))
	out = appendString(out, 2, stringValue(requirement, "device_type"))
	out = appendVarint(out, 3, uint64(intValue(requirement["count"])))
	if boolValue(requirement, "connect_to_main_ecosystem") {
		out = appendVarint(out, 4, 1)
	}
	for _, raw := range sliceValue(requirement, "filters") {
		filter, ok := raw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("device_selection.requirements.filters must contain objects")
		}
		encodedFilter, err := encodeFilter(filter)
		if err != nil {
			return nil, err
		}
		out = protowire.AppendTag(out, 5, protowire.BytesType)
		out = protowire.AppendBytes(out, encodedFilter)
	}
	return out, nil
}

func encodeFilter(filter map[string]interface{}) ([]byte, error) {
	var out []byte
	out = appendString(out, 1, stringValue(filter, "field"))
	out = appendVarint(out, 2, uint64(filterOp(stringValue(filter, "op"))))
	if value, ok := filter["value"]; ok {
		structValue, err := structpb.NewValue(value)
		if err != nil {
			return nil, fmt.Errorf("encode filter value: %w", err)
		}
		rawValue, err := proto.Marshal(structValue)
		if err != nil {
			return nil, fmt.Errorf("marshal filter value: %w", err)
		}
		out = protowire.AppendTag(out, 3, protowire.BytesType)
		out = protowire.AppendBytes(out, rawValue)
	}
	return out, nil
}

func decodeDeviceSelectionResponse(data []byte) ([]map[string]interface{}, error) {
	var points []map[string]interface{}
	for len(data) > 0 {
		field, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return nil, protowire.ParseError(n)
		}
		data = data[n:]
		if field != 1 || typ != protowire.BytesType {
			skipped, err := consumeFieldValue(typ, data)
			if err != nil {
				return nil, err
			}
			data = data[skipped:]
			continue
		}
		rawPoint, n := protowire.ConsumeBytes(data)
		if n < 0 {
			return nil, protowire.ParseError(n)
		}
		point, err := decodeParetoPoint(rawPoint)
		if err != nil {
			return nil, err
		}
		points = append(points, point)
		data = data[n:]
	}
	return points, nil
}

func decodeParetoPoint(data []byte) (map[string]interface{}, error) {
	point := map[string]interface{}{
		"items": []interface{}{},
	}
	for len(data) > 0 {
		field, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return nil, protowire.ParseError(n)
		}
		data = data[n:]
		switch field {
		case 1:
			rawListing, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			listing, err := decodeSelectedListing(rawListing)
			if err != nil {
				return nil, err
			}
			point["items"] = append(point["items"].([]interface{}), listing)
			data = data[n:]
		case 2:
			value, n := consumeDouble(data)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			point["total_cost"] = value
			data = data[n:]
		case 3:
			value, n := consumeDouble(data)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			point["avg_quality"] = value
			data = data[n:]
		case 4:
			value, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			point["num_ecosystems"] = int(value)
			data = data[n:]
		case 5:
			value, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			point["num_hubs"] = int(value)
			data = data[n:]
		case 6:
			value, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			point["is_recommended"] = value != 0
			data = data[n:]
		default:
			skipped, err := consumeFieldValue(typ, data)
			if err != nil {
				return nil, err
			}
			data = data[skipped:]
		}
	}
	return point, nil
}

func decodeSelectedListing(data []byte) (map[string]interface{}, error) {
	listing := map[string]interface{}{
		"connection": map[string]interface{}{},
	}
	for len(data) > 0 {
		field, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return nil, protowire.ParseError(n)
		}
		data = data[n:]
		switch field {
		case 1:
			value, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			listing["id"] = int(value)
			data = data[n:]
		case 2:
			value, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			listing["requirement_id"] = int(value)
			data = data[n:]
		case 3:
			value, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			listing["device_id"] = int(value)
			data = data[n:]
		case 4:
			value, n := consumeDouble(data)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			listing["quality"] = value
			data = data[n:]
		case 5:
			rawConnection, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			connection, err := decodeConnectionInfo(rawConnection)
			if err != nil {
				return nil, err
			}
			listing["connection"].(map[string]interface{})["direct"] = connection
			data = data[n:]
		case 6:
			rawConnection, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			connection, err := decodeConnectionInfo(rawConnection)
			if err != nil {
				return nil, err
			}
			listing["connection"].(map[string]interface{})["final"] = connection
			data = data[n:]
		case 7:
			value, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			listing["extracted_listing_id"] = int(value)
			data = data[n:]
		case 8:
			value, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			listing["devices_per_listing"] = int(value)
			data = data[n:]
		case 9:
			value, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			listing["quantity"] = int(value)
			data = data[n:]
		case 10:
			value, n := consumeDouble(data)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			listing["price"] = value
			data = data[n:]
		default:
			skipped, err := consumeFieldValue(typ, data)
			if err != nil {
				return nil, err
			}
			data = data[skipped:]
		}
	}
	connection := listing["connection"].(map[string]interface{})
	if _, ok := connection["direct"]; !ok {
		connection["direct"] = nil
	}
	if _, ok := connection["final"]; !ok {
		connection["final"] = nil
	}
	return listing, nil
}

func decodeConnectionInfo(data []byte) (map[string]interface{}, error) {
	connection := map[string]interface{}{}
	for len(data) > 0 {
		field, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return nil, protowire.ParseError(n)
		}
		data = data[n:]
		switch field {
		case 1:
			value, n := protowire.ConsumeString(data)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			connection["ecosystem"] = value
			data = data[n:]
		case 2:
			value, n := protowire.ConsumeString(data)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			connection["protocol"] = value
			data = data[n:]
		case 3:
			value, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			connection["hub_solution_item_id"] = int(value)
			data = data[n:]
		default:
			skipped, err := consumeFieldValue(typ, data)
			if err != nil {
				return nil, err
			}
			data = data[skipped:]
		}
	}
	return connection, nil
}

func appendString(out []byte, fieldNumber protowire.Number, value string) []byte {
	if value == "" {
		return out
	}
	out = protowire.AppendTag(out, fieldNumber, protowire.BytesType)
	return protowire.AppendString(out, value)
}

func appendDouble(out []byte, fieldNumber protowire.Number, value float64) []byte {
	if value == 0 {
		return out
	}
	return appendDoubleValue(out, fieldNumber, value)
}

func appendDoubleValue(out []byte, fieldNumber protowire.Number, value float64) []byte {
	out = protowire.AppendTag(out, fieldNumber, protowire.Fixed64Type)
	return protowire.AppendFixed64(out, math.Float64bits(value))
}

func appendVarint(out []byte, fieldNumber protowire.Number, value uint64) []byte {
	out = protowire.AppendTag(out, fieldNumber, protowire.VarintType)
	return protowire.AppendVarint(out, value)
}

func consumeDouble(data []byte) (float64, int) {
	value, n := protowire.ConsumeFixed64(data)
	if n < 0 {
		return 0, n
	}
	return math.Float64frombits(value), n
}

func consumeFieldValue(typ protowire.Type, data []byte) (int, error) {
	var n int
	switch typ {
	case protowire.VarintType:
		_, n = protowire.ConsumeVarint(data)
	case protowire.Fixed32Type:
		_, n = protowire.ConsumeFixed32(data)
	case protowire.Fixed64Type:
		_, n = protowire.ConsumeFixed64(data)
	case protowire.BytesType:
		_, n = protowire.ConsumeBytes(data)
	case protowire.StartGroupType:
		_, n = protowire.ConsumeGroup(0, data)
	default:
		return 0, fmt.Errorf("unsupported protobuf wire type %v", typ)
	}
	if n < 0 {
		return 0, protowire.ParseError(n)
	}
	return n, nil
}

func filterOp(raw string) int {
	switch strings.ToLower(strings.TrimPrefix(raw, "OP_")) {
	case "eq":
		return opEQ
	case "neq":
		return opNEQ
	case "gt":
		return opGT
	case "gte":
		return opGTE
	case "lt":
		return opLT
	case "lte":
		return opLTE
	case "contains":
		return opContains
	case "exists":
		return opExists
	default:
		return opUnspecified
	}
}

func stringValue(input map[string]interface{}, key string) string {
	value, _ := input[key].(string)
	return value
}

func numberValue(input map[string]interface{}, key string) float64 {
	return floatValue(input[key])
}

func boolValue(input map[string]interface{}, key string) bool {
	value, _ := input[key].(bool)
	return value
}

func sliceValue(input map[string]interface{}, key string) []interface{} {
	value, _ := input[key].([]interface{})
	return value
}

func stringSliceValue(input map[string]interface{}, key string) []string {
	items := sliceValue(input, key)
	out := make([]string, 0, len(items))
	for _, item := range items {
		if value, ok := item.(string); ok {
			out = append(out, value)
		}
	}
	return out
}

func intValue(value interface{}) int {
	return int(floatValue(value))
}

func int64Value(value interface{}) int64 {
	return int64(floatValue(value))
}

func floatValue(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case int32:
		return float64(v)
	case uint64:
		return float64(v)
	case uint:
		return float64(v)
	default:
		return 0
	}
}
