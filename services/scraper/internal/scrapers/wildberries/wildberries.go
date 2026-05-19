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
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/stealth"
	"golang.org/x/time/rate"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
)

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
	limiter              *rate.Limiter
	cardBasket           string
	sessionPath          string
	session              *Session
	mu                   sync.RWMutex
	discoveryURLTemplate string
	discoveryMaxPages    int
}

func NewScraper(
	timeout time.Duration,
	proxyURL, cardBasket string,
	rps float64,
	sessionPath string,
	discoveryURLTemplate string,
	discoveryMaxPages int,
) *Scraper {
	transport := &http.Transport{}
	if proxyURL != "" {
		if proxy, err := url.Parse(proxyURL); err == nil {
			transport.Proxy = http.ProxyURL(proxy)
		}
	}
	limiter := rate.NewLimiter(rate.Limit(rps), 1)
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
		limiter:              limiter,
		cardBasket:           cardBasket,
		sessionPath:          sessionPath,
		session:              session,
		discoveryURLTemplate: discoveryURLTemplate,
		discoveryMaxPages:    discoveryMaxPages,
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

func (s *Scraper) mineSession() (*Session, error) {
	fmt.Println("[DEBUG] mineSession: starting headless browser...")
	l := launcher.New().Headless(true).Set("no-sandbox").Set("disable-setuid-sandbox")
	url, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("launch browser: %w", err)
	}
	browser := rod.New().ControlURL(url).MustConnect()
	defer browser.MustClose()

	page := stealth.MustPage(browser)
	defer page.MustClose()

	page.MustNavigate("https://www.wildberries.ru/")
	page.MustWaitIdle()
	time.Sleep(5 * time.Second)

	cookies := page.MustCookies()
	var cookieList []Cookie
	var tokenValue string
	for _, c := range cookies {
		if c.Name == "x_wbaas_token" {
			tokenValue = c.Value
		}
		cookieList = append(cookieList, Cookie{
			Name:   c.Name,
			Value:  c.Value,
			Domain: c.Domain,
			Path:   c.Path,
		})
	}
	fmt.Println(cookieList)
	if tokenValue == "" {
		return nil, fmt.Errorf("x_wbaas_token not found in cookies")
	}

	uaVal := page.MustEval(`() => navigator.userAgent`)
	userAgent := uaVal.Str()

	return &Session{
		UserAgent: userAgent,
		Cookies:   cookieList,
		Token:     tokenValue,
	}, nil
}

func (s *Scraper) ensureSession() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.session != nil && time.Since(s.session.UpdatedAt) < 30*time.Minute {
		return nil
	}

	_, err := s.loadSession()
	if err == nil && s.session != nil && time.Since(s.session.UpdatedAt) < 30*time.Minute {
		return nil
	}

	fmt.Println("[DEBUG] ensureSession: session not found or expired, mining new session...")
	sess, err := s.mineSession()
	if err != nil {
		return fmt.Errorf("mine session: %w", err)
	}
	sess.UpdatedAt = time.Now()
	s.session = sess
	if err := s.saveSession(sess); err != nil {
		fmt.Printf("[WARN] failed to save session: %v\n", err)
	}
	return nil
}

func (s *Scraper) fetchJSON(ctx context.Context, url string) ([]byte, error) {
	fmt.Printf("[DEBUG] fetchJSON: start %s\n", url)
	if err := s.ensureSession(); err != nil {
		fmt.Printf("[DEBUG] fetchJSON: ensureSession error: %v\n", err)
		return nil, err
	}
	fmt.Printf("[DEBUG] fetchJSON: session ok, token len=%d\n", len(s.session.Token))
	if err := s.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	req.Header.Set("User-Agent", s.session.UserAgent)
	for _, c := range s.session.Cookies {
		req.AddCookie(&http.Cookie{
			Name:   c.Name,
			Value:  c.Value,
			Domain: c.Domain,
			Path:   c.Path,
		})
	}
	s.mu.RUnlock()

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
	fmt.Printf("[DEBUG] fetchJSON: %s, body size = %d\n", url, len(body))
	return body, nil
}

func (s *Scraper) fetchWithRetry(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	for i := 0; i < 3; i++ {
		fmt.Printf("[DEBUG] fetchWithRetry: attempt %d for %s\n", i+1, url)
		body, err := s.fetchJSON(ctx, url)
		if err == nil {
			fmt.Printf("[DEBUG] fetchWithRetry: success\n")
			return body, nil
		}
		fmt.Printf("[DEBUG] fetchWithRetry: error: %v\n", err)
		lastErr = err
		if strings.Contains(err.Error(), "session invalid") {
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
		fmt.Println("[DEBUG] urlExists: session is nil")
		return false
	}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		fmt.Printf("[DEBUG] urlExists: request error %v\n", err)
		return false
	}
	req.Header.Set("User-Agent", s.session.UserAgent)
	req.Header.Set("Range", "bytes=0-0")
	resp, err := s.client.Do(req)
	if err != nil {
		fmt.Printf("[DEBUG] urlExists: do error %v\n", err)
		return false
	}
	defer resp.Body.Close()
	fmt.Printf("[DEBUG] urlExists: status %d for %s\n", resp.StatusCode, url)
	return resp.StatusCode == http.StatusOK
}

func (s *Scraper) getCardURL(ctx context.Context, nmID int) (string, error) {
	newURL := buildCardURLNew("01", nmID)
	fmt.Printf("[DEBUG] checking new CDN: %s\n", newURL)
	if s.urlExists(ctx, newURL) {
		fmt.Println("[DEBUG] new CDN works")
		return newURL, nil
	}
	vol := nmID / 100000
	part := nmID / 1000
	for basket := 1; basket <= 41; basket++ {
		oldURL := fmt.Sprintf("https://basket-%d.wbbasket.ru/vol%d/part%d/%d/info/ru/card.json", basket, vol, part, nmID)
		fmt.Printf("[DEBUG] checking basket %d: %s\n", basket, oldURL)
		if s.urlExists(ctx, oldURL) {
			fmt.Printf("[DEBUG] found working basket %d\n", basket)
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
	if err := s.ensureSession(); err != nil {
		return nil, fmt.Errorf("ensure session: %w", err)
	}

	switch task.PageType {
	case domain.PageTypeListing:
		return s.scrapeListing(ctx, task)
	case domain.PageTypeDiscovery:
		return s.scrapeDiscoveryTask(ctx, task)
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
		searchURL := strings.ReplaceAll(urlTemplate, "{query}", url.QueryEscape(query))
		searchURL = strings.ReplaceAll(searchURL, "{page}", strconv.Itoa(page))

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
