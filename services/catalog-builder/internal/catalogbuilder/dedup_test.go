package catalogbuilder

import (
	"fmt"
	"testing"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/catalog-builder/internal/domain"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestGetPrimaryKey(t *testing.T) {
	listing := domain.ExtractedListing{
		Brand:    "aqara",
		Model:    ptr("WSAO-23"),
		Category: "water_leak_sensor",
	}

	key, err := getPrimaryKey(&listing)

	require.NoError(t, err)
	require.Equal(t, "water_leak_sensor:WSAO-23", key)
}

func TestGetPrimaryKeyNilModel(t *testing.T) {
	listing := domain.ExtractedListing{
		Brand:    "aqara",
		Model:    nil,
		Category: "water_leak_sensor",
	}

	_, err := getPrimaryKey(&listing)

	require.ErrorIs(t, err, errNoModel)
}

func TestGetSecondaryKey(t *testing.T) {
	tests := []struct {
		name          string
		listing       *domain.ExtractedListing
		attrs         []string
		expectedKey   string
		expectedError error
	}{
		// Success cases
		{
			name: "single string attribute",
			listing: &domain.ExtractedListing{
				Brand:    "aqara",
				Category: "water_leak_sensor",
				DeviceAttributes: map[string]any{
					"protocol": "zigbee",
				},
			},
			attrs:       []string{"protocol"},
			expectedKey: "aqara:water_leak_sensor:zigbee",
		},
		{
			name: "multiple string attributes",
			listing: &domain.ExtractedListing{
				Brand:    "sonoff",
				Category: "switch",
				DeviceAttributes: map[string]any{
					"protocol": "wifi",
					"voltage":  "110v",
				},
			},
			attrs:       []string{"protocol", "voltage"},
			expectedKey: "sonoff:switch:wifi:110v",
		},
		{
			name: "integer attribute",
			listing: &domain.ExtractedListing{
				Brand:    "tp-link",
				Category: "smart_plug",
				DeviceAttributes: map[string]any{
					"wattage": 1500,
				},
			},
			attrs:       []string{"wattage"},
			expectedKey: "tp-link:smart_plug:1500",
		},
		{
			name: "float attribute rounded to 1 decimal",
			listing: &domain.ExtractedListing{
				Brand:    "shelly",
				Category: "energy_meter",
				DeviceAttributes: map[string]any{
					"max_power": 3680.75,
				},
			},
			attrs:       []string{"max_power"},
			expectedKey: "shelly:energy_meter:3680.8", // rounded up
		},
		{
			name: "boolean attribute",
			listing: &domain.ExtractedListing{
				Brand:    "philips",
				Category: "light_bulb",
				DeviceAttributes: map[string]any{
					"dimmable": true,
				},
			},
			attrs:       []string{"dimmable"},
			expectedKey: "philips:light_bulb:true",
		},
		{
			name: "mixed attribute types",
			listing: &domain.ExtractedListing{
				Brand:    "xiaomi",
				Category: "sensor",
				DeviceAttributes: map[string]any{
					"protocol":   "bluetooth",
					"range":      50,
					"battery":    3.7,
					"waterproof": true,
				},
			},
			attrs:       []string{"protocol", "range", "battery", "waterproof"},
			expectedKey: "xiaomi:sensor:bluetooth:50:3.7:true",
		},
		{
			name: "attributes with spaces",
			listing: &domain.ExtractedListing{
				Brand:    "generic",
				Category: "controller",
				DeviceAttributes: map[string]any{
					"name": "Smart Controller v2.0",
				},
			},
			attrs:       []string{"name"},
			expectedKey: "generic:controller:Smart Controller v2.0",
		},
		{
			name: "attributes with empty string value",
			listing: &domain.ExtractedListing{
				Brand:    "test",
				Category: "device",
				DeviceAttributes: map[string]any{
					"empty_attr": "",
				},
			},
			attrs:       []string{"empty_attr"},
			expectedKey: "test:device:",
		},
		{
			name: "string array attribute",
			listing: &domain.ExtractedListing{
				Brand:    "test",
				Category: "device",
				DeviceAttributes: map[string]any{
					"attr": []any{"cc", "aa", "bb"},
				},
			},
			attrs:       []string{"attr"},
			expectedKey: "test:device:aa,bb,cc",
		},
		{
			name: "different integer types",
			listing: &domain.ExtractedListing{
				Brand:    "test",
				Category: "device",
				DeviceAttributes: map[string]any{
					"int8":   int8(-10),
					"int16":  int16(1000),
					"int32":  int32(50000),
					"int64":  int64(1000000),
					"uint":   uint(255),
					"uint8":  uint8(200),
					"uint16": uint16(60000),
					"uint32": uint32(4000000),
					"uint64": uint64(1000000000),
				},
			},
			attrs:       []string{"int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64"},
			expectedKey: "test:device:-10:1000:50000:1000000:255:200:60000:4000000:1000000000",
		},
		{
			name: "attribute is null",
			listing: &domain.ExtractedListing{
				Brand:    "aqara",
				Category: "sensor",
				DeviceAttributes: map[string]any{
					"protocol": nil,
				},
			},
			attrs:       []string{"protocol"},
			expectedKey: "aqara:sensor:null",
		},

		{
			name: "multiple attributes with one null",
			listing: &domain.ExtractedListing{
				Brand:    "test",
				Category: "device",
				DeviceAttributes: map[string]any{
					"protocol": "wifi",
					"range":    100,
					"attr":     nil,
				},
			},
			attrs:       []string{"protocol", "attr", "range"},
			expectedKey: "test:device:wifi:null:100",
		},

		// Error cases
		{
			name: "no attributes provided",
			listing: &domain.ExtractedListing{
				Brand:    "aqara",
				Category: "sensor",
				DeviceAttributes: map[string]any{
					"protocol": "zigbee",
				},
			},
			attrs:         []string{},
			expectedError: errNoAttrs,
		},
		{
			name: "attribute not found",
			listing: &domain.ExtractedListing{
				Brand:    "aqara",
				Category: "sensor",
				DeviceAttributes: map[string]any{
					"protocol": "zigbee",
				},
			},
			attrs:         []string{"nonexistent"},
			expectedError: errAttrNotFound,
		},
		{
			name: "unsupported attribute type (slice)",
			listing: &domain.ExtractedListing{
				Brand:    "test",
				Category: "device",
				DeviceAttributes: map[string]any{
					"protocol": []string{"wifi", "bluetooth"},
				},
			},
			attrs:         []string{"protocol"},
			expectedError: errUnsupportedAttr,
		},
		{
			name: "unsupported attribute type (map)",
			listing: &domain.ExtractedListing{
				Brand:    "test",
				Category: "device",
				DeviceAttributes: map[string]any{
					"settings": map[string]any{},
				},
			},
			attrs:         []string{"settings"},
			expectedError: errUnsupportedAttr,
		},
		{
			name: "multiple attributes with one missing",
			listing: &domain.ExtractedListing{
				Brand:    "test",
				Category: "device",
				DeviceAttributes: map[string]any{
					"protocol": "wifi",
					"range":    100,
				},
			},
			attrs:         []string{"protocol", "nonexistent", "range"},
			expectedError: errAttrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := getSecondaryKey(tt.listing, tt.attrs)

			if tt.expectedError != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedError)
				require.Empty(t, key)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedKey, key)
			}
		})
	}
}

