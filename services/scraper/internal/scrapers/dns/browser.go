package dns

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
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
	time.Sleep(1500 * time.Millisecond)

	val, err := s.browserPage.Context(ctx).Eval(`() => document.documentElement.outerHTML`)
	if err != nil {
		return nil, fmt.Errorf("dns browser html: %w", err)
	}

	html := val.Value.Str()
	if html == "" {
		return nil, fmt.Errorf("dns browser html: empty document")
	}

	s.syncBrowserCookies(s.browserPage)
	s.log.Info().Str("url", pageURL).Int("bytes", len(html)).Msg("dns browser: page fetched")
	return []byte(html), nil
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
