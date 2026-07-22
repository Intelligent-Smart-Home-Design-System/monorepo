package dns

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/go-rod/rod"
	"github.com/rs/zerolog"
)

type Scraper struct {
	log          zerolog.Logger
	client       *http.Client
	proxyURL     string
	userAgent    string
	forceBrowser bool
	mu           sync.Mutex
	warmedUp     bool
	browserMu    sync.Mutex
	browser      *rod.Browser
	browserPage  *rod.Page
	browserMode  bool
}

func NewScraper(log zerolog.Logger, timeout time.Duration, proxyURL, userAgent string, browserUserMode *bool) *Scraper {
	client, err := newScraperClient(timeout, proxyURL)
	if err != nil {
		log.Warn().Err(err).Str("proxy", proxyURL).Msg("dns: invalid proxy URL, requests will be direct")
		client, _ = newScraperClient(timeout, "")
		proxyURL = ""
	}
	return &Scraper{
		log:          log.With().Str("source", "dns").Logger(),
		client:       client,
		proxyURL:     proxyURL,
		userAgent:    userAgent,
		forceBrowser: defaultBrowserUserMode(browserUserMode),
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
