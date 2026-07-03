package dns

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
)

var characteristicsURLRE = regexp.MustCompile(`id="product-card-characteristics"[^>]*data-url="([^"]+)"`)

func extractCharacteristicsPath(html []byte) string {
	match := characteristicsURLRE.FindSubmatch(html)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(string(match[1]))
}

func resolveCharacteristicsURL(pageURL, path string) (string, error) {
	base, err := url.Parse(pageURL)
	if err != nil {
		return "", err
	}
	ref, err := url.Parse(path)
	if err != nil {
		return "", err
	}
	return base.ResolveReference(ref).String(), nil
}

func (s *Scraper) fetchCharacteristics(ctx context.Context, pageURL string, html []byte) (*domain.Resource, error) {
	path := extractCharacteristicsPath(html)
	if path == "" {
		return nil, fmt.Errorf("characteristics URL not found")
	}

	charURL, err := resolveCharacteristicsURL(pageURL, path)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, charURL, nil)
	if err != nil {
		return nil, err
	}
	setNavigationHeaders(req, s.userAgent, pageURL)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("characteristics HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &domain.Resource{
		Name:         "characteristics.html",
		URL:          charURL,
		ResponseBody: body,
		StatusCode:   resp.StatusCode,
		Status:       resp.Status,
		Timestamp:    time.Now(),
	}, nil
}
