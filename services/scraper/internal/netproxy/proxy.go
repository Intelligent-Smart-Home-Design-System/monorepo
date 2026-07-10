package netproxy

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ChromeProxyServer returns a value for Chrome's --proxy-server flag.
// Credentials in the URL are omitted (Chrome does not accept them in this flag).
func ChromeProxyServer(proxyURL string) (string, error) {
	proxyURL = strings.TrimSpace(proxyURL)
	if proxyURL == "" {
		return "", nil
	}
	if !strings.Contains(proxyURL, "://") {
		proxyURL = "http://" + proxyURL
	}

	u, err := url.Parse(proxyURL)
	if err != nil {
		return "", fmt.Errorf("parse proxy URL: %w", err)
	}

	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		if u.Host == "" {
			return "", fmt.Errorf("proxy URL missing host")
		}
		return "http://" + u.Host, nil
	case "socks5", "socks5h":
		if u.Host == "" {
			return "", fmt.Errorf("proxy URL missing host")
		}
		return "socks5://" + u.Host, nil
	default:
		return "", fmt.Errorf("unsupported proxy scheme %q", u.Scheme)
	}
}

var placeholderUsers = map[string]bool{
	"user": true, "username": true, "login": true,
}
var placeholderPasswords = map[string]bool{
	"pass": true, "password": true, "pwd": true,
}

// ValidateProxyURL rejects documentation placeholders (USER:PASS) and empty hosts.
func ValidateProxyURL(proxyURL string) error {
	proxyURL = strings.TrimSpace(proxyURL)
	if proxyURL == "" {
		return fmt.Errorf("proxy URL is empty — set SCRAPER_SCRAPING_PROXY with login and password from your provider panel (not the USER:PASS placeholder from docs)")
	}
	if !strings.Contains(proxyURL, "://") {
		proxyURL = "http://" + proxyURL
	}
	u, err := url.Parse(proxyURL)
	if err != nil {
		return fmt.Errorf("parse proxy URL: %w", err)
	}
	if u.Host == "" {
		return fmt.Errorf("proxy URL missing host")
	}
	if u.User == nil {
		return fmt.Errorf("proxy URL missing login:password — mobile proxies require auth, e.g. http://LOGIN:PASSWORD@host:PORT")
	}
	user := strings.ToLower(u.User.Username())
	pass, _ := u.User.Password()
	passLower := strings.ToLower(pass)
	if placeholderUsers[user] || user == "" {
		return fmt.Errorf("proxy login looks like a docs placeholder %q — paste real credentials from your provider (Megafon panel), not USER from examples", u.User.Username())
	}
	if placeholderPasswords[passLower] || pass == "" {
		return fmt.Errorf("proxy password looks like a docs placeholder — paste the real password from your provider panel, not PASS from examples")
	}
	if strings.Contains(user, "xxxxx") || strings.Contains(passLower, "xxxxx") {
		return fmt.Errorf("proxy URL still contains XXXXX placeholder — use real login/password from provider")
	}
	return nil
}

// RedactURL returns proxy URL safe for logs (password hidden).
func RedactURL(proxyURL string) string {
	proxyURL = strings.TrimSpace(proxyURL)
	if proxyURL == "" {
		return ""
	}
	u, err := url.Parse(proxyURL)
	if err != nil || u.User == nil {
		return proxyURL
	}
	if pass, ok := u.User.Password(); ok && pass != "" {
		return strings.Replace(proxyURL, pass, "****", 1)
	}
	return proxyURL
}

// ConfigureTransport sets http.ProxyURL on transport when proxyURL is non-empty.
func ConfigureTransport(transport *http.Transport, proxyURL string) error {
	proxyURL = strings.TrimSpace(proxyURL)
	if proxyURL == "" {
		return nil
	}
	if !strings.Contains(proxyURL, "://") {
		proxyURL = "http://" + proxyURL
	}
	proxy, err := url.Parse(proxyURL)
	if err != nil {
		return fmt.Errorf("parse proxy URL: %w", err)
	}
	transport.Proxy = http.ProxyURL(proxy)
	return nil
}

// NewHTTPClient builds an http.Client with optional proxy.
func NewHTTPClient(timeout time.Duration, proxyURL string) (*http.Client, error) {
	transport := &http.Transport{}
	if err := ConfigureTransport(transport, proxyURL); err != nil {
		return nil, err
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}, nil
}
