package catalogbuilder

import (
	"os"
	"testing"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/catalog-builder/internal/domain"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func defaultCfg() BuilderConfig {
	return BuilderConfig{
		IdentifyingAttributes: map[string][]string{
			"smart_lamp": {"socket_type", "wattage_w"},
		},
		CloudEcosystems:  []string{"yandex", "sber", "vk"},
		MatterEcosystems: []string{"apple", "google"},
		MatterProtocols:  []string{"matter-over-wifi", "matter-over-thread"},
	}
}

func newBuilder(t *testing.T) *Builder {
	t.Helper()
	b, err := NewBuilder(defaultCfg(), zerolog.Nop())
	require.NoError(t, err)
	return b
}

func newBuilderWithSchema(t *testing.T, schemaJSON string, strict bool) *Builder {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "taxonomy_schema*.json")
	require.NoError(t, err)
	_, err = f.WriteString(schemaJSON)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	cfg := defaultCfg()
	cfg.TaxonomySchemaPath = f.Name()
	cfg.StrictSchema = strict

	b, err := NewBuilder(cfg, zerolog.Nop())
	require.NoError(t, err)
	return b
}

func TestBuild_SameModelMerged(t *testing.T) {
	listings := []*domain.ExtractedListing{
		{
			Id: 1, Brand: "yeelight", Model: ptr("YLDP010"), Category: "smart_lamp",
			TaxonomyVersion: "v1",
			DeviceAttributes: map[string]any{
				"ecosystem":     []any{"yandex"},
				"protocol":      []any{"zigbee"},
				"socket_type":   "E27",
				"brightness_lm": float64(800),
			},
		},
		{
			Id: 2, Brand: "yeelight", Model: ptr("YLDP010"), Category: "smart_lamp",
			TaxonomyVersion: "v1",
			DeviceAttributes: map[string]any{
				"ecosystem":     []any{"yandex"},
				"protocol":      []any{"zigbee"},
				"socket_type":   nil,
				"brightness_lm": nil,
			},
		},
	}
	catalog := newBuilder(t).Build(listings, nil)
	require.Equal(t, 1, len(catalog.Devices))
	d := catalog.Devices[0]
	assert.Equal(t, "yeelight", d.Brand)
	assert.Equal(t, ptr("YLDP010"), d.Model)
	assert.Equal(t, "smart_lamp", d.Category)
	assert.Equal(t, "v1", d.TaxonomyVersion)
	assert.Equal(t, map[string]any{
		"ecosystem":     []string{"yandex"},
		"protocol":      []string{"zigbee"},
		"socket_type":   "E27",
		"brightness_lm": float64(800),
	}, d.DeviceAttributes)
}

func TestBuild_DifferentModelNotMerged(t *testing.T) {
	listings := []*domain.ExtractedListing{
		{
			Id: 1, Brand: "yeelight", Model: ptr("YLDP010"), Category: "smart_lamp",
			TaxonomyVersion:  "v1",
			DeviceAttributes: map[string]any{"ecosystem": []any{"yandex"}, "protocol": []any{"wifi"}, "socket_type": "E27"},
		},
		{
			Id: 2, Brand: "yeelight", Model: ptr("YLDP011"), Category: "smart_lamp",
			TaxonomyVersion:  "v1",
			DeviceAttributes: map[string]any{"ecosystem": []any{"yandex"}, "protocol": []any{"wifi"}, "socket_type": "E27"},
		},
	}
	catalog := newBuilder(t).Build(listings, nil)
	require.Equal(t, 2, len(catalog.Devices))
}

func TestBuild_NoPrimaryOrSecondaryKey(t *testing.T) {
	listings := []*domain.ExtractedListing{
		{
			Id: 1, Brand: "tuya", Model: nil, Category: "smart_lamp",
			TaxonomyVersion: "v1",
			DeviceAttributes: map[string]any{
				"ecosystem": []any{"tuya"}, "protocol": []any{"wifi"},
				"socket_type": "E27",
			},
		},
		{
			Id: 2, Brand: "tuya", Model: nil, Category: "smart_lamp",
			TaxonomyVersion: "v1",
			DeviceAttributes: map[string]any{
				"ecosystem": []any{"tuya"}, "protocol": []any{"wifi"},
				"socket_type": "E27",
			},
		},
	}
	catalog := newBuilder(t).Build(listings, nil)
	require.Equal(t, 2, len(catalog.Devices))
	for _, d := range catalog.Devices {
		assert.Nil(t, d.Model)
		assert.Equal(t, "tuya", d.Brand)
		assert.Equal(t, map[string]any{
			"ecosystem":   []string{"tuya"},
			"protocol":    []string{"wifi"},
			"socket_type": "E27",
		}, d.DeviceAttributes)
	}
}

