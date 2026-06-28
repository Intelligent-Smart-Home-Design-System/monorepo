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

func TestBrowseParser_CategoryHub(t *testing.T) {
	p := NewCategoryParser()
	links, err := p.Parse(1, []*parser.ArchiveFile{
		{Name: "html", Data: loadFixture(t, "category_umnaa_tehnika.html")},
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(links.CategoryURLs), 10)
	assert.Empty(t, links.ListingURLs)
	assert.Contains(t, links.CategoryURLs, "https://www.dns-shop.ru/catalog/195a8a0d7cee53b3/umnye-datciki/")
}

func TestBrowseParser_CategoryLeaf(t *testing.T) {
	p := NewCategoryParser()
	links, err := p.Parse(2, []*parser.ArchiveFile{
		{Name: "html", Data: loadFixture(t, "category_leak_sensors.html")},
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(links.ListingURLs), 15)
	assert.Empty(t, links.CategoryURLs)
	assert.True(t, strings.HasPrefix(links.ListingURLs[0], "https://www.dns-shop.ru/product/"))
}

func TestBrowseParser_SearchWithProducts(t *testing.T) {
	p := NewDiscoveryParser()
	links, err := p.Parse(3, []*parser.ArchiveFile{
		{Name: "html", Data: loadFixture(t, "search_zigbee.html")},
	})
	require.NoError(t, err)
	assert.Len(t, links.ListingURLs, 24)
	assert.Empty(t, links.CategoryURLs)
}

func TestExtractBrowseLinks_FiltersServicePaths(t *testing.T) {
	html := []byte(`<html><body>
		<a href="/catalog/recipe/abc123/">recipe</a>
		<a href="/catalog/product/get-option-description/">api</a>
		<a class="subcategory__item" href="/catalog/aaaaaaaaaaaaaaaa/foo/">ok</a>
	</body></html>`)
	links, err := extractBrowseLinks(html)
	require.NoError(t, err)
	require.Len(t, links.CategoryURLs, 1)
	assert.Equal(t, "https://www.dns-shop.ru/catalog/aaaaaaaaaaaaaaaa/foo/", links.CategoryURLs[0])
}
