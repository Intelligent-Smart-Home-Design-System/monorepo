package yandex

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"golang.org/x/time/rate"
)

type Scraper struct {
	client  *http.Client
	limiter *rate.Limiter
}

func NewScraper(timeout time.Duration, proxyURL string, rps float64) *Scraper {
	transport := &http.Transport{}
	if proxyURL != "" {
		if proxy, err := url.Parse(proxyURL); err == nil {
			transport.Proxy = http.ProxyURL(proxy)
		}
	}
	limiter := rate.NewLimiter(rate.Limit(rps), 1)
	return &Scraper{
		client: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
		limiter: limiter,
	}
}

func (s *Scraper) Scrape(ctx context.Context, task domain.ScrapeTask) (*domain.ScrapeResult, error) {
	if err := s.limiter.Wait(ctx); err != nil {
		return nil, err
	}

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