func TestMergeFieldValues(t *testing.T) {
	tests := []struct {
		name     string
		values   []any
		expected any
	}{
		{"bool: all true", []any{true, true, true}, true},
		{"bool: all false", []any{false, false, false}, false},
		{"bool: majority true", []any{true, true, false}, true},
		{"bool: majority false", []any{true, false, false}, false},
		{"bool: exact half", []any{true, false}, true},
		{"bool: single true", []any{true}, true},
		{"bool: single false", []any{false}, false},
		{"float: single value", []any{float64(5)}, float64(5)},
		{"float: odd count", []any{float64(1), float64(3), float64(9)}, float64(3)},
		{"float: even count takes lower middle", []any{float64(1), float64(2), float64(3), float64(4)}, float64(2)},
		{"float: all same", []any{float64(7), float64(7), float64(7)}, float64(7)},
		{"float: outlier", []any{float64(1), float64(2), float64(999)}, float64(2)},
		{"string: single value", []any{"E27"}, "E27"},
		{"string: majority", []any{"E27", "E27", "E14", "E27", "E14", "10"}, "E27"},
		{"string: all same", []any{"E27", "E27", "E27"}, "E27"},
		{
			"string set: all have zigbee",
			[]any{
				[]any{"zigbee"},
				[]any{"zigbee"},
				[]any{"zigbee"},
			},
			[]string{"zigbee"},
		},
		{
			"string set: majority have zigbee wifi",
			[]any{
				[]any{"zigbee", "wifi"},
				[]any{"zigbee", "wifi"},
				[]any{"zigbee"},
			},
			[]string{"zigbee", "wifi"},
		},
		{
			"string set: minority value excluded",
			[]any{
				[]any{"zigbee", "bluetooth"},
				[]any{"zigbee"},
				[]any{"zigbee"},
			},
			[]string{"zigbee"},
		},
		{
			"string set: exact half included",
			[]any{
				[]any{"zigbee", "wifi"},
				[]any{"zigbee"},
			},
			[]string{"zigbee", "wifi"},
		},
		{
			"string set: complex case",
			[]any{
				[]any{"matter-over-wifi", "wifi"},
				[]any{"matter-over-wifi", "zigbee"},
				[]any{"matter-over-thread", "zigbee"},
				[]any{"zigbee", "bt"},
			},
			[]string{"matter-over-wifi", "zigbee"},
		},
		{
			"string set: lots of empty string sets",
			[]any{
				[]any{"matter-over-wifi", "wifi"},
				[]any{"matter-over-wifi", "zigbee"},
				[]any{"matter-over-thread", "zigbee"},
				[]any{"zigbee", "bt"},
				[]any{},
				[]any{},
				[]any{},
				nil,
				nil,
				nil,
			},
			[]string{"matter-over-wifi", "zigbee"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mergeFieldValues(tt.values)
			require.NoError(t, err)
			if _, isStringSetTest := tt.expected.([]string); isStringSetTest {
				require.ElementsMatch(t, tt.expected, got)
			} else {
				require.Equal(t, tt.expected, got)
			}
		})
	}
}

