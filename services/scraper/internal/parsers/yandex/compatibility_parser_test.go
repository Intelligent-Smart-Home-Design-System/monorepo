package yandex

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parser"
)

func TestCompatibilityParser_Parse(t *testing.T) {
	htmlData, err := os.ReadFile("testdata/supported-zigbee-devices.html")
	require.NoError(t, err, "failed to read test HTML file")

	files := []*parser.ArchiveFile{
		{Name: "html", Data: htmlData},
	}

	aliases := map[string]string{
		"яндекс": "yandex",
		"aqara":  "aqara",
	}
	p := NewCompatibilityParser(aliases)

	records, err := p.Parse(123, files)
	require.NoError(t, err)
	assert.NotEmpty(t, records)

	expectedFirst := []struct {
		brand string
		model string
	}{
		{"yandex", "YNDX-00531"},
		{"yandex", "YNDX-00532"},
		{"yandex", "YNDX-00534"},
		{"yandex", "YNDX-00535"},
		{"aqara", "WS-EUK01"},
	}

	for i, exp := range expectedFirst {
		if i >= len(records) {
			break
		}
		assert.Equal(t, exp.brand, records[i].Brand, "brand mismatch at index %d", i)
		assert.Equal(t, exp.model, records[i].Model, "model mismatch at index %d", i)
	}
}

func TestCompatibilityParser_NormalizeModel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"YNDX‑00531", "YNDX-00531"},
		{"WS-EUK01", "WS-EUK01"},
		{"", ""},
	}
	for _, tt := range tests {
		result := normalizeModel(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

func TestCompatibilityParser_NormalizeBrand(t *testing.T) {
	aliases := map[string]string{
		"яндекс": "yandex",
		"aqara":  "aqara",
	}
	p := NewCompatibilityParser(aliases)

	tests := []struct {
		input    string
		expected string
	}{
		{"  Яндекс  ", "yandex"},
		{"Aqara", "aqara"},
		{"Xiaomi", "xiaomi"},
		{"Apple Home", "apple-home"},
	}
	for _, tt := range tests {
		result := p.normalizeBrand(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}
