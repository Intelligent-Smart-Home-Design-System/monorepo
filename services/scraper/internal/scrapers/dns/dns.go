package dns

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
)

type Scraper struct {
	client    *http.Client
	userAgent string
}

func NewScraper(timeout time.Duration, proxyURL, userAgent string) *Scraper {
	transport := &http.Transport{}
	if proxyURL != "" {
		if proxy, err := url.Parse(proxyURL); err == nil {
			transport.Proxy = http.ProxyURL(proxy)
		}
	}
	return &Scraper{
		client: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
		userAgent: userAgent,
	}
}

func (s *Scraper) Scrape(ctx context.Context, task domain.ScrapeTask) (*domain.ScrapeResult, error) {
	client := s.clientWithJar()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, task.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", s.userAgent)

	resp, err := client.Do(req)
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

	resources := []domain.Resource{{
		Name:         "html",
		URL:          task.URL,
		ResponseBody: body,
		StatusCode:   resp.StatusCode,
		Status:       resp.Status,
		Timestamp:    time.Now(),
	}}

	if isProductPage(task) {
		if buy, err := s.fetchProductBuy(ctx, client, task.URL, body); err == nil {
			resources = append(resources, *buy)
		}
		if chars, err := s.fetchCharacteristics(ctx, client, task.URL, body); err == nil {
			resources = append(resources, *chars)
		}
	}

	return &domain.ScrapeResult{Resources: resources}, nil
}

func isProductPage(task domain.ScrapeTask) bool {
	if task.PageType == domain.PageTypeListing {
		return true
	}
	return strings.Contains(task.URL, "/product/")
}

func (s *Scraper) clientWithJar() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Timeout:   s.client.Timeout,
		Transport: s.client.Transport,
		Jar:       jar,
	}
}
