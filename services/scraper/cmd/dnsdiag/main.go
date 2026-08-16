package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/viper"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	dnsscraper "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scrapers/dns"
)

const defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

func main() {
	configFile := flag.String("config", "", "config file (scraping.proxy, dns.browser_user_mode)")
	proxyFlag := flag.String("proxy", "", "HTTP/SOCKS proxy URL (overrides config)")
	flag.Parse()

	cfg, err := resolveConfig(*configFile, *proxyFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	log := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()

	userAgent := cfg.Dns.UserAgent
	if userAgent == "" {
		userAgent = cfg.Scraping.UserAgent
	}
	if userAgent == "" {
		userAgent = defaultUserAgent
	}

	timeout := cfg.Scraping.Timeout
	if timeout <= 0 {
		timeout = 90 * time.Second
	}

	s := dnsscraper.NewScraper(log, timeout, cfg.Scraping.Proxy, userAgent, cfg.Dns.BrowserUserMode)

	targetURL := "https://www.dns-shop.ru/catalog/9bbe8fe270e7c3ae/umnaa-tehnika/"
	if len(flag.Args()) > 0 {
		targetURL = flag.Args()[0]
	}

	res, err := s.Scrape(context.Background(), domain.ScrapeTask{URL: targetURL})
	if err != nil {
		fmt.Println("ERR:", err)
		os.Exit(1)
	}
	html := res.Resources[0].ResponseBody
	fmt.Printf("OK status=%d bytes=%d\n", res.Resources[0].StatusCode, len(html))
	text := string(html)
	hasSub := contains(text, "subcategory__item")
	hasProd := contains(text, "catalog-product__image-link")
	fmt.Printf("hub=%v grid=%v title_snip=%q\n", hasSub, hasProd, snippet(text, 120))
}

func resolveConfig(configFile, proxyFlag string) (config.Config, error) {
	var cfg config.Config

	if configFile != "" {
		if err := loadConfig(configFile, &cfg); err != nil {
			return cfg, err
		}
	}

	if p := strings.TrimSpace(proxyFlag); p != "" {
		cfg.Scraping.Proxy = p
	}
	if p := strings.TrimSpace(os.Getenv("SCRAPER_SCRAPING_PROXY")); p != "" {
		cfg.Scraping.Proxy = p
	}

	if v := os.Getenv("DNS_BROWSER_USER_MODE"); v == "1" || v == "true" {
		trueVal := true
		cfg.Dns.BrowserUserMode = &trueVal
	} else if v == "0" || v == "false" {
		falseVal := false
		cfg.Dns.BrowserUserMode = &falseVal
	}

	return cfg, nil
}

func loadConfig(cfgFile string, cfg *config.Config) error {
	viper.SetConfigFile(cfgFile)
	viper.SetEnvPrefix("SCRAPER")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		return err
	}
	return viper.Unmarshal(cfg)
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func snippet(s string, n int) string {
	if i := indexOf(s, "<title"); i >= 0 {
		end := indexOf(s[i:], "</title>")
		if end > 0 {
			t := s[i : i+end+8]
			if len(t) > n {
				return t[:n]
			}
			return t
		}
	}
	if len(s) > n {
		return s[:n]
	}
	return s
}
