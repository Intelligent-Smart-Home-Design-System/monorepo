package dns

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

const dnsOrigin = "https://www.dns-shop.ru/"

func newScraperClient(timeout time.Duration, proxyURL string) *http.Client {
	transport := &http.Transport{}
	if proxyURL != "" {
		if proxy, err := url.Parse(proxyURL); err == nil {
			transport.Proxy = http.ProxyURL(proxy)
		}
	}
	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		Jar:       jar,
	}
}

func setNavigationHeaders(req *http.Request, userAgent, referer string) {
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-User", "?1")
	if referer != "" {
		req.Header.Set("Referer", referer)
		req.Header.Set("Sec-Fetch-Site", "same-origin")
	} else {
		req.Header.Set("Sec-Fetch-Site", "none")
	}
}

func setXHRHeaders(req *http.Request, userAgent, referer string) {
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
}

func (s *Scraper) resetWarmup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.warmedUp = false
}

func (s *Scraper) warmup(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.warmedUp {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dnsOrigin, nil)
	if err != nil {
		return err
	}
	setNavigationHeaders(req, s.userAgent, "")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("dns warmup: %w", err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dns warmup: HTTP %d", resp.StatusCode)
	}
	s.warmedUp = true
	return nil
}

func (s *Scraper) getHTML(ctx context.Context, pageURL string) ([]byte, int, string, error) {
	referer := dnsOrigin
	if u, err := url.Parse(pageURL); err == nil && u.Host != "" {
		referer = fmt.Sprintf("%s://%s/", u.Scheme, u.Host)
	}

	var lastStatus int
	var lastErr error

	for attempt := 0; attempt < 2; attempt++ {
		if attempt > 0 {
			s.resetWarmup()
		}
		if err := s.warmup(ctx); err != nil {
			lastErr = err
			continue
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
		if err != nil {
			return nil, 0, "", err
		}
		setNavigationHeaders(req, s.userAgent, referer)

		resp, err := s.client.Do(req)
		if err != nil {
			return nil, 0, "", err
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, resp.StatusCode, resp.Status, readErr
		}

		lastStatus = resp.StatusCode
		if resp.StatusCode == http.StatusOK {
			return body, resp.StatusCode, resp.Status, nil
		}

		lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
		if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusForbidden {
			return body, resp.StatusCode, resp.Status, lastErr
		}
	}

	if lastErr != nil {
		return nil, lastStatus, "", lastErr
	}
	return nil, lastStatus, "", fmt.Errorf("HTTP %d", lastStatus)
}
