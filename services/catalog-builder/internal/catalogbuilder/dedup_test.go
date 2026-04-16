package catalogbuilder

import (
	"testing"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/catalog-builder/internal/domain"
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
	require.Equal(t, "aqara:water_leak_sensor:WSAO-23", key)
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

func ptr(s string) *string {
	return &s
}
