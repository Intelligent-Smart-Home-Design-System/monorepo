package wildberries

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

const wbOrigin = "https://www.wildberries.ru/"

func antibotWaitTimeout(userMode bool) time.Duration {
	if userMode {
		return 5 * time.Minute
	}
	return 90 * time.Second
}

// waitForAntibotClear waits until WBAAS leaves the "Почти готово..." challenge page.
func (s *Scraper) waitForAntibotClear(ctx context.Context, page *rod.Page, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return err
		}
		val, err := page.Eval(`() => ({
			title: document.title || '',
			path: location.pathname || '',
			html: (document.documentElement && document.documentElement.outerHTML || '').slice(0, 800),
		})`)
		if err != nil {
			return fmt.Errorf("antibot poll: %w", err)
		}
		title := val.Value.Get("title").Str()
		path := val.Value.Get("path").Str()
		html := val.Value.Get("html").Str()
		onChallenge := strings.Contains(title, "Почти готово") ||
			strings.Contains(path, "/__wbaas/challenges/antibot") ||
			strings.Contains(html, "/__wbaas/challenges/antibot")
		if !onChallenge {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("antibot challenge not cleared within %s", timeout)
}

func (s *Scraper) pageWithCtx(ctx context.Context) (*rod.Page, error) {
	s.browserMu.Lock()
	defer s.browserMu.Unlock()
	if s.browserPage == nil {
		return nil, fmt.Errorf("browser not active")
	}
	return s.browserPage.Context(ctx), nil
}

func (s *Scraper) activateBrowser(ctx context.Context) error {
	s.browserMu.Lock()
	defer s.browserMu.Unlock()

	if s.browserPage != nil {
		return nil
	}

	l := newBrowserLauncher(s.log, true, s.proxyURL, s.browserProfileDir)
	controlURL, err := l.Launch()
	if err != nil {
		return fmt.Errorf("launch browser: %w", err)
	}

	browser := rod.New().ControlURL(controlURL)
	if err := browser.Connect(); err != nil {
		return fmt.Errorf("connect browser: %w", err)
	}

	page, err := browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		browser.Close()
		return fmt.Errorf("open page: %w", err)
	}

	op := page.Context(ctx)

	if err := s.applySessionCookies(op); err != nil {
		browser.Close()
		return fmt.Errorf("apply session cookies: %w", err)
	}

	s.log.Info().Str("url", wbOrigin).Msg("wb browser: navigating homepage")
	if err := op.Navigate(wbOrigin); err != nil {
		browser.Close()
		return fmt.Errorf("navigate homepage: %w", err)
	}
	if err := op.WaitLoad(); err != nil {
		browser.Close()
		return fmt.Errorf("wait homepage: %w", err)
	}
	if err := s.waitForAntibotClear(ctx, op, antibotWaitTimeout(s.browserUserMode)); err != nil {
		s.log.Warn().Err(err).Msg("wb browser: antibot still active after homepage (solve captcha in visible Chrome)")
	}
	if err := s.waitForWBAASToken(ctx, op, 45*time.Second); err != nil {
		s.log.Warn().Err(err).Msg("wb browser: token not ready after homepage (captcha may be required)")
	}

	s.browser = browser
	// Keep page on a long-lived context — scrape task contexts are canceled per HTTP/subtest.
	s.browserPage = page.Context(context.Background())
	s.log.Info().Msg("wb browser: ready")
	if err := s.syncSessionFromBrowser(s.browserPage); err != nil {
		s.log.Warn().Err(err).Msg("wb browser: sync session from profile")
	}
	return nil
}

// Close shuts down the dedicated Chrome instance (call between isolated smoke steps).
func (s *Scraper) Close() {
	s.closeBrowser()
}

