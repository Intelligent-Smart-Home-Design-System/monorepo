package dns

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
)

type pageCapture struct {
	HTML      []byte
	NavStatus int
	Title     string
	FinalURL  string
}

func (s *Scraper) newBrowserPage(browser *rod.Browser) (*rod.Page, error) {
	if s.forceBrowser {
		return browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	}
	return stealth.Page(browser)
}

func (s *Scraper) activateBrowser(ctx context.Context) error {
	s.browserMu.Lock()
	defer s.browserMu.Unlock()

	if s.browserPage != nil {
		return nil
	}

	// newBrowserLauncher() already sets proxy-server via applyBrowserProxy to
	// the local forwarder (netproxy.BrowserProxyServer) — do not overwrite it
	// with the raw upstream s.proxyURL here: Chrome's --proxy-server flag
	// can't take embedded credentials, so that overwrite broke every browser
	// launch with net::ERR_NO_SUPPORTED_PROXIES.
	l := s.newBrowserLauncher()

	controlURL, err := l.Launch()
	if err != nil {
		s.log.Error().Err(err).Msg("dns browser: launch failed")
		return fmt.Errorf("dns browser launch: %w", err)
	}

	browser := rod.New().ControlURL(controlURL)
	if err := browser.Connect(); err != nil {
		s.log.Error().Err(err).Msg("dns browser: connect failed")
		return fmt.Errorf("dns browser connect: %w", err)
	}

	page, err := s.newBrowserPage(browser)
	if err != nil {
		browser.Close()
		s.log.Error().Err(err).Msg("dns browser: page failed")
		return fmt.Errorf("dns browser page: %w", err)
	}

	s.log.Info().Str("url", dnsOrigin).Msg("dns browser: navigating homepage")
	cap, err := s.capturePage(ctx, page, dnsOrigin)
	if err != nil {
		browser.Close()
		return err
	}

	s.syncBrowserCookies(page)
	if ua, err := page.Context(ctx).Eval(`() => navigator.userAgent`); err == nil {
		s.userAgent = ua.Value.Str()
	}

	s.browser = browser
	s.browserPage = page
	s.browserMode = true

	s.log.Info().
		Int("nav_status", cap.NavStatus).
		Str("title", cap.Title).
		Int("cookies", s.cookieCount()).
		Msg("dns browser: ready")
	return nil
}

func (s *Scraper) browserGetHTML(ctx context.Context, pageURL string) ([]byte, error) {
	s.browserMu.Lock()
	defer s.browserMu.Unlock()

	if s.browserPage == nil {
		return nil, fmt.Errorf("dns browser not initialized")
	}

	s.log.Info().Str("url", pageURL).Msg("dns browser: fetching page")
	cap, err := s.capturePage(ctx, s.browserPage, pageURL)
	if err != nil {
		return nil, err
	}

	s.syncBrowserCookies(s.browserPage)
	s.logBrowseCapture(pageURL, cap)
	return cap.HTML, nil
}

