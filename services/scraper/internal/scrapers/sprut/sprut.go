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
	client    *http.Client
	userAgent string
}

func NewScraper(timeout time.Duration, userAgent string) *Scraper {
	return &Scraper{
		client:    &http.Client{Timeout: timeout},
		userAgent: userAgent,
	}
}

func (s *Scraper) Scrape(ctx context.Context, task domain.ScrapeTask) (*domain.ScrapeResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, task.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", s.userAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
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

	return &domain.ScrapeResult{
		Resources: []domain.Resource{resource},
	}, nil
}