func TestMergeFieldValues_UnsupportedType_ReturnsError(t *testing.T) {
	_, err := mergeFieldValues([]any{map[string]any{"key": "value"}})
	require.Error(t, err)
}

func TestMergeFieldValues_Empty_ReturnsNil(t *testing.T) {
	got, err := mergeFieldValues([]any{})
	require.Nil(t, got)
	require.NoError(t, err)
}

func TestDeduplicateAttributes_AllNullFieldOmitted(t *testing.T) {
	tests := []struct {
		name     string
		listings []*domain.ExtractedListing
		expected map[string]any
	}{
		{
			name: "all null field omitted",
			listings: []*domain.ExtractedListing{
				{DeviceAttributes: map[string]any{"brightness_lm": nil}},
				{DeviceAttributes: map[string]any{"brightness_lm": nil}},
			},
			expected: map[string]any{},
		},
		{
			name: "partial null",
			listings: []*domain.ExtractedListing{
				{DeviceAttributes: map[string]any{"brightness_lm": float64(800)}},
				{DeviceAttributes: map[string]any{"brightness_lm": nil}},
				{DeviceAttributes: map[string]any{"brightness_lm": float64(900)}},
				{DeviceAttributes: map[string]any{"brightness_lm": float64(100)}},
			},
			expected: map[string]any{
				"brightness_lm": float64(800),
			},
		},
		{
			name: "field missing",
			listings: []*domain.ExtractedListing{
				{DeviceAttributes: map[string]any{"socket_type": "E27"}},
				{DeviceAttributes: map[string]any{}},
				{DeviceAttributes: map[string]any{"socket_type": "E27"}},
			},
			expected: map[string]any{
				"socket_type": "E27",
			},
		},
		{
			name: "all null field omitted",
			listings: []*domain.ExtractedListing{
				{DeviceAttributes: map[string]any{"socket_type": "E27"}},
				{DeviceAttributes: map[string]any{}},
				{DeviceAttributes: map[string]any{"socket_type": "E27"}},
			},
			expected: map[string]any{
				"socket_type": "E27",
			},
		},
		{
			name:     "empty",
			listings: nil,
			expected: nil,
		},
		{
			name: "complex",
			listings: []*domain.ExtractedListing{
				{DeviceAttributes: map[string]any{
					"name":          "yandex smart lamp 800lm",
					"socket_type":   "E27",
					"brightness_lm": float64(800),
					"protocol":      []any{"matter-over-wifi", "zigbee"},
				}},
				{DeviceAttributes: map[string]any{
					"name":          "yandex smart lamp 800 lm",
					"socket_type":   "E27",
					"brightness_lm": nil,
					"protocol":      []any{"matter-over-wifi", "zigbee", "bt"},
				}},
				{DeviceAttributes: map[string]any{
					"name":          "yandex smart lamp 800lm",
					"socket_type":   nil,
					"brightness_lm": float64(800),
					"protocol":      []any{"zigbee"},
				}},
			},
			expected: map[string]any{
				"name":          "yandex smart lamp 800lm",
				"socket_type":   "E27",
				"brightness_lm": float64(800),
				"protocol":      []string{"matter-over-wifi", "zigbee"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deduplicateAttributes(tt.listings, zerolog.Nop())
			fmt.Println(tt.name)
			fmt.Println(got)
			require.Equal(t, tt.expected, got)
		})
	}
}

func ptr(s string) *string {
	return &s
}
