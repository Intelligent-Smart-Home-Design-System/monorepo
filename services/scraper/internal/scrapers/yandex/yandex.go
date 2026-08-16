package yandex

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/netproxy"
)

type Scraper struct {
	client *http.Client
}

func NewScraper(timeout time.Duration, proxyURL string, rps float64) *Scraper {
	client, err := netproxy.NewHTTPClient(timeout, proxyURL)
	if err != nil {
		client = &http.Client{Timeout: timeout}
	}
	return &Scraper{
		client: client,
	}
}

func (s *Scraper) Scrape(ctx context.Context, task domain.ScrapeTask) (*domain.ScrapeResult, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", task.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; YandexScraper/1.0)")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	resource := domain.Resource{
		Name:         "html",
		URL:          task.URL,
		ResponseBody: body,
		StatusCode:   resp.StatusCode,
		Status:       resp.Status,
		Timestamp:    time.Now(),
	}

	return &domain.ScrapeResult{
		Resources: []domain.Resource{resource},
	}, nil
}
