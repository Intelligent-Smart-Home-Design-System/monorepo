package ozon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
	"strings"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
)

const (
	actorID = "zen-studio/ozon-scraper-pro"
)

type Scraper struct {
	client   *http.Client
	apiKey   string
	maxItems int
}

func NewScraper(timeout time.Duration, proxyURL, apiKey string, maxItems int) *Scraper {
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
		apiKey:   apiKey,
		maxItems: maxItems,
	}
}

func (s *Scraper) Scrape(ctx context.Context, task domain.ScrapeTask) (*domain.ScrapeResult, error) {
    if task.PageType != domain.PageTypeDiscovery {
        return nil, fmt.Errorf("unsupported page type %s for ozon scraper", task.PageType)
    }
    const prefix = "ozon://discovery/"
    if !strings.HasPrefix(task.URL, prefix) {
        return nil, fmt.Errorf("invalid discovery URL: %s", task.URL)
    }
    query := strings.TrimPrefix(task.URL, prefix)
    return s.scrapeSearch(ctx, query)
}

func (s *Scraper) scrapeSearch(ctx context.Context, query string) (*domain.ScrapeResult, error) {
	input := map[string]interface{}{
		"queries":     []string{query},
		"maxResults":  s.maxItems,
		"skipDetails": false,
		"includeSellerDetails": true,
		"language":    "ru",
		"currency":    "RUB",
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	runURL := fmt.Sprintf("https://api.apify.com/v2/acts/%s/runs?token=%s", actorID, s.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", runURL, bytes.NewReader(inputJSON))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to start actor: %s", body)
	}

	var runResp struct {
		Data struct {
			ID        string `json:"id"`
			Status    string `json:"status"`
			DatasetID string `json:"defaultDatasetId"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&runResp); err != nil {
		return nil, err
	}

	runID := runResp.Data.ID
	datasetID := runResp.Data.DatasetID

	for {
		statusURL := fmt.Sprintf("https://api.apify.com/v2/acts/%s/runs/%s?token=%s", actorID, runID, s.apiKey)
		req, err := http.NewRequestWithContext(ctx, "GET", statusURL, nil)
		if err != nil {
			return nil, err
		}
		resp, err := s.client.Do(req)
		if err != nil {
			return nil, err
		}
		var statusResp struct {
			Data struct {
				Status string `json:"status"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		if statusResp.Data.Status == "SUCCEEDED" {
			break
		} else if statusResp.Data.Status == "FAILED" || statusResp.Data.Status == "ABORTED" {
			return nil, fmt.Errorf("actor run failed with status %s", statusResp.Data.Status)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}

	datasetURL := fmt.Sprintf("https://api.apify.com/v2/datasets/%s/items?token=%s", datasetID, s.apiKey)
	req, err = http.NewRequestWithContext(ctx, "GET", datasetURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err = s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	resource := domain.Resource{
		Name:         "apify_result.json",
		URL:          datasetURL,
		ResponseBody: body,
		StatusCode:   resp.StatusCode,
		Status:       resp.Status,
		Timestamp:    time.Now(),
	}
	return &domain.ScrapeResult{Resources: []domain.Resource{resource}}, nil
}
