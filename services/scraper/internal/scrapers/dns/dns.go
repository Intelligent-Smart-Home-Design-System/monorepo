package dns

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
)

type Scraper struct {
	client    *http.Client
	userAgent string
	mu        sync.Mutex
	warmedUp  bool
}

func NewScraper(timeout time.Duration, proxyURL, userAgent string) *Scraper {
	return &Scraper{
		client:    newScraperClient(timeout, proxyURL),
		userAgent: userAgent,
	}
}

func (s *Scraper) Scrape(ctx context.Context, task domain.ScrapeTask) (*domain.ScrapeResult, error) {
	body, statusCode, status, err := s.getHTML(ctx, task.URL)
	if err != nil {
		return nil, err
	}

	resources := []domain.Resource{{
		Name:         "html",
		URL:          task.URL,
		ResponseBody: body,
		StatusCode:   statusCode,
		Status:       status,
		Timestamp:    time.Now(),
	}}

	if isProductPage(task) {
		if buy, err := s.fetchProductBuy(ctx, task.URL, body); err == nil {
			resources = append(resources, *buy)
		}
		if chars, err := s.fetchCharacteristics(ctx, task.URL, body); err == nil {
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