func (s *Scraper) syncSessionFromBrowser(page *rod.Page) error {
	if page == nil {
		return nil
	}
	cookies, err := page.Browser().GetCookies()
	if err != nil {
		return err
	}
	var token string
	cookieList := make([]Cookie, 0, len(cookies))
	for _, c := range cookies {
		if c.Name == "x_wbaas_token" && c.Value != "" {
			token = c.Value
		}
		cookieList = append(cookieList, Cookie{
			Name: c.Name, Value: c.Value, Domain: c.Domain, Path: c.Path,
		})
	}
	if token == "" {
		return nil
	}
	ua := defaultWBUserAgent
	if val, err := page.Eval(`() => navigator.userAgent`); err == nil {
		if s := val.Value.Str(); s != "" {
			ua = s
		}
	}
	s.mu.Lock()
	s.session = &Session{
		UserAgent: ua,
		Cookies:   cookieList,
		Token:     token,
		UpdatedAt: time.Now(),
	}
	sess := s.session
	s.mu.Unlock()
	if s.sessionPath != "" {
		return s.saveSession(sess)
	}
	return nil
}

func (s *Scraper) fetchHTMLViaBrowser(ctx context.Context, targetURL string) ([]byte, error) {
	if err := s.limiter.Wait(ctx); err != nil {
		return nil, err
	}
	if err := s.activateBrowser(ctx); err != nil {
		return nil, err
	}
	op, err := s.pageWithCtx(ctx)
	if err != nil {
		return nil, err
	}

	if err := op.Navigate(targetURL); err != nil {
		return nil, fmt.Errorf("navigate: %w", err)
	}
	if err := op.WaitLoad(); err != nil {
		return nil, fmt.Errorf("wait load: %w", err)
	}
	if err := s.waitForAntibotClear(ctx, op, antibotWaitTimeout(s.browserUserMode)); err != nil {
		return nil, err
	}
	_ = s.syncSessionFromBrowser(op)

	val, err := op.Eval(`() => document.documentElement.outerHTML`)
	if err != nil {
		return nil, fmt.Errorf("read html: %w", err)
	}
	html := val.Value.Str()
	if html == "" {
		return nil, fmt.Errorf("empty html from browser")
	}
	return []byte(html), nil
}

