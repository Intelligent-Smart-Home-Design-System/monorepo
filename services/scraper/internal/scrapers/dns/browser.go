package dns

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/stealth"
)

func (s *Scraper) activateBrowser(ctx context.Context) error {
	s.browserMu.Lock()
	defer s.browserMu.Unlock()

	if s.browserPage != nil {
		return nil
	}

	s.log.Info().Msg("dns browser: launching headless Chrome")

	l := launcher.New().Headless(true).Set("no-sandbox").Set("disable-setuid-sandbox")
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

	page, err := stealth.Page(browser)
	if err != nil {
		browser.Close()
		s.log.Error().Err(err).Msg("dns browser: stealth page failed")
		return fmt.Errorf("dns browser page: %w", err)
	}

	s.log.Info().Str("url", dnsOrigin).Msg("dns browser: navigating homepage")
	if err := page.Context(ctx).Navigate(dnsOrigin); err != nil {
		browser.Close()
		s.log.Error().Err(err).Str("url", dnsOrigin).Msg("dns browser: homepage navigation failed")
		return fmt.Errorf("dns browser homepage: %w", err)
	}
	if err := page.Context(ctx).WaitLoad(); err != nil {
		browser.Close()
		return fmt.Errorf("dns browser homepage wait: %w", err)
	}
	time.Sleep(2 * time.Second)

	s.syncBrowserCookies(page)

	if ua, err := page.Context(ctx).Eval(`() => navigator.userAgent`); err == nil {
		s.userAgent = ua.Value.Str()
	}

	s.browser = browser
	s.browserPage = page
	s.browserMode = true

	s.log.Info().Int("cookies", s.cookieCount()).Msg("dns browser: ready")
	return nil
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
	s.log.Info().Int("cookies", len(httpCookies)).Msg("dns browser: cookies synced to HTTP client")
}

func (s *Scraper) browserGetHTML(ctx context.Context, pageURL string) ([]byte, error) {
	s.browserMu.Lock()
	defer s.browserMu.Unlock()

	if s.browserPage == nil {
		return nil, fmt.Errorf("dns browser not initialized")
	}

	s.log.Info().Str("url", pageURL).Msg("dns browser: fetching page")
	if err := s.browserPage.Context(ctx).Navigate(pageURL); err != nil {
		return nil, fmt.Errorf("dns browser navigate: %w", err)
	}
	if err := s.browserPage.Context(ctx).WaitLoad(); err != nil {
		return nil, fmt.Errorf("dns browser wait load: %w", err)
	}

	s.dismissCookieBanner(ctx, s.browserPage)
	pageKind := s.waitForDNSContent(ctx, s.browserPage)

	val, err := s.browserPage.Context(ctx).Eval(`() => document.documentElement.outerHTML`)
	if err != nil {
		return nil, fmt.Errorf("dns browser html: %w", err)
	}

	html := val.Value.Str()
	if html == "" {
		return nil, fmt.Errorf("dns browser html: empty document")
	}

	s.syncBrowserCookies(s.browserPage)
	s.logBrowseSignals(pageURL, pageKind, html)
	return []byte(html), nil
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

	deadline := time.Now().Add(20 * time.Second)
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

	s.log.Warn().Msg("dns browser: content selectors timeout (page may be empty shell)")
	return "unknown"
}

func (s *Scraper) logBrowseSignals(pageURL, pageKind string, html string) {
	s.log.Info().
		Str("url", pageURL).
		Str("page_kind", pageKind).
		Int("bytes", len(html)).
		Bool("has_subcategory_item", strings.Contains(html, "subcategory__item")).
		Bool("has_product_link", strings.Contains(html, "catalog-product__image-link")).
		Bool("has_products_page", strings.Contains(html, "products-page")).
		Bool("has_header_plug", strings.Contains(html, "header-plug")).
		Msg("dns browser: page fetched")
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
