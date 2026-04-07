package printer

import (
	"net/http"
	"testing"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestScrape(t *testing.T) {
	printer := NewPrinterScraper()
	task := domain.ScrapeTask{
		Source:   "test",
		PageType: "testing_page_type",
		URL:      "testing://test.url",
	}

	result, err := printer.Scrape(t.Context(), task)
	resources := make(map[string]domain.Resource)
	for _, resource := range result.Resources {
		resources[resource.Name] = resource
	}

	require.NoError(t, err)
	require.Equal(t, resources, map[string]domain.Resource{
		"url": {
			Name:         "url",
			StatusCode:   http.StatusOK,
			ResponseBody: []byte("testing://test.url"),
		},
		"page_type": {
			Name:         "page_type",
			StatusCode:   http.StatusOK,
			ResponseBody: []byte("testing_page_type"),
		},
	})
}