func (s *Scraper) capturePage(ctx context.Context, page *rod.Page, pageURL string) (*pageCapture, error) {
	if err := page.Context(ctx).Navigate(pageURL); err != nil {
		return nil, fmt.Errorf("dns browser navigate: %w", err)
	}
	if err := page.Context(ctx).WaitLoad(); err != nil {
		return nil, fmt.Errorf("dns browser wait load: %w", err)
	}

	s.dismissCookieBanner(ctx, page)
	if !s.waitForQratorPass(ctx, page) {
		s.log.Warn().Str("url", pageURL).Msg("dns browser: qrator challenge may not have completed")
	}

	isHomepage := isDNSHomepage(pageURL)
	var pageKind string
	if isHomepage {
		pageKind = "homepage"
	} else {
		pageKind = s.waitForDNSContent(ctx, page)
	}

	meta, err := page.Context(ctx).Eval(`() => ({
		title: document.title || '',
		url: location.href || '',
		status: (performance.getEntriesByType('navigation')[0] || {}).responseStatus || 0
	})`)
	if err != nil {
		return nil, fmt.Errorf("dns browser page meta: %w", err)
	}

	title := meta.Value.Get("title").Str()
	finalURL := meta.Value.Get("url").Str()
	navStatus := meta.Value.Get("status").Int()

	val, err := page.Context(ctx).Eval(`() => document.documentElement.outerHTML`)
	if err != nil {
		return nil, fmt.Errorf("dns browser html: %w", err)
	}
	html := val.Value.Str()
	if html == "" {
		return nil, fmt.Errorf("dns browser html: empty document")
	}

	cap := &pageCapture{
		HTML:      []byte(html),
		NavStatus: navStatus,
		Title:     title,
		FinalURL:  finalURL,
	}

	if blocked, reason := isBlockedBrowsePage(cap.HTML, navStatus, title, isHomepage); blocked {
		s.log.Warn().
			Str("url", pageURL).
			Str("final_url", finalURL).
			Int("nav_status", navStatus).
			Str("title", title).
			Str("page_kind", pageKind).
			Str("reason", reason).
			Int("bytes", len(html)).
			Msg("dns browser: Qrator/blocked page (headless detected?)")
		return nil, fmt.Errorf("dns browser blocked: %s (HTTP %d, title=%q)", reason, navStatus, title)
	}

	return cap, nil
}

func isDNSHomepage(pageURL string) bool {
	u, err := url.Parse(pageURL)
	if err != nil {
		return false
	}
	if u.Host != "www.dns-shop.ru" && u.Host != "dns-shop.ru" {
		return false
	}
	path := strings.TrimSuffix(u.Path, "/")
	return path == ""
}

func isBlockedBrowsePage(html []byte, navStatus int, title string, isHomepage bool) (bool, string) {
	text := strings.ToLower(string(html))
	titleLower := strings.ToLower(title)

	if strings.Contains(text, "qauth_utm") || strings.Contains(text, "__qrator/qauth") {
		return true, "qrator_challenge"
	}

	if navStatus == http.StatusForbidden || navStatus == http.StatusUnauthorized {
		if !(isHomepage && len(html) > 15_000) {
			return true, "navigation status"
		}
	}

	for _, marker := range []string{
		"403 forbidden",
		"access denied",
		"доступ запрещ",
		"request blocked",
	} {
		if strings.Contains(text, marker) || strings.Contains(titleLower, marker) {
			return true, marker
		}
	}

	if strings.Contains(titleLower, "http 403") || strings.Contains(titleLower, "http 401") {
		return true, "blocked title"
	}

	if isHomepage && len(html) > 15_000 {
		return false, ""
	}

	pageKind, _, _ := diagnoseBrowseHTMLQuick(html)
	if pageKind == "empty_shell" && len(html) < 20_000 {
		return true, "empty_shell"
	}
	return false, ""
}

func (s *Scraper) waitForQratorPass(ctx context.Context, page *rod.Page) bool {
	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return false
		}

		state, err := page.Context(ctx).Eval(`() => {
			const title = document.title || '';
			const html = document.documentElement.outerHTML;
			const hasQauth = !!document.querySelector('script[src*="qauth"]');
			const blockedTitle = /HTTP\s*40[13]/i.test(title);
			return {
				hasQauth: hasQauth,
				blockedTitle: blockedTitle,
				bytes: html.length,
				title: title,
			};
		}`)
		if err != nil {
			time.Sleep(400 * time.Millisecond)
			continue
		}

		hasQauth := state.Value.Get("hasQauth").Bool()
		blockedTitle := state.Value.Get("blockedTitle").Bool()
		bytes := state.Value.Get("bytes").Int()
		title := state.Value.Get("title").Str()

		if !hasQauth && !blockedTitle && bytes > 15_000 {
			s.log.Info().
				Int("bytes", bytes).
				Str("title", title).
				Msg("dns browser: qrator challenge passed")
			return true
		}

		time.Sleep(400 * time.Millisecond)
	}

	s.log.Warn().Msg("dns browser: qrator challenge timeout")
	return false
}

