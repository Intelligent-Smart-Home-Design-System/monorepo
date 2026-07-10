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
	if s.warmedUp {
		s.log.Info().Msg("dns warmup: reset")
	}
	s.warmedUp = false
}

func (s *Scraper) warmupHTTP(ctx context.Context) error {
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
	return nil
}

func (s *Scraper) warmup(ctx context.Context) error {
	s.mu.Lock()
	if s.warmedUp {
		s.mu.Unlock()
		s.log.Debug().Bool("browser_mode", s.browserMode).Msg("dns warmup: skipped (already done)")
		return nil
	}
	s.mu.Unlock()

	if s.browserMode {
		if err := s.activateBrowser(ctx); err != nil {
			return err
		}
		s.mu.Lock()
		s.warmedUp = true
		s.mu.Unlock()
		return nil
	}

	s.log.Info().Str("url", dnsOrigin).Msg("dns warmup: requesting homepage (HTTP)")

	httpErr := s.warmupHTTP(ctx)
	if httpErr == nil {
		s.mu.Lock()
		s.warmedUp = true
		s.mu.Unlock()
		s.log.Info().
			Int("status", http.StatusOK).
			Int("cookies", s.cookieCount()).
			Str("mode", "http").
			Msg("dns warmup: ok")
		return nil
	} else if !isAuthBlockedStatus(httpErr) {
		s.log.Error().Err(httpErr).Str("url", dnsOrigin).Msg("dns warmup: failed")
		return httpErr
	}

	s.log.Warn().Err(httpErr).Msg("dns warmup: HTTP blocked by Qrator, switching to headless browser")
	if err := s.activateBrowser(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	s.warmedUp = true
	s.mu.Unlock()
	s.log.Info().Str("mode", "browser").Msg("dns warmup: ok")
	return nil
}

func isAuthBlockedStatus(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return containsHTTPStatus(msg, http.StatusUnauthorized) || containsHTTPStatus(msg, http.StatusForbidden)
}

func containsHTTPStatus(msg string, code int) bool {
	return msg == fmt.Sprintf("dns warmup: HTTP %d", code) ||
		msg == fmt.Sprintf("HTTP %d", code)
}

func isAuthBlocked(status int) bool {
	return status == http.StatusUnauthorized || status == http.StatusForbidden
}

func (s *Scraper) getHTML(ctx context.Context, pageURL string) ([]byte, int, string, error) {
	if err := s.warmup(ctx); err != nil {
		return nil, 0, "", err
	}

	if s.browserMode {
		body, err := s.browserGetHTML(ctx, pageURL)
		if err != nil {
			return nil, 0, "", err
		}
		return body, http.StatusOK, http.StatusText(http.StatusOK), nil
	}

	referer := dnsOrigin
	if u, err := url.Parse(pageURL); err == nil && u.Host != "" {
		referer = fmt.Sprintf("%s://%s/", u.Scheme, u.Host)
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

	if resp.StatusCode == http.StatusOK {
		return body, resp.StatusCode, resp.Status, nil
	}

	if !isAuthBlocked(resp.StatusCode) {
		return body, resp.StatusCode, resp.Status, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	s.log.Warn().
		Int("status", resp.StatusCode).
		Str("url", pageURL).
		Msg("dns fetch: HTTP blocked, switching to headless browser")

	s.mu.Lock()
	s.warmedUp = false
	s.browserMode = true
	s.mu.Unlock()

	if err := s.warmup(ctx); err != nil {
		return nil, resp.StatusCode, resp.Status, err
	}

	body, err = s.browserGetHTML(ctx, pageURL)
	if err != nil {
		return nil, resp.StatusCode, resp.Status, err
	}
	return body, http.StatusOK, http.StatusText(http.StatusOK), nil
}