func TestBuild_NoModelFallsBackToSecondaryKey(t *testing.T) {
	listings := []*domain.ExtractedListing{
		{
			Id: 1, Brand: "tuya", Model: nil, Category: "smart_lamp",
			TaxonomyVersion: "v1",
			DeviceAttributes: map[string]any{
				"ecosystem": []any{"tuya"}, "protocol": []any{"wifi"},
				"socket_type": "E27", "wattage_w": float64(9),
			},
		},
		{
			Id: 2, Brand: "tuya", Model: nil, Category: "smart_lamp",
			TaxonomyVersion: "v1",
			DeviceAttributes: map[string]any{
				"ecosystem": []any{"tuya"}, "protocol": []any{"wifi"},
				"socket_type": "E27", "wattage_w": float64(9),
			},
		},
	}
	catalog := newBuilder(t).Build(listings, nil)
	require.Equal(t, 1, len(catalog.Devices))
	d := catalog.Devices[0]
	assert.Nil(t, d.Model)
	assert.Equal(t, "tuya", d.Brand)
	assert.Equal(t, map[string]any{
		"ecosystem":   []string{"tuya"},
		"protocol":    []string{"wifi"},
		"socket_type": "E27",
		"wattage_w":   float64(9),
	}, d.DeviceAttributes)
}

func TestBuild_DirectCompatFromScrapedRecords(t *testing.T) {
	listings := []*domain.ExtractedListing{
		{
			Id: 1, Brand: "aqara", Model: ptr("SJCGQ12LM"), Category: "water_leak_sensor",
			TaxonomyVersion: "v1",
			DeviceAttributes: map[string]any{
				"ecosystem": []any{"aqara"}, "protocol": []any{"zigbee", "matter-over-thread"},
			},
		},
	}
	compat := []*domain.ScrapedDirectCompatibility{
		{Brand: "aqara", Model: "SJCGQ12LM", Ecosystem: "yandex", Protocol: "zigbee"},
	}
	catalog := newBuilder(t).Build(listings, compat)
	require.Equal(t, 1, len(catalog.Devices))
	d := catalog.Devices[0]
	require.ElementsMatch(t, []*domain.DirectCompatibility{
		{Ecosystem: "aqara", Protocol: "zigbee"},
		{Ecosystem: "aqara", Protocol: "matter-over-thread"},
		{Ecosystem: "yandex", Protocol: "zigbee"},
	}, d.DirectCompatibility)
}

func TestBuild_CloudEcosystemGetsBridgeCompat(t *testing.T) {
	listings := []*domain.ExtractedListing{
		{
			Id: 1, Brand: "tuya", Model: ptr("TS0001"), Category: "smart_plug",
			TaxonomyVersion: "v1",
			DeviceAttributes: map[string]any{
				"ecosystem": []any{"tuya", "yandex"}, "protocol": []any{"zigbee"},
			},
		},
	}
	catalog := newBuilder(t).Build(listings, nil)
	require.Equal(t, 1, len(catalog.Devices))
	d := catalog.Devices[0]
	require.ElementsMatch(t, []*domain.DirectCompatibility{
		{Ecosystem: "tuya", Protocol: "zigbee"},
	}, d.DirectCompatibility)
	require.ElementsMatch(t, []*domain.BridgeCompatibility{
		{SourceEcosystem: "tuya", TargetEcosystem: "yandex", Protocol: "cloud"},
	}, d.BridgeCompatibility)
}

func TestBuild_MatterEcosystemGetsDirectCompatIfMatterProtocol(t *testing.T) {
	listings := []*domain.ExtractedListing{
		{
			Id: 1, Brand: "aqara", Model: ptr("T1"), Category: "smart_lamp",
			TaxonomyVersion: "v1",
			DeviceAttributes: map[string]any{
				"ecosystem": []any{"aqara", "apple"}, "protocol": []any{"matter-over-wifi", "matter-over-thread"},
			},
		},
	}
	catalog := newBuilder(t).Build(listings, nil)
	require.Equal(t, 1, len(catalog.Devices))
	d := catalog.Devices[0]
	require.ElementsMatch(t, []*domain.DirectCompatibility{
		{Ecosystem: "aqara", Protocol: "matter-over-wifi"},
		{Ecosystem: "aqara", Protocol: "matter-over-thread"},
		{Ecosystem: "apple", Protocol: "matter-over-wifi"},
		{Ecosystem: "apple", Protocol: "matter-over-thread"},
	}, d.DirectCompatibility)
}