func diagnoseBrowseHTMLQuick(html []byte) (pageKind string, subcategoryAnchors, productAnchors int) {
	text := string(html)
	if strings.Contains(text, "subcategory__item") {
		pageKind = "hub"
	}
	if strings.Contains(text, "catalog-product__image-link") {
		pageKind = "grid"
	}
	// rough counts for logging
	subcategoryAnchors = strings.Count(text, "subcategory__item")
	productAnchors = strings.Count(text, "catalog-product__image-link")
	if pageKind == "" {
		if len(html) < 20_000 && strings.Contains(text, "header-plug") {
			pageKind = "empty_shell"
		} else {
			pageKind = "unknown"
		}
	}
	return pageKind, subcategoryAnchors, productAnchors
}

func (s *Scraper) logBrowseCapture(pageURL string, cap *pageCapture) {
	pageKind, subAnchors, prodAnchors := diagnoseBrowseHTMLQuick(cap.HTML)
	s.log.Info().
		Str("url", pageURL).
		Str("final_url", cap.FinalURL).
		Int("nav_status", cap.NavStatus).
		Str("title", cap.Title).
		Str("page_kind", pageKind).
		Int("bytes", len(cap.HTML)).
		Int("subcategory_anchors", subAnchors).
		Int("product_anchors", prodAnchors).
		Msg("dns browser: page fetched")
}

func (s *Scraper) dismissCookieBanner(ctx context.Context, page *rod.Page) {
	_, err := page.Context(ctx).Eval(`() => {
		const labels = ['Понятно', 'OK', 'Accept'];
		for (const label of labels) {
			const btn = [...document.querySelectorAll('button')].find(
				b => (b.textContent || '').trim() === label
			);
			if (btn) { btn.click(); return true; }
		}
		return false;
	}`)
	if err == nil {
		s.log.Debug().Msg("dns browser: cookie banner dismissed")
	}
}

func (s *Scraper) waitForDNSContent(ctx context.Context, page *rod.Page) string {
	selectors := []struct {
		sel  string
		kind string
	}{
		{"a.subcategory__item", "hub"},
		{"a.catalog-product__image-link", "grid"},
		{".subcategory[data-subcategory-container]", "hub"},
		{".products-page", "grid"},
	}

	deadline := time.Now().Add(25 * time.Second)
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			break
		}
		for _, item := range selectors {
			has, _, err := page.Context(ctx).Has(item.sel)
			if err == nil && has {
				s.log.Info().
					Str("selector", item.sel).
					Str("page_kind", item.kind).
					Msg("dns browser: content ready")
				time.Sleep(500 * time.Millisecond)
				return item.kind
			}
		}
		time.Sleep(400 * time.Millisecond)
	}

	s.log.Warn().Msg("dns browser: content selectors timeout")
	return "unknown"
}

func (s *Scraper) syncBrowserCookies(page *rod.Page) {
	cookies, err := page.Cookies([]string{dnsOrigin})
	if err != nil {
		s.log.Warn().Err(err).Msg("dns browser: failed to read cookies")
		return
	}

	u, err := url.Parse(dnsOrigin)
	if err != nil {
		return
	}

	httpCookies := make([]*http.Cookie, 0, len(cookies))
	for _, c := range cookies {
		httpCookies = append(httpCookies, &http.Cookie{
			Name:   c.Name,
			Value:  c.Value,
			Domain: c.Domain,
			Path:   c.Path,
		})
	}
	s.client.Jar.SetCookies(u, httpCookies)
}

func (s *Scraper) closeBrowser() {
	s.browserMu.Lock()
	defer s.browserMu.Unlock()

	if s.browser != nil {
		s.browser.Close()
	}
	s.browser = nil
	s.browserPage = nil
	s.browserMode = false
}

func (s *Scraper) cookieCount() int {
	u, err := url.Parse(dnsOrigin)
	if err != nil {
		return 0
	}
	return len(s.client.Jar.Cookies(u))
}
