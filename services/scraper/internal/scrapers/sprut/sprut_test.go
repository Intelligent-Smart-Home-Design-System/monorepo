package sprut

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
)

type roundTripFunc func(req *http.Request) *http.Response

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

type errorTransport struct {
	err error
}

func (et errorTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, et.err
}

func TestScraper_Scrape_Success(t *testing.T) {
	testHTML, err := os.ReadFile("testdata/page1.html")
	require.NoError(t, err)

	scraper := &Scraper{
		client: &http.Client{
			Timeout: 5 * time.Second,
			Transport: roundTripFunc(func(req *http.Request) *http.Response {
				assert.Equal(t, http.MethodGet, req.Method)
				assert.Equal(t, "https://sprut.ai/test-page", req.URL.String())
				assert.Contains(t, req.Header.Get("User-Agent"), "Mozilla")

				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       io.NopCloser(bytes.NewReader(testHTML)),
					Header:     make(http.Header),
				}
			}),
		},
		userAgent: "Mozilla",
	}

	task := domain.ScrapeTask{
		Source:   Source,
		PageType: domain.PageTypeListing,
		URL:      "https://sprut.ai/test-page",
	}

	result, err := scraper.Scrape(context.Background(), task)
	require.NoError(t, err)

	require.Len(t, result.Resources, 1)
	res := result.Resources[0]

	assert.Equal(t, "html", res.Name)
	assert.Equal(t, task.URL, res.URL)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "200 OK", res.Status)
	assert.Equal(t, testHTML, res.ResponseBody)
	assert.False(t, res.Timestamp.IsZero())
}

func TestScraper_Scrape_NonOKStatus(t *testing.T) {
	scraper := &Scraper{
		client: &http.Client{
			Timeout: 5 * time.Second,
			Transport: roundTripFunc(func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Status:     "404 Not Found",
					Body:       io.NopCloser(bytes.NewReader([]byte("not found"))),
					Header:     make(http.Header),
				}
			}),
		},
	}

	task := domain.ScrapeTask{
		Source: "sprut",
		URL:    "https://sprut.ai/missing",
	}

	_, err := scraper.Scrape(context.Background(), task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status: 404 Not Found")
}

func TestScraper_Scrape_NetworkError(t *testing.T) {
	expectedErr := io.EOF
	scraper := &Scraper{
		client: &http.Client{
			Timeout:   5 * time.Second,
			Transport: errorTransport{err: expectedErr},
		},
	}

	task := domain.ScrapeTask{
		Source: "sprut",
		URL:    "https://sprut.ai/test",
	}

	_, err := scraper.Scrape(context.Background(), task)
	assert.Error(t, err)
}