func TestBuild_MatterEcosystemGetsBridgeCompatIfNoMatter(t *testing.T) {
	listings := []*domain.ExtractedListing{
		{
			Id: 1, Brand: "tuya", Model: ptr("TS0001"), Category: "smart_plug",
			TaxonomyVersion: "v1",
			DeviceAttributes: map[string]any{
				"ecosystem": []any{"tuya", "google"}, "protocol": []any{"zigbee"},
			},
		},
	}
	catalog := newBuilder(t).Build(listings, nil)
	require.Equal(t, 1, len(catalog.Devices))
	d := catalog.Devices[0]
	require.ElementsMatch(t, []*domain.DirectCompatibility{
		{Ecosystem: "tuya", Protocol: "zigbee"},
	}, d.DirectCompatibility)
	require.ElementsMatch(t, []*domain.BridgeCompatibility{
		{SourceEcosystem: "tuya", TargetEcosystem: "google", Protocol: "cloud"},
	}, d.BridgeCompatibility)
}

const minimalSchema = `{
	"smart_lamp": {
		"schema": {
			"$schema": "http://json-schema.org/draft-07/schema#",
			"type": "object",
			"required": ["socket_type"],
			"properties": {
				"socket_type": {"type": "string"}
			}
		}
	}
}`

func TestBuild_StrictSchema_ExcludesInvalidDevice(t *testing.T) {
	b := newBuilderWithSchema(t, minimalSchema, true)

	listings := []*domain.ExtractedListing{
		{
			Id: 1, Brand: "tuya", Model: ptr("TS001"), Category: "smart_lamp",
			TaxonomyVersion: "v1",
			DeviceAttributes: map[string]any{
				"ecosystem": []any{"tuya"}, "protocol": []any{"wifi"},
				// socket_type missing — required by schema
			},
		},
	}

	catalog := b.Build(listings, nil)
	assert.Empty(t, catalog.Devices)
}

func TestBuild_NonStrictSchema_IncludesInvalidDeviceWithWarning(t *testing.T) {
	b := newBuilderWithSchema(t, minimalSchema, false)

	listings := []*domain.ExtractedListing{
		{
			Id: 1, Brand: "tuya", Model: ptr("TS001"), Category: "smart_lamp",
			TaxonomyVersion: "v1",
			DeviceAttributes: map[string]any{
				"ecosystem": []any{"tuya"}, "protocol": []any{"wifi"},
				// socket_type missing — warns but device is still included
			},
		},
	}

	catalog := b.Build(listings, nil)
	require.Len(t, catalog.Devices, 1)
	assert.Equal(t, "tuya", catalog.Devices[0].Brand)
}

func TestBuild_SchemaValidDevice_Passes(t *testing.T) {
	b := newBuilderWithSchema(t, minimalSchema, true)

	listings := []*domain.ExtractedListing{
		{
			Id: 1, Brand: "yeelight", Model: ptr("YLDP010"), Category: "smart_lamp",
			TaxonomyVersion: "v1",
			DeviceAttributes: map[string]any{
				"ecosystem":   []any{"yandex"},
				"protocol":    []any{"zigbee"},
				"socket_type": "E27",
			},
		},
	}

	catalog := b.Build(listings, nil)
	require.Len(t, catalog.Devices, 1)
}

func TestBuild_NoSchemaForCategory_AlwaysPasses(t *testing.T) {
	// water_leak_sensor has no schema in minimalSchema
	b := newBuilderWithSchema(t, minimalSchema, true)

	listings := []*domain.ExtractedListing{
		{
			Id: 1, Brand: "aqara", Model: ptr("SJCGQ12LM"), Category: "water_leak_sensor",
			TaxonomyVersion: "v1",
			DeviceAttributes: map[string]any{
				"ecosystem": []any{"aqara"}, "protocol": []any{"zigbee"},
			},
		},
	}

	catalog := b.Build(listings, nil)
	require.Len(t, catalog.Devices, 1)
}
