package wildberries

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
)

type Scraper struct {
	client      *http.Client
	userAgent   string
	basketCache map[int]int
	mu          sync.RWMutex
}

func NewScraper(timeout time.Duration, userAgent string) *Scraper {
	return &Scraper{
		client: &http.Client{Timeout: timeout},
		userAgent: userAgent,
		basketCache: make(map[int]int),
	}
}

func (s *Scraper) Scrape(ctx context.Context, task domain.ScrapeTask) (*domain.ScrapeResult, error) {
	nmID, err := extractNmID(task.URL)
	if err != nil {
		return nil, fmt.Errorf("extract nm_id: %w", err)
	}

	detailURL := fmt.Sprintf(
		"https://www.wildberries.ru/__internal/u-card/cards/v4/detail?appType=1&curr=rub&dest=-1257786&spp=30&hide_vflags=4294967296&ab_testing=false&lang=ru&nm=%d",
		nmID,
	)
	detailBody, err := s.fetchJSON(ctx, detailURL)
	if err != nil {
		return nil, fmt.Errorf("fetch detail.json: %w", err)
	}
	var detailResp WBDetailResponse
	if err := json.Unmarshal(detailBody, &detailResp); err != nil {
		return nil, fmt.Errorf("parse detail.json: %w", err)
	}

	cardURL, err := s.getCardURL(ctx, nmID)
	if err != nil {
		return nil, fmt.Errorf("get card.json URL: %w", err)
	}
	cardBody, err := s.fetchJSON(ctx, cardURL)
	if err != nil {
		return nil, fmt.Errorf("fetch card.json: %w", err)
	}
	var cardResp WBCardResponse
	if err := json.Unmarshal(cardBody, &cardResp); err != nil {
		return nil, fmt.Errorf("parse card.json: %w", err)
	}

	resources := []domain.Resource{
		{
			Name:         "detail.json",
			URL:          detailURL,
			ResponseBody: detailBody,
			StatusCode:   http.StatusOK,
			Status:       "200 OK",
			Timestamp:    time.Now(),
		},
		{
			Name:         "card.json",
			URL:          cardURL,
			ResponseBody: cardBody,
			StatusCode:   http.StatusOK,
			Status:       "200 OK",
			Timestamp:    time.Now(),
		},
	}
	return &domain.ScrapeResult{Resources: resources}, nil
}

func (s *Scraper) fetchJSON(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", s.userAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func extractNmID(url string) (int, error) {
	parts := strings.Split(url, "/catalog/")
	if len(parts) < 2 {
		return 0, fmt.Errorf("catalog not found in url")
	}
	rest := parts[1]
	end := strings.IndexAny(rest, "/?")
	if end == -1 {
		end = len(rest)
	}
	idStr := rest[:end]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, fmt.Errorf("invalid nm_id: %s", idStr)
	}
	return id, nil
}

func (s *Scraper) getCardURL(ctx context.Context, nmID int) (string, error) {
	s.mu.RLock()
	basket, ok := s.basketCache[nmID]
	s.mu.RUnlock()
	if ok {
		return buildCardURL(basket, nmID), nil
	}

	for basket := 1; basket <= 41; basket++ {
		testURL := buildCardURL(basket, nmID)
		if s.urlExists(ctx, testURL) {
			s.mu.Lock()
			s.basketCache[nmID] = basket
			s.mu.Unlock()
			return testURL, nil
		}
	}
	return "", fmt.Errorf("no working basket for nm_id %d", nmID)
}

func buildCardURL(basket, nmID int) string {
	vol := nmID / 1000
	part := nmID / 100
	return fmt.Sprintf("https://basket-%d.wbbasket.ru/vol%d/part%d/%d/info/ru/card.json", basket, vol, part, nmID)
}

func (s *Scraper) urlExists(ctx context.Context, url string) bool {
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", s.userAgent)
	resp, err := s.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
