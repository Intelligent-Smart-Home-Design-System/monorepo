package sources

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
)

func scraperModuleRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

// maybeDumpSmokeHTML saves fetched HTML when SMOKE_DUMP_HTML=1 (optional SMOKE_DUMP_DIR).
func maybeDumpSmokeHTML(t *testing.T, label string, data []byte) {
	t.Helper()
	if os.Getenv("SMOKE_DUMP_HTML") != "1" {
		return
	}
	dir := os.Getenv("SMOKE_DUMP_DIR")
	if dir == "" {
		dir = filepath.Join("testdata", "smoke-dump")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Logf("dump html: mkdir %s: %v", dir, err)
		return
	}
	name := fmt.Sprintf("%s-%s.html", label, time.Now().Format("20060102-150405"))
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Logf("dump html: write %s: %v", path, err)
		return
	}
	t.Logf("dumped HTML to %s (%d bytes)", path, len(data))
}

// applyLiveNetworkConfig merges proxy/user-agent from env into cfg (DNS smoke; WB uses config file).
func applyLiveNetworkConfig(cfg *config.Config) {
	if p := os.Getenv("SCRAPER_SCRAPING_PROXY"); p != "" {
		cfg.Scraping.Proxy = p
	}
	if ua := os.Getenv("SCRAPER_SCRAPING_USER_AGENT"); ua != "" {
		cfg.Scraping.UserAgent = ua
	}
	if cfg.Dns.UserAgent == "" && cfg.Scraping.UserAgent != "" {
		cfg.Dns.UserAgent = cfg.Scraping.UserAgent
	}
}
