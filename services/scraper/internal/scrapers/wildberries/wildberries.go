package wildberries

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
	"github.com/rs/zerolog"
	"golang.org/x/time/rate"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/netproxy"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/pathnorm"
)

const defaultWBUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

type Session struct {
	UserAgent string    `json:"userAgent"`
	Cookies   []Cookie  `json:"cookies"`
	Token     string    `json:"token"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Cookie struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Domain string `json:"domain"`
	Path   string `json:"path"`
}

type Scraper struct {
	client               *http.Client
	proxyURL             string
	limiter              *rate.Limiter
	cardBasket           string
	sessionPath          string
	session              *Session
	mu                   sync.RWMutex
	discoveryURLTemplate string
	discoveryMaxPages    int
	browserUserMode      bool
	browserProfileDir    string
	browserMu            sync.Mutex
	browser              *rod.Browser
	browserPage          *rod.Page
	log                  zerolog.Logger
}

func NewScraper(
	log zerolog.Logger,
	timeout time.Duration,
	proxyURL, cardBasket string,
	rps float64,
	sessionPath string,
	discoveryURLTemplate string,
	discoveryMaxPages int,
	browserUserMode *bool,
	browserProfileDir string,
) *Scraper {
	transport := &http.Transport{}
	if err := netproxy.ConfigureTransport(transport, proxyURL); err != nil {
		log.Warn().Err(err).Str("proxy", netproxy.RedactURL(proxyURL)).Msg("wb: invalid proxy URL, requests will be direct")
		proxyURL = ""
	}
	limiter := rate.NewLimiter(rate.Limit(rps), 1)
	if sessionPath != "" {
		if abs, err := pathnorm.Abs(sessionPath); err == nil {
			sessionPath = abs
		}
	}
	var session *Session
	if data, err := os.ReadFile(sessionPath); err == nil {
		var sess Session
		if err := json.Unmarshal(data, &sess); err == nil && time.Since(sess.UpdatedAt) < time.Hour {
			session = &sess
		}
	}
	return &Scraper{
		client: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
		proxyURL:             proxyURL,
		limiter:              limiter,
		cardBasket:           cardBasket,
		sessionPath:          sessionPath,
		session:              session,
		discoveryURLTemplate: discoveryURLTemplate,
		discoveryMaxPages:    discoveryMaxPages,
		browserUserMode:      defaultBrowserUserMode(browserUserMode),
		browserProfileDir:    resolveWBBrowserProfile(browserProfileDir),
		log:                  log,
	}
}

func (s *Scraper) loadSession() (*Session, error) {
	data, err := os.ReadFile(s.sessionPath)
	if err != nil {
		return nil, err
	}
	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, err
	}
	if time.Since(sess.UpdatedAt) > time.Hour {
		return nil, fmt.Errorf("session expired")
	}
	s.session = &sess
	return &sess, nil
}

func (s *Scraper) saveSession(sess *Session) error {
	sess.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.sessionPath, data, 0600)
}

func (s *Scraper) mineSession(ctx context.Context) (*Session, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	mode := "headless"
	if s.browserUserMode {
		mode = "user"
	}
	s.log.Info().Str("mode", mode).Msg("mineSession: launching browser")

	l := newBrowserLauncher(s.log, s.browserUserMode, s.proxyURL, s.browserProfileDir)
	controlURL, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("launch browser: %w", err)
	}
	browser := rod.New().ControlURL(controlURL)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("connect browser: %w", err)
	}
	defer browser.Close()

	var page *rod.Page
	if s.browserUserMode {
		page, err = browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
		if err != nil {
			return nil, fmt.Errorf("open page: %w", err)
		}
	} else {
		page = stealth.MustPage(browser)
	}
	defer page.Close()

	page = page.Context(ctx)
	if err := page.Navigate("https://www.wildberries.ru/"); err != nil {
		return nil, fmt.Errorf("navigate: %w", err)
	}
	_ = page.MustWaitIdle()

	if s.browserUserMode {
		s.log.Info().Msg("mineSession: solve captcha in Chrome if shown, then waiting for token...")
		if err := page.Navigate("https://www.wildberries.ru/catalog/0/search.aspx?search=умный%20дом"); err == nil {
			_ = page.MustWaitIdle()
		}
	}

	deadline := time.Now().Add(45 * time.Second)
	if s.browserUserMode {
		if d, ok := ctx.Deadline(); ok {
			deadline = d
		} else {
			deadline = time.Now().Add(5 * time.Minute)
		}
		s.log.Info().Time("until", deadline).Msg("mineSession: waiting for x_wbaas_token in visible Chrome")
	}
	var cookies []*proto.NetworkCookie
	var tokenValue string
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		cookies, err = page.Browser().GetCookies()
		if err != nil {
			return nil, fmt.Errorf("read cookies: %w", err)
		}
		for _, c := range cookies {
			if c.Name == "x_wbaas_token" && c.Value != "" {
				tokenValue = c.Value
				break
			}
		}
		if tokenValue != "" {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if tokenValue == "" {
		return nil, fmt.Errorf("x_wbaas_token not found in cookies (try -user-mode or complete captcha in visible Chrome)")
	}

	cookieList := make([]Cookie, 0, len(cookies))
	for _, c := range cookies {
		cookieList = append(cookieList, Cookie{
			Name:   c.Name,
			Value:  c.Value,
			Domain: c.Domain,
			Path:   c.Path,
		})
	}
	s.log.Info().Int("cookies", len(cookieList)).Msg("mineSession: cookies obtained")

	uaVal, err := page.Eval(`() => navigator.userAgent`)
	if err != nil {
		return nil, fmt.Errorf("user agent: %w", err)
	}
	userAgent := uaVal.Value.Str()

	return &Session{
		UserAgent: userAgent,
		Cookies:   cookieList,
		Token:     tokenValue,
	}, nil
}

// Warmup loads session (best-effort) and opens Chrome in user-mode.
func (s *Scraper) Warmup(ctx context.Context) error {
	if s.browserUserMode {
		s.loadSessionBestEffort()
		return s.activateBrowser(ctx)
	}
	return s.ensureSessionFresh()
}

// RefreshSession mines a fresh browser session and writes it to sessionPath.
func (s *Scraper) RefreshSession(ctx context.Context) error {
	sess, err := s.mineSession(ctx)
	if err != nil {
		return err
	}
	sess.UpdatedAt = time.Now()
	s.mu.Lock()
	s.session = sess
	s.mu.Unlock()
	if s.sessionPath == "" {
		return fmt.Errorf("session path is empty")
	}
	return s.saveSession(sess)
}

func (s *Scraper) loadSessionBestEffort() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session != nil {
		return
	}
	_, _ = s.loadSession()
}

func (s *Scraper) ensureSessionFresh() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.session != nil && time.Since(s.session.UpdatedAt) < 30*time.Minute {
		return nil
	}

	_, err := s.loadSession()
	if err == nil && s.session != nil && time.Since(s.session.UpdatedAt) < 30*time.Minute {
		return nil
	}

	if s.browserUserMode {
		return fmt.Errorf("WB session missing or expired; run with the same proxy: go run ./cmd/wbsession -config <cfg> -out session.json")
	}

	s.log.Debug().Msg("ensureSession: session not found or expired, mining new session...")
	sess, err := s.mineSession(context.Background())
	if err != nil {
		return fmt.Errorf("mine session: %w", err)
	}
	sess.UpdatedAt = time.Now()
	s.session = sess
	if err := s.saveSession(sess); err != nil {
		s.log.Warn().Err(err).Msg("failed to save session")
	}
	return nil
}

func (s *Scraper) fetchJSON(ctx context.Context, url string) ([]byte, error) {
	s.log.Debug().Str("url", url).Msg("fetchJSON: start")
	needsSession := strings.Contains(url, "wildberries.ru/__internal/") || strings.Contains(url, ".geobasket.ru/")
	if needsSession {
		if err := s.ensureSessionFresh(); err != nil {
			s.log.Debug().Err(err).Msg("fetchJSON: ensureSessionFresh error")
			return nil, err
		}
	} else {
		s.loadSessionBestEffort()
	}
	s.mu.RLock()
	sess := s.session
	s.mu.RUnlock()
	tokenLen := 0
	if sess != nil {
		tokenLen = len(sess.Token)
	}
	s.log.Debug().Int("token_len", tokenLen).Msg("fetchJSON: session ok")
	if err := s.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	ua := defaultWBUserAgent
	var token string
	var cookies []Cookie
	if sess != nil {
		if sess.UserAgent != "" {
			ua = sess.UserAgent
		}
		token = sess.Token
		cookies = sess.Cookies
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Referer", "https://www.wildberries.ru/")
	req.Header.Set("Origin", "https://www.wildberries.ru")
	if token != "" {
		req.Header.Set("Cookie", "x_wbaas_token="+token)
	}
	for _, c := range cookies {
		if c.Name == "x_wbaas_token" {
			continue
		}
		req.AddCookie(&http.Cookie{
			Name:   c.Name,
			Value:  c.Value,
			Domain: c.Domain,
			Path:   c.Path,
		})
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 || resp.StatusCode == 498 {
		s.mu.Lock()
		s.session = nil
		s.mu.Unlock()
		return nil, fmt.Errorf("session invalid (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	s.log.Debug().Str("url", url).Int("body_size", len(body)).Msg("fetchJSON: complete")
	return body, nil
}

func (s *Scraper) fetchWithRetry(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	for i := 0; i < 3; i++ {
		s.log.Debug().Int("attempt", i+1).Str("url", url).Msg("fetchWithRetry: attempt")

		var body []byte
		var err error
		if s.browserUserMode && strings.Contains(url, "wildberries.ru") {
			if strings.Contains(url, "__internal/") {
				body, err = s.fetchJSONViaBrowser(ctx, url)
			} else {
				body, err = s.fetchHTMLViaBrowser(ctx, url)
			}
		} else {
			body, err = s.fetchJSON(ctx, url)
		}
		if err == nil {
			s.log.Debug().Msg("fetchWithRetry: success")
			return body, nil
		}
		s.log.Debug().Err(err).Msg("fetchWithRetry: error")
		lastErr = err
		if strings.Contains(err.Error(), "session invalid") && !s.browserUserMode {
			s.mu.Lock()
			s.session = nil
			s.mu.Unlock()
			continue
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(i+1) * 2 * time.Second):
		}
	}
	return nil, fmt.Errorf("max retries: %w", lastErr)
}

func (s *Scraper) urlExists(ctx context.Context, url string) bool {
	if s.session == nil {
		s.log.Debug().Msg("urlExists: session is nil")
		return false
	}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		s.log.Debug().Err(err).Msg("urlExists: request error")
		return false
	}
	req.Header.Set("User-Agent", s.session.UserAgent)
	req.Header.Set("Range", "bytes=0-0")
	resp, err := s.client.Do(req)
	if err != nil {
		s.log.Debug().Err(err).Msg("urlExists: do error")
		return false
	}
	defer resp.Body.Close()
	s.log.Debug().Int("status", resp.StatusCode).Str("url", url).Msg("urlExists: status")
	return resp.StatusCode == http.StatusOK
}

func (s *Scraper) getCardURL(ctx context.Context, nmID int) (string, error) {
	newURL := buildCardURLNew("01", nmID)
	s.log.Debug().Str("url", newURL).Msg("checking new CDN")
	if s.urlExists(ctx, newURL) {
		s.log.Debug().Msg("new CDN works")
		return newURL, nil
	}
	vol := nmID / 100000
	part := nmID / 1000
	for basket := 1; basket <= 41; basket++ {
		oldURL := fmt.Sprintf("https://basket-%d.wbbasket.ru/vol%d/part%d/%d/info/ru/card.json", basket, vol, part, nmID)
		s.log.Debug().Int("basket", basket).Str("url", oldURL).Msg("checking basket")
		if s.urlExists(ctx, oldURL) {
			s.log.Debug().Int("basket", basket).Msg("found working basket")
			return oldURL, nil
		}
	}
	return "", fmt.Errorf("card.json not found for nm_id %d", nmID)
}

func extractNmID(rawURL string) (int, error) {
	parts := strings.Split(rawURL, "/catalog/")
	if len(parts) < 2 {
		return 0, fmt.Errorf("catalog not found")
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

func buildCardURLNew(basket string, nmID int) string {
	vol := nmID / 100000
	part := nmID / 1000
	return fmt.Sprintf("https://mow-basket-cdn-%s.geobasket.ru/vol%d/part%d/%d/info/ru/card.json", basket, vol, part, nmID)
}

func (s *Scraper) Scrape(ctx context.Context, task domain.ScrapeTask) (*domain.ScrapeResult, error) {
	switch task.PageType {
	case domain.PageTypeListing:
		if err := s.ensureSessionFresh(); err != nil {
			return nil, fmt.Errorf("ensure session: %w", err)
		}
		return s.scrapeListing(ctx, task)
	case domain.PageTypeDiscovery:
		return s.scrapeDiscoveryTask(ctx, task)
	case domain.PageTypeCategory:
    	return s.scrapeCategory(ctx, task)
	default:
		return nil, fmt.Errorf("unsupported page type %s for source %s", task.PageType.String(), task.Source)
	}
}

func (s *Scraper) scrapeListing(ctx context.Context, task domain.ScrapeTask) (*domain.ScrapeResult, error) {
	nmID, err := extractNmID(task.URL)
	if err != nil {
		return nil, err
	}

	detailURL := fmt.Sprintf(
		"https://www.wildberries.ru/__internal/u-card/cards/v4/detail?appType=1&curr=rub&dest=-1257786&spp=30&hide_vflags=4294967296&ab_testing=false&lang=ru&nm=%d",
		nmID,
	)

	cardURL, err := s.getCardURL(ctx, nmID)
	if err != nil {
		return nil, fmt.Errorf("get card URL: %w", err)
	}

	detailBody, err := s.fetchWithRetry(ctx, detailURL)
	if err != nil {
		return nil, fmt.Errorf("detail.json: %w", err)
	}

	cardBody, err := s.fetchWithRetry(ctx, cardURL)
	if err != nil {
		return nil, fmt.Errorf("card.json: %w", err)
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

func (s *Scraper) scrapeDiscoveryTask(ctx context.Context, task domain.ScrapeTask) (*domain.ScrapeResult, error) {
	const prefix = "wildberries://discovery/"
	if !strings.HasPrefix(task.URL, prefix) {
		return nil, fmt.Errorf("invalid discovery URL: %s", task.URL)
	}
	query := strings.TrimPrefix(task.URL, prefix)
	if query == "" {
		return nil, fmt.Errorf("empty discovery query")
	}
	resources, err := s.scrapeDiscovery(ctx, query, s.discoveryMaxPages, s.discoveryURLTemplate)
	if err != nil {
		return nil, fmt.Errorf("discovery scrape failed: %w", err)
	}
	return &domain.ScrapeResult{Resources: resources}, nil
}

func (s *Scraper) scrapeDiscovery(ctx context.Context, query string, maxPages int, urlTemplate string) ([]domain.Resource, error) {
	var resources []domain.Resource
	for page := 1; page <= maxPages; page++ {
		searchURL := BuildDiscoverySearchURL(urlTemplate, query, page)

		body, err := s.fetchWithRetry(ctx, searchURL)
		if err != nil {
			if page == 1 {
				return nil, fmt.Errorf("failed to fetch search page %d: %w", page, err)
			}
			break
		}

		var resp struct {
			Products []interface{} `json:"products"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			if page == 1 {
				return nil, fmt.Errorf("invalid response on page %d: %w", page, err)
			}
			break
		}
		if len(resp.Products) == 0 {
			break
		}

		resource := domain.Resource{
			Name:         fmt.Sprintf("page_%d.json", page),
			URL:          searchURL,
			ResponseBody: body,
			StatusCode:   http.StatusOK,
			Status:       "200 OK",
			Timestamp:    time.Now(),
		}
		resources = append(resources, resource)
	}
	if len(resources) == 0 {
		return nil, fmt.Errorf("no search results found for query %s", query)
	}
	return resources, nil
}

func (s *Scraper) scrapeCategory(ctx context.Context, task domain.ScrapeTask) (*domain.ScrapeResult, error) {
    body, err := s.fetchWithRetry(ctx, task.URL)
    if err != nil {
        return nil, fmt.Errorf("fetch category page: %w", err)
    }
    resource := domain.Resource{
        Name:         "html",
        URL:          task.URL,
        ResponseBody: body,
        StatusCode:   http.StatusOK,
        Status:       "200 OK",
        Timestamp:    time.Now(),
    }
    return &domain.ScrapeResult{Resources: []domain.Resource{resource}}, nil
}

// BuildDiscoverySearchURL substitutes {query} and {page} in the discovery API template.
func BuildDiscoverySearchURL(urlTemplate, query string, page int) string {
	searchURL := strings.ReplaceAll(urlTemplate, "{query}", url.QueryEscape(query))
	return strings.ReplaceAll(searchURL, "{page}", strconv.Itoa(page))
}
