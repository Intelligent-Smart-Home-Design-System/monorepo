// Quick proxy check — HTTP GET via scraping.proxy / SCRAPER_SCRAPING_PROXY.
//
//	go run ./cmd/proxycheck
//	go run ./cmd/proxycheck -config cmd/scraper/config.wb-smoke.toml
package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/netproxy"
)

func main() {
	configFile := flag.String("config", "", "optional config (scraping.proxy + env override)")
	flag.Parse()

	proxyURL := strings.TrimSpace(os.Getenv("SCRAPER_SCRAPING_PROXY"))
	if *configFile != "" {
		cfg, err := config.LoadFile(*configFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "config: %v\n", err)
			os.Exit(1)
		}
		if cfg.Scraping.Proxy != "" {
			proxyURL = cfg.Scraping.Proxy
		}
	}
	if proxyURL == "" {
		fmt.Fprintln(os.Stderr, "proxy empty — set SCRAPER_SCRAPING_PROXY=http://LOGIN:PASSWORD@host:PORT")
		os.Exit(1)
	}
	if err := netproxy.ValidateProxyURL(proxyURL); err != nil {
		fmt.Fprintf(os.Stderr, "proxy: %v\n", err)
		os.Exit(1)
	}

	client, err := netproxy.NewHTTPClient(30*time.Second, proxyURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "client: %v\n", err)
		os.Exit(1)
	}

	const checkURL = "https://api.ipify.org?format=json"
	resp, err := client.Get(checkURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: HTTP via proxy: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "FAIL: HTTP %d: %s\n", resp.StatusCode, strings.TrimSpace(string(body)))
		os.Exit(1)
	}
	fmt.Printf("OK: proxy works, exit IP response: %s\n", strings.TrimSpace(string(body)))
	fmt.Printf("    upstream: %s\n", netproxy.RedactURL(proxyURL))
}
