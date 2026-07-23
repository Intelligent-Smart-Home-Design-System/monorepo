// Usage:
//
//	cd services/scraper
//	$env:SCRAPER_SCRAPING_PROXY = "http://USER:PASS@host:PORT"
//	go run ./cmd/wbsession -config cmd/scraper/config.wb-smoke.toml
//
// Paths and browser settings: cmd/scraper/config.wb-smoke.toml
// Opens visible Chrome (profile from config or %LOCALAPPDATA%\rod\wildberries-chrome).
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/netproxy"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scrapers/wildberries"
)

func main() {
	configFile := flag.String("config", "cmd/scraper/config.wb-smoke.toml", "config file (paths, proxy, browser)")
	out := flag.String("out", "", "override wb_session_path from config")
	proxyFlag := flag.String("proxy", "", "HTTP/SOCKS proxy URL (overrides config and env)")
	timeout := flag.Duration("timeout", 10*time.Minute, "browser mining timeout (include captcha time)")
	headless := flag.Bool("headless", false, "use headless Chrome (often blocked by WB APIs)")
	userMode := flag.Bool("user-mode", true, "use system Chrome with dedicated profile (recommended)")
	flag.Parse()

	cfg, err := config.LoadFile(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	outPath := cfg.Scraping.WBSessionPath
	if *out != "" {
		outPath = *out
	}
	if outPath == "" {
		fmt.Fprintln(os.Stderr, "wb_session_path empty — set in config or -out")
		os.Exit(1)
	}

	proxyURL := strings.TrimSpace(*proxyFlag)
	if proxyURL == "" {
		proxyURL = cfg.Scraping.Proxy
	}
	if proxyURL == "" {
		proxyURL = strings.TrimSpace(os.Getenv("SCRAPER_SCRAPING_PROXY"))
	}
	if proxyURL == "" {
		fmt.Fprintln(os.Stderr, "proxy empty — set scraping.proxy, SCRAPER_SCRAPING_PROXY, or -proxy")
		os.Exit(1)
	}
	if err := netproxy.ValidateProxyURL(proxyURL); err != nil {
		fmt.Fprintf(os.Stderr, "proxy: %v\n", err)
		os.Exit(1)
	}

	browserUserMode := *userMode && !*headless
	basket := cfg.Scraping.WBCardBasket
	if basket == "" {
		basket = "01"
	}
	rps := cfg.Scraping.WBRPS
	if rps <= 0 {
		rps = 2
	}
	scrapeTimeout := cfg.Scraping.Timeout
	if scrapeTimeout <= 0 {
		scrapeTimeout = time.Minute
	}

	log := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	modeLabel := "user"
	if !browserUserMode {
		modeLabel = "headless"
	}
	log.Info().
		Str("config", *configFile).
		Str("path", outPath).
		Str("mode", modeLabel).
		Str("profile", profileLabel(cfg)).
		Msg("mining WB session...")

	mode := wildberries.NewScraper(
		log,
		scrapeTimeout,
		proxyURL,
		basket,
		rps,
		outPath,
		"",
		1,
		&browserUserMode,
		cfg.Wildberries.BrowserProfileDir,
	)

	if err := mode.RefreshSession(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to mine session")
	}

	log.Info().Str("path", outPath).Msg("session saved")
	fmt.Printf("OK: %s\n", outPath)
}

func profileLabel(cfg config.Config) string {
	if cfg.Wildberries.BrowserProfileDir != "" {
		return cfg.Wildberries.BrowserProfileDir
	}
	return wildberries.BrowserProfileDir()
}
