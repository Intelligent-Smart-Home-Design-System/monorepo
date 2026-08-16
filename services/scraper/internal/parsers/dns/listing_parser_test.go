package dns

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parser"
)

func loadParserFixture(t *testing.T, parts ...string) []byte {
	t.Helper()
	path := filepath.Join(append([]string{"testdata"}, parts...)...)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return data
}

func TestListingParser_Parse(t *testing.T) {
	p := NewListingParser(map[string]string{"moes": "moes"}, []string{"Zigbee", "умный дом"})
	result, err := p.Parse(42, []*parser.ArchiveFile{
		{Name: "html", Data: loadParserFixture(t, "product_moes.html")},
		{Name: "characteristics.html", Data: loadParserFixture(t, "product_moes_characteristics.html")},
		{Name: "product-buy.json", Data: loadParserFixture(t, "product_buy_moes.json")},
	})
	require.NoError(t, err)

	assert.True(t, result.HasSmartHomeMarkers)
	assert.Contains(t, result.Name, "MOES")
	assert.Equal(t, "moes", result.Brand)
	require.NotNil(t, result.Price)
	assert.Equal(t, 2499, *result.Price)
	require.NotNil(t, result.Currency)
	assert.Equal(t, "RUB", *result.Currency)
	require.NotNil(t, result.ModelNumber)
	assert.Equal(t, "MOES ZigBee Flood Sensor Water Leakage Detector", *result.ModelNumber)
	require.NotNil(t, result.Category)
	assert.Equal(t, "датчик", *result.Category)
	assert.Equal(t, 11, result.ReviewCount)
	assert.Equal(t, 4.45, result.Rating)
	assert.Contains(t, result.Text, "Zigbee")
	assert.Contains(t, result.Text, "Датчик протечки MOES")
	assert.Contains(t, result.Text, "Модель: MOES ZigBee Flood Sensor Water Leakage Detector")
	assert.NotEmpty(t, result.ImageURL)
	assert.NotEmpty(t, result.ContentHash)
	assert.Equal(t, ExtractorVersion, result.ExtractorVer)
}

func TestListingParser_SkipsWithoutSmartHomeMarkers(t *testing.T) {
	p := NewListingParser(nil, []string{"Zigbee"})
	result, err := p.Parse(1, []*parser.ArchiveFile{
		{Name: "html", Data: []byte(`<html><body><h1>USB кабель</h1></body></html>`)},
	})
	require.NoError(t, err)
	assert.False(t, result.HasSmartHomeMarkers)
}

func TestListingParser_FallsBackToProductCodeWithoutCharacteristics(t *testing.T) {
	p := NewListingParser(nil, []string{"Zigbee"})
	result, err := p.Parse(1, []*parser.ArchiveFile{
		{Name: "html", Data: loadParserFixture(t, "product_moes.html")},
	})
	require.NoError(t, err)
	require.NotNil(t, result.ModelNumber)
	assert.Equal(t, "5608711", *result.ModelNumber)
}
