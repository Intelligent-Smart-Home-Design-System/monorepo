package sprut

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
)

type Scraper struct {
	client *http.Client
}

func NewScraper(timeout time.Duration) *Scraper {
    return &Scraper{
        client: &http.Client{Timeout: timeout},
    }
}

func (s *Scraper) Scrape(ctx context.Context, task domain.ScrapeTask) (domain.ScrapeResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, task.URL, nil)
	if err != nil {
		return domain.ScrapeResult{}, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := s.client.Do(req)
	if err != nil {
		return domain.ScrapeResult{}, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return domain.ScrapeResult{}, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.ScrapeResult{}, fmt.Errorf("read body: %w", err)
	}

	resource := domain.Resource{
		Name:            "html",
		URL:             task.URL,
		RequestHeaders:  req.Header,
		StatusCode:      resp.StatusCode,
		Status:          resp.Status,
		ResponseHeaders: resp.Header,
		ResponseBody:    body,
		Timestamp:       time.Now(),
	}

	return domain.ScrapeResult{
		Resources: []domain.Resource{resource},
	}, nil
}
