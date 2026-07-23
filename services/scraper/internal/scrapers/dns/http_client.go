package dns

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/netproxy"
)

const dnsOrigin = "https://www.dns-shop.ru/"

// newScraperClient routes through the same local forwarder as the browser
// path (netproxy.BrowserProxyServer): it handles authenticated and/or
// TLS-fronted (https-scheme) upstream proxies uniformly, so http.Client only
// ever talks to a plain, no-auth local proxy — the upstream's own TLS/auth
// quirks are dealt with in one place.
func newScraperClient(timeout time.Duration, proxyURL string) (*http.Client, error) {
	transport := &http.Transport{}
	localProxy, err := netproxy.BrowserProxyServer(proxyURL)
	if err != nil {
		return nil, err
	}
	if localProxy != "" {
		u, err := url.Parse(localProxy)
		if err != nil {
			return nil, fmt.Errorf("parse local proxy URL: %w", err)
		}
		transport.Proxy = http.ProxyURL(u)
	}
	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		Jar:       jar,
	}, nil
}

func setNavigationHeaders(req *http.Request, userAgent, referer string) {
	req.Header.Set("User-Agent", userAgent)
	// ---
	req.Header.Set("Sec-Ch-Ua", `"Not_A Brand";v="8", "Chromium";v="120", "Google Chrome";v="120"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?1")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Android"`)
	// -
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

// Warmup establishes cookies/session state used by subsequent Scrape calls.
func (s *Scraper) Warmup(ctx context.Context) error {
	return s.warmup(ctx)
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

	// forceBrowser only picks which Chrome launch mode activateBrowser uses
	// (real user profile vs headless) — it must not skip this HTTP attempt.
	// Going straight to Chrome cold (no prior request through the local proxy
	// forwarder) reproducibly fails with net::ERR_NO_SUPPORTED_PROXIES; warming
	// the forwarder up with one HTTP request first avoids it.
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

	s.log.Warn().Err(httpErr).Msg("dns warmup: HTTP blocked by Qrator, switching to browser")
	if err := s.activateBrowser(ctx); err != nil {
		return err
	}

	if err := s.warmupHTTP(ctx); err == nil {
		s.mu.Lock()
		s.browserMode = false
		s.warmedUp = true
		s.mu.Unlock()
		s.log.Info().
			Int("cookies", s.cookieCount()).
			Str("mode", "http").
			Msg("dns warmup: ok (HTTP after browser cookies)")
		return nil
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

// isAuthBlocked reports statuses that mean "this identity is blocked/throttled" —
// worth rotating proxy IP + cookies for, since a new IP tends to clear them.
func isAuthBlocked(status int) bool {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusTooManyRequests, http.StatusServiceUnavailable:
		return true
	default:
		return false
	}
}

// isGatewayError reports upstream infra errors (WAF/proxy overloaded) that are
// not tied to our identity — retrying is worth it, but rotating the IP won't
// fix a stressed origin, so we don't burn a rotation on these.
func isGatewayError(status int) bool {
	return status == http.StatusBadGateway || status == http.StatusGatewayTimeout
}

// getHTML drives the per-page fetch retry loop. Each attempt delegates to
// attemptBrowserFetch or attemptHTTPFetch depending on current mode; those
// report back whether the attempt succeeded/failed outright (retry=false,
// return their body/status/err as-is) or hit a recoverable condition worth
// looping again for (retry=true).
func (s *Scraper) getHTML(ctx context.Context, taskID int, pageURL string) ([]byte, int, string, error) {
	const maxAttempts = 3

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := s.warmup(ctx); err != nil {
			return nil, 0, "", err
		}

		var body []byte
		var status int
		var statusText string
		var err error
		var retry bool
		if s.browserMode {
			body, status, statusText, err, retry = s.attemptBrowserFetch(ctx, taskID, pageURL, attempt, maxAttempts)
		} else {
			body, status, statusText, err, retry = s.attemptHTTPFetch(ctx, taskID, pageURL, attempt, maxAttempts)
		}
		if !retry {
			return body, status, statusText, err
		}
	}

	return nil, 0, "", fmt.Errorf("unexpected end of getHTML loop")
}

// attemptBrowserFetch tries one page fetch through the active Chrome session.
// A failure here (Qrator still blocking, page crashed, etc.) is treated like
// an auth block: rotate the proxy identity and retry, unless attempts are
// exhausted.
func (s *Scraper) attemptBrowserFetch(ctx context.Context, taskID int, pageURL string, attempt, maxAttempts int) (body []byte, status int, statusText string, err error, retry bool) {
	body, err = s.browserGetHTML(ctx, pageURL)
	if err == nil {
		return body, http.StatusOK, http.StatusText(http.StatusOK), nil, false
	}

	if attempt == maxAttempts {
		return nil, 0, "", err, false
	}

	s.log.Warn().
		Err(err).
		Int("attempt", attempt).
		Int("task_id", taskID).
		Str("url", pageURL).
		Msg("dns browser: fetch failed, rotating proxy and retrying")
	s.rotateProxyAndWait(ctx, taskID)
	return nil, 0, "", nil, true
}

// attemptHTTPFetch tries one page fetch via the plain HTTP client. Outcomes:
//   - success (200): done, no retry
//   - transient network error (dropped connection, timeout): rotate + retry
//   - gateway error (502/504): the origin/WAF is overloaded, not our
//     identity — short backoff and retry, no rotation
//   - blocked (401/403/429/503): escalate to browser mode and retry. A bare
//     HTTP retry after rotation rarely clears a Qrator challenge on its own —
//     browser mode reliably does, often without even needing a new IP.
//   - anything else: done, no retry
func (s *Scraper) attemptHTTPFetch(ctx context.Context, taskID int, pageURL string, attempt, maxAttempts int) (body []byte, status int, statusText string, err error, retry bool) {
	referer := dnsOrigin
	if u, perr := url.Parse(pageURL); perr == nil && u.Host != "" {
		referer = fmt.Sprintf("%s://%s/", u.Scheme, u.Host)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, 0, "", err, false
	}
	setNavigationHeaders(req, s.userAgent, referer)

	resp, err := s.client.Do(req)
	if err != nil {
		if attempt == maxAttempts {
			return nil, 0, "", err, false
		}
		s.log.Error().Err(err).Int("task_id", taskID).Msg("Сетевая ошибка при запросе, пробуем сбросить сессию")
		s.rotateProxyAndWait(ctx, taskID)
		return nil, 0, "", nil, true
	}

	body, readErr := io.ReadAll(resp.Body)
	resp.Body.Close()
	if readErr != nil {
		return nil, resp.StatusCode, resp.Status, readErr, false
	}

	if resp.StatusCode == http.StatusOK {
		return body, resp.StatusCode, resp.Status, nil, false
	}

	if isGatewayError(resp.StatusCode) {
		if attempt == maxAttempts {
			return body, resp.StatusCode, resp.Status, fmt.Errorf("HTTP %d: gateway error after max retries", resp.StatusCode), false
		}
		s.log.Warn().
			Int("status", resp.StatusCode).
			Int("attempt", attempt).
			Int("task_id", taskID).
			Str("url", pageURL).
			Msg("dns fetch: upstream gateway error, retrying without proxy rotation")
		time.Sleep(2 * time.Second)
		return nil, 0, "", nil, true
	}

	if !isAuthBlocked(resp.StatusCode) {
		return body, resp.StatusCode, resp.Status, fmt.Errorf("HTTP %d", resp.StatusCode), false
	}

	if attempt == maxAttempts {
		s.log.Error().Int("status", resp.StatusCode).Int("task_id", taskID).Msg("Превышено количество попыток обхода блокировки. Завершаем работу.")
		return body, resp.StatusCode, resp.Status, fmt.Errorf("HTTP %d: blocked after max retries", resp.StatusCode), false
	}

	s.log.Warn().
		Int("status", resp.StatusCode).
		Int("attempt", attempt).
		Int("task_id", taskID).
		Str("url", pageURL).
		Msg("dns fetch: HTTP blocked, switching to browser")
	s.mu.Lock()
	s.warmedUp = false
	s.browserMode = true
	s.mu.Unlock()
	return nil, 0, "", nil, true
}

func (s *Scraper) prepareForProxyRotation() {
	s.mu.Lock()
	s.warmedUp = false
	// Очищаем старые забаненные куки Qrator/DNS из клиента
	newJar, _ := cookiejar.New(nil)
	s.client.Jar = newJar
	s.mu.Unlock()
}

// rotateProxyAndWait clears cookies and blocks (bounded) until the mobile
// proxy's own scheduled rotation hands us a new IP.
func (s *Scraper) rotateProxyAndWait(ctx context.Context, taskID int) {
	s.prepareForProxyRotation()
	s.log.Info().Int("task_id", taskID).Msg("dns fetch: запускаем ротацию мобильного прокси...")

	// Bound the wait: the provider's own auto-rotation is expected within a
	// few minutes (this account rotates ~every 2min); without a cap a
	// dead/never-rotating session would hang here indefinitely.
	rotateCtx, cancel := context.WithTimeout(ctx, 4*time.Minute)
	defer cancel()
	netproxy.RotateSharedProxy(rotateCtx, s.proxyURL, func(msg string) {
		s.log.Info().Msg(msg)
	})
}