func (s *Scraper) waitForWBAASToken(ctx context.Context, page *rod.Page, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return err
		}
		cookies, err := page.Browser().GetCookies()
		if err != nil {
			return fmt.Errorf("read cookies: %w", err)
		}
		for _, c := range cookies {
			if c.Name == "x_wbaas_token" && c.Value != "" {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("x_wbaas_token not found in browser profile")
}

func (s *Scraper) applySessionCookies(page *rod.Page) error {
	s.mu.RLock()
	sess := s.session
	s.mu.RUnlock()
	if sess == nil {
		return nil
	}

	seen := make(map[string]bool)
	var params []*proto.NetworkCookieParam
	add := func(name, value, domain, path string) {
		if value == "" {
			return
		}
		key := name + "|" + domain
		if seen[key] {
			return
		}
		seen[key] = true
		params = append(params, &proto.NetworkCookieParam{
			Name:   name,
			Value:  value,
			Domain: domain,
			Path:   path,
			URL:    wbOrigin,
		})
	}

	for _, c := range sess.Cookies {
		domain := c.Domain
		if domain == "" {
			domain = ".wildberries.ru"
		}
		path := c.Path
		if path == "" {
			path = "/"
		}
		add(c.Name, c.Value, domain, path)
	}
	if sess.Token != "" {
		add("x_wbaas_token", sess.Token, ".wildberries.ru", "/")
		add("x_wbaas_token", sess.Token, "www.wildberries.ru", "/")
	}
	if len(params) == 0 {
		return nil
	}
	s.log.Debug().Int("cookies", len(params)).Msg("wb browser: applying session.json cookies to profile")
	return page.SetCookies(params)
}

func searchPageForAPI(targetURL string) string {
	u, err := url.Parse(targetURL)
	if err != nil {
		return wbOrigin
	}
	query := u.Query().Get("query")
	if query == "" {
		return wbOrigin + "catalog/0/search.aspx"
	}
	return wbOrigin + "catalog/0/search.aspx?search=" + url.QueryEscape(query)
}

func discoveryPageNumber(targetURL string) int {
	u, err := url.Parse(targetURL)
	if err != nil {
		return 1
	}
	page := u.Query().Get("page")
	if page == "" {
		return 1
	}
	var n int
	if _, err := fmt.Sscanf(page, "%d", &n); err != nil || n < 1 {
		return 1
	}
	return n
}

// fetchJSONViaBrowser loads __internal API URLs inside the dedicated Chrome profile.
func (s *Scraper) fetchJSONViaBrowser(ctx context.Context, targetURL string) ([]byte, error) {
	if err := s.ensureSessionFresh(); err != nil {
		return nil, err
	}
	if err := s.limiter.Wait(ctx); err != nil {
		return nil, err
	}
	if err := s.activateBrowser(ctx); err != nil {
		return nil, err
	}
	op, err := s.pageWithCtx(ctx)
	if err != nil {
		return nil, err
	}

	searchURL := searchPageForAPI(targetURL)
	// Same flow as wbsession: open search page, let WB's own JS call __internal API.
	if body, err := s.captureSearchAPIFromPage(ctx, op, searchURL); err == nil {
		_ = s.syncSessionFromBrowser(op)
		return body, nil
	} else {
		s.log.Debug().Err(err).Msg("wb browser: network capture failed, trying in-page fetch")
	}

	body, err := s.browserFetchJSON(op, targetURL)
	if err != nil {
		return nil, err
	}
	_ = s.syncSessionFromBrowser(op)
	return body, nil
}

func (s *Scraper) browserFetchJSON(page *rod.Page, targetURL string) ([]byte, error) {
	val, err := page.Eval(`async (url) => {
		const r = await fetch(url, {
			credentials: 'include',
			headers: {
				'Accept': 'application/json',
				'Referer': location.href,
			},
		});
		const text = await r.text();
		return { status: r.status, body: text };
	}`, targetURL)
	if err != nil {
		return nil, fmt.Errorf("browser fetch: %w", err)
	}

	status := val.Value.Get("status").Int()
	body := val.Value.Get("body").Str()
	if status == 403 || status == 498 {
		return nil, fmt.Errorf("session invalid (HTTP %d)", status)
	}
	if status != 200 {
		preview := body
		if len(preview) > 200 {
			preview = preview[:200]
		}
		return nil, fmt.Errorf("HTTP %d: %s", status, strings.TrimSpace(preview))
	}
	if body == "" {
		return nil, fmt.Errorf("empty response body")
	}
	return []byte(body), nil
}

func (s *Scraper) captureSearchAPIFromPage(ctx context.Context, page *rod.Page, searchURL string) ([]byte, error) {
	type hijackResult struct {
		status int
		body   string
	}

	page = page.Context(ctx)

	ch := make(chan hijackResult, 1)
	stop := page.EachEvent(func(e *proto.NetworkResponseReceived) {
		if e.Response == nil || !strings.Contains(e.Response.URL, "__internal/u-search") {
			return
		}
		body, err := proto.NetworkGetResponseBody{RequestID: e.RequestID}.Call(page)
		if err != nil {
			return
		}
		select {
		case ch <- hijackResult{status: int(e.Response.Status), body: body.Body}:
		default:
		}
	})
	defer stop()

	if err := page.Navigate(searchURL); err != nil {
		return nil, fmt.Errorf("navigate search page: %w", err)
	}
	if err := page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("wait search page: %w", err)
	}
	if err := s.waitForAntibotClear(ctx, page, antibotWaitTimeout(s.browserUserMode)); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-ch:
		if r.status == 403 || r.status == 498 {
			return nil, fmt.Errorf("session invalid (HTTP %d)", r.status)
		}
		if r.status != 200 {
			return nil, fmt.Errorf("HTTP %d", r.status)
		}
		if r.body == "" {
			return nil, fmt.Errorf("empty network response")
		}
		return []byte(r.body), nil
	case <-time.After(25 * time.Second):
		return nil, fmt.Errorf("search API response timeout")
	}
}

func (s *Scraper) closeBrowser() {
	s.browserMu.Lock()
	defer s.browserMu.Unlock()

	if s.browser != nil {
		s.browser.Close()
	}
	s.browser = nil
	s.browserPage = nil
}
