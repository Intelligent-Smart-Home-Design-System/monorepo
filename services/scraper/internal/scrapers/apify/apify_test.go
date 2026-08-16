package apify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
)

// fakeApify simulates just enough of the Apify API for Scraper.Scrape to
// exercise its full lifecycle: start run -> poll status -> fetch dataset.
func fakeApify(t *testing.T, statusSequence []string, datasetItems string) *httptest.Server {
	t.Helper()
	pollCount := 0
	mux := http.NewServeMux()

	mux.HandleFunc("/acts/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/runs"):
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"id":               "run123",
					"status":           "RUNNING",
					"defaultDatasetId": "dataset123",
				},
			})
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/runs/run123"):
			status := statusSequence[min(pollCount, len(statusSequence)-1)]
			pollCount++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"status": status},
			})
		default:
			http.NotFound(w, r)
		}
	})

	mux.HandleFunc("/datasets/dataset123/items", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(datasetItems))
	})

	return httptest.NewServer(mux)
}

func newTestScraper(t *testing.T, baseURL string) *Scraper {
	t.Helper()
	s := NewScraper(10*time.Second, "", "test-key", "someuser/some-actor", 213, 20)
	s.baseURL = baseURL
	s.pollInterval = 10 * time.Millisecond
	return s
}

func TestApifyScraper_Scrape_Success(t *testing.T) {
	srv := fakeApify(t, []string{"RUNNING", "RUNNING", "SUCCEEDED"}, `[{"name":"умная лампа","price":990}]`)
	defer srv.Close()

	s := newTestScraper(t, srv.URL)

	result, err := s.Scrape(context.Background(), domain.ScrapeTask{
		PageType: domain.PageTypeDiscovery,
		URL:      "умный дом",
	})
	require.NoError(t, err)
	require.Len(t, result.Resources, 1)
	assert.Equal(t, "apify_result.json", result.Resources[0].Name)
	assert.Contains(t, string(result.Resources[0].ResponseBody), "умная лампа")
	assert.Equal(t, http.StatusOK, result.Resources[0].StatusCode)
}

func TestApifyScraper_Scrape_ActorFailed(t *testing.T) {
	srv := fakeApify(t, []string{"FAILED"}, `[]`)
	defer srv.Close()

	s := newTestScraper(t, srv.URL)

	_, err := s.Scrape(context.Background(), domain.ScrapeTask{
		PageType: domain.PageTypeDiscovery,
		URL:      "умный дом",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "FAILED")
}

func TestApifyScraper_Scrape_RejectsNonDiscoveryPageType(t *testing.T) {
	s := newTestScraper(t, "http://unused.invalid")

	_, err := s.Scrape(context.Background(), domain.ScrapeTask{
		PageType: domain.PageTypeListing,
		URL:      "https://example.com",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported page type")
}

func TestApifyScraper_Scrape_ContextCancelledDuringPoll(t *testing.T) {
	srv := fakeApify(t, []string{"RUNNING", "RUNNING", "RUNNING"}, `[]`)
	defer srv.Close()

	s := newTestScraper(t, srv.URL)
	s.pollInterval = 50 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	_, err := s.Scrape(ctx, domain.ScrapeTask{
		PageType: domain.PageTypeDiscovery,
		URL:      "умный дом",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}
