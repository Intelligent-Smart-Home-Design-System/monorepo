package dns

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parser"
)

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("testdata", name)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return data
}

func TestDiscoveryParser_CategoryHub(t *testing.T) {
	p := NewDiscoveryParser()
	links, err := p.Parse(1, []*parser.ArchiveFile{
		{Name: "html", Data: loadFixture(t, "category_umnaa_tehnika.html")},
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(links.DiscoveryURLs), 10)
	assert.Empty(t, links.ListingURLs)
	assert.False(t, links.IsProductGrid())
}

func TestDiscoveryParser_ProductGrid(t *testing.T) {
	p := NewDiscoveryParser()
	links, err := p.Parse(2, []*parser.ArchiveFile{
		{Name: "html", Data: loadFixture(t, "category_leak_sensors.html")},
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(links.ListingURLs), 15)
	assert.True(t, links.IsProductGrid())
	assert.True(t, strings.HasPrefix(links.ListingURLs[0], "https://www.dns-shop.ru/product/"))
}

func TestCategoryParser_LeafWithListings(t *testing.T) {
	p := NewCategoryParser()
	links, err := p.Parse(2, []*parser.ArchiveFile{
		{Name: "html", Data: loadFixture(t, "category_leak_sensors.html")},
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(links.ListingURLs), 15)
	assert.Empty(t, links.DiscoveryURLs)
}

func TestDiscoveryParser_SearchWithProducts(t *testing.T) {
	links, err := extractBrowseLinks(loadFixture(t, "search_zigbee.html"), "https://www.dns-shop.ru/search/?q=zigbee")
	require.NoError(t, err)
	assert.Len(t, links.ListingURLs, 24)
	assert.True(t, links.IsProductGrid())
	assert.GreaterOrEqual(t, len(links.PaginationURLs), 1)
	assert.Contains(t, links.PaginationURLs[0], "page=2")
}

func TestExtractBrowseLinks_FiltersServicePaths(t *testing.T) {
	html := []byte(`<html><body>
		<a href="/catalog/recipe/abc123/">recipe</a>
		<a href="/catalog/product/get-option-description/">api</a>
		<a class="subcategory__item" href="/catalog/aaaaaaaaaaaaaaaa/foo/">ok</a>
	</body></html>`)
	links, err := extractBrowseLinks(html, "")
	require.NoError(t, err)
	require.Len(t, links.DiscoveryURLs, 1)
	assert.Equal(t, "https://www.dns-shop.ru/catalog/aaaaaaaaaaaaaaaa/foo/", links.DiscoveryURLs[0])
}
