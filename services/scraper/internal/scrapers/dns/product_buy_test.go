package dns

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadProbeFixture(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("testdata", "probe", "02_https_www_dns_shop_ru_product_420b8a9ab11ad0a4_datcik_moes_zigbee_flood_sensor_w.html")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return data
}

func TestExtractProductBuyRequest(t *testing.T) {
	html := loadProbeFixture(t)

	payload, csrf, err := extractProductBuyRequest(html)
	require.NoError(t, err)

	assert.NotEmpty(t, csrf)
	assert.Equal(t, "product-buy", payload.Type)
	assert.NotEmpty(t, payload.Hash)
	require.Len(t, payload.Containers, 1)
	assert.Equal(t, "5608711", payload.Containers[0].Data.ID)
	assert.Equal(t, "as-n3NQhU", payload.Containers[0].ID)
	assert.True(t, payload.Containers[0].Data.Params.IsCard)
}

func TestBuildProductBuyForm(t *testing.T) {
	html := loadProbeFixture(t)
	payload, _, err := extractProductBuyRequest(html)
	require.NoError(t, err)

	form, err := buildProductBuyForm(payload)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(form, "data="))
	assert.Contains(t, form, "product-buy")
	assert.Contains(t, form, "5608711")
}
