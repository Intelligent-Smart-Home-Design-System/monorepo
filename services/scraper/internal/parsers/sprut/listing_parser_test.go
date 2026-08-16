package sprut

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parser"
)

func TestListingParser_Parse(t *testing.T) {
	html, err := os.ReadFile("testdata/item_aqara_t1.html")
	require.NoError(t, err)

	files := []*parser.ArchiveFile{
		{Name: "html", Data: html},
	}

	p := NewListingParser(map[string]string{"aqara": "aqara"})
	result, err := p.Parse(42, files)
	require.NoError(t, err)

	assert.True(t, result.HasSmartHomeMarkers)
	assert.Equal(t, "Temperature and Humidity Sensor T1", result.Name)
	assert.Equal(t, "aqara", result.Brand)
	require.NotNil(t, result.ModelNumber)
	assert.Equal(t, "WSDCGQ12LM", *result.ModelNumber)
	require.NotNil(t, result.Category)
	assert.Equal(t, "Датчики", *result.Category)
	assert.True(t, result.InStock)
	assert.Nil(t, result.Price)
	assert.NotContains(t, result.Text, "Модель:")
	assert.NotContains(t, result.Text, "Тип устройства:")
	assert.Contains(t, result.Text, "Протокол: Zigbee")
	assert.Contains(t, result.Text, "Характеристики: Версия ZigBee: 3.0")
	assert.Contains(t, result.Text, "Служебные: SH model: lumi.sensor_ht.agl02")
	assert.Equal(t, "https://api.sprut.ai/static/media/cache/example/400x400x_image.png", result.ImageURL)
	assert.Equal(t, float64(5), result.Rating)
	assert.Equal(t, 2, result.ReviewCount)
	assert.NotEmpty(t, result.ContentHash)
	assert.Equal(t, ExtractorVersion, result.ExtractorVer)
}

func TestClampRating(t *testing.T) {
	assert.Equal(t, 5.0, clampRating(50))
	assert.Equal(t, 5.0, clampRating(100))
	assert.Equal(t, 4.5, clampRating(4.5))
}

func TestListingParser_Parse_MissingTitle(t *testing.T) {
	files := []*parser.ArchiveFile{
		{Name: "html", Data: []byte("<html><body></body></html>")},
	}
	p := NewListingParser(nil)
	_, err := p.Parse(1, files)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "product title not found")
}
