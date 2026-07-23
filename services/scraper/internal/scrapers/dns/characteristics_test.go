package dns

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractCharacteristicsPath(t *testing.T) {
	html := loadProbeFixture(t)
	path := extractCharacteristicsPath(html)
	require.NotEmpty(t, path)
	assert.Contains(t, path, "/product/characteristics/")
}

func TestResolveCharacteristicsURL(t *testing.T) {
	pageURL := "https://www.dns-shop.ru/product/420b8a9ab11ad0a4/datcik-moes-zigbee-flood-sensor-water-leakage-detector/"
	path := "/product/characteristics/420b8a9ab11ad0a4/datcik-moes-zigbee-flood-sensor-water-leakage-detector/"

	resolved, err := resolveCharacteristicsURL(pageURL, path)
	require.NoError(t, err)
	assert.Equal(t, "https://www.dns-shop.ru/product/characteristics/420b8a9ab11ad0a4/datcik-moes-zigbee-flood-sensor-water-leakage-detector/", resolved)
}
