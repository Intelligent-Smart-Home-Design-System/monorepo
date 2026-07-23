// WB smoke micro-steps. Run from services/scraper:
//   .\scripts\test-wb-smoke.ps1 -E2E
//
// Artifacts: testdata/wb-smoke-dump/step*.html|json

package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/netproxy"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parser"
	wbParser "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/parsers/wildberries"
	wbScraper "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/scrapers/wildberries"
)

func TestWBStep01_Config(t *testing.T) {
	if testing.Short() {
		t.Skip("skipped in -short mode")
	}
	cfg, path := loadWBSmokeConfig(t)
	logWBConfig(t, cfg, path)
	saveWBArtifact(t, "step01-config.json", mustJSON(cfg))
}

func TestWBStep02_SessionFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipped in -short mode")
	}
	cfg, _ := loadWBSmokeConfig(t)
	meta := requireSessionFile(t, cfg)
	t.Logf("session age: %s", time.Since(meta.UpdatedAt).Round(time.Second))
	t.Logf("token length: %d", len(meta.Token))
	t.Logf("cookies: %d", len(meta.Cookies))
	if age := time.Since(meta.UpdatedAt); age > cfg.Scraping.WBSessionMaxAge {
		t.Fatalf("session expired (%s ago, max %s) — run: go run ./cmd/wbsession -config cmd/scraper/config.wb-smoke.toml",
			age.Round(time.Second), cfg.Scraping.WBSessionMaxAge)
	}
	raw, _ := os.ReadFile(cfg.Scraping.WBSessionPath)
	saveWBArtifact(t, "step02-session.json", raw)
}

func TestWBStep03_CategoryHTTP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipped in -short mode")
	}
	cfg, _ := loadWBSmokeConfig(t)
	requireSessionFile(t, cfg)
	src := newWBSource(t, cfg)
	defer closeWBSource(t, src)
	runWBCategoryStep(t, cfg, src)
}

func TestWBStep04_BrowserWarmup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipped in -short mode")
	}
	cfg, _ := loadWBSmokeConfig(t)
	requireSessionFile(t, cfg)
	src := newWBSource(t, cfg)
	defer closeWBSource(t, src)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	require.NoError(t, src.Warmup(ctx))
	saveWBArtifact(t, "step04-warmup-ok.txt", []byte("browser warmup OK\n"))
	t.Log("browser warmup OK")
}

func TestWBStep05_DiscoveryAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipped in -short mode")
	}
	cfg, _ := loadWBSmokeConfig(t)
	requireSessionFile(t, cfg)
	src := newWBSource(t, cfg)
	defer closeWBSource(t, src)
	runWBDiscoveryStep(t, cfg, src)
}

// E2E: one Chrome instance, artifacts at every step.
func TestWildberriesSmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skipped in -short mode")
	}
	cfg, path := loadWBSmokeConfig(t)
	logWBConfig(t, cfg, path)
	saveWBArtifact(t, "step01-config.json", mustJSON(cfg))

	meta := requireSessionFile(t, cfg)
	t.Logf("session age: %s, token=%d chars", time.Since(meta.UpdatedAt).Round(time.Second), len(meta.Token))
	raw, _ := os.ReadFile(cfg.Scraping.WBSessionPath)
	saveWBArtifact(t, "step02-session.json", raw)

	src := newWBSource(t, cfg)
	defer closeWBSource(t, src)

	t.Run("Step05_DiscoveryAPI", func(t *testing.T) {
		runWBDiscoveryStep(t, cfg, src)
	})
	t.Run("Step03_Category", func(t *testing.T) {
		runWBCategoryStep(t, cfg, src)
	})
}

func runWBCategoryStep(t *testing.T, cfg config.Config, src Source) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	result, err := src.Scraper().Scrape(ctx, domain.ScrapeTask{
		Source: domain.SourceWildberries, PageType: domain.PageTypeCategory,
		URL: cfg.Wildberries.Category.CategoryURL,
	})
	require.NoError(t, err)
	require.Len(t, result.Resources, 1)

	html := result.Resources[0].ResponseBody
	saveWBArtifact(t, "step03-category.html", html)

	files := []*parser.ArchiveFile{{Name: "html", Data: html}}
	listings, err := wbParser.NewCategoryParser().Parse(0, files)
	if err != nil {
		t.Logf("category parser: %v — HTML size check (JS category OK)", err)
		assert.Greater(t, len(html), 5_000, "category HTML too small — likely block page")
		return
	}
	t.Logf("category listings parsed: %d", len(listings))
}

func runWBDiscoveryStep(t *testing.T, cfg config.Config, src Source) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()

	query := cfg.Wildberries.Discovery.DiscoveryTextQueries[0]
	result, err := src.Scraper().Scrape(ctx, domain.ScrapeTask{
		Source: domain.SourceWildberries, PageType: domain.PageTypeDiscovery,
		URL: "wildberries://discovery/" + query,
	})
	if err != nil {
		saveWBArtifact(t, "step05-discovery-error.txt", []byte(err.Error()))
		require.NoError(t, err, "discovery API failed — see testdata/wb-smoke-dump/step05-discovery-error.txt")
	}

	for i, res := range result.Resources {
		name := fmt.Sprintf("step05-discovery-%s", res.Name)
		if filepath.Ext(res.Name) == "" {
			name += ".json"
		}
		saveWBArtifact(t, name, res.ResponseBody)
		t.Logf("discovery resource %d: %s (%d bytes)", i, res.Name, len(res.ResponseBody))
	}

	files := make([]*parser.ArchiveFile, 0, len(result.Resources))
	for _, res := range result.Resources {
		files = append(files, &parser.ArchiveFile{Name: res.Name, Data: res.ResponseBody})
	}
	listings, err := wbParser.NewDiscoveryParser().Parse(0, files)
	require.NoError(t, err)
	require.NotEmpty(t, listings, "discovery returned no listing URLs")
	t.Logf("discovery: %d page(s), %d listing URL(s)", len(result.Resources), len(listings))
}

func logWBConfig(t *testing.T, cfg config.Config, path string) {
	t.Helper()
	t.Logf("config file: %s", path)
	t.Logf("wb_session_path: %s", cfg.Scraping.WBSessionPath)
	t.Logf("browser_profile: %s", wbBrowserProfileLabel(cfg))
	t.Logf("proxy: %s", netproxy.RedactURL(cfg.Scraping.Proxy))
	t.Logf("user_agent: %s", cfg.Scraping.UserAgent)
	t.Logf("category_url: %s", cfg.Wildberries.Category.CategoryURL)
	t.Logf("dump dir: %s", wbSmokeDumpDir())
}

func loadWBSmokeConfig(t *testing.T) (config.Config, string) {
	t.Helper()
	path := wbSmokeConfigPath()
	cfg, err := config.LoadFile(path)
	require.NoError(t, err, "load config %s", path)
	if cfg.Scraping.Proxy == "" {
		t.Skip("scraping.proxy empty — set SCRAPER_SCRAPING_PROXY")
	}
	return cfg, path
}

func wbSmokeConfigPath() string {
	if p := os.Getenv("SCRAPER_CONFIG"); p != "" {
		return p
	}
	return filepath.Join(scraperModuleRoot(), "cmd", "scraper", "config.wb-smoke.toml")
}

func wbSmokeDumpDir() string {
	return filepath.Join(scraperModuleRoot(), "testdata", "wb-smoke-dump")
}

func saveWBArtifact(t *testing.T, name string, data []byte) {
	t.Helper()
	dir := wbSmokeDumpDir()
	require.NoError(t, os.MkdirAll(dir, 0o755))
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, data, 0o644))
	t.Logf("artifact saved: %s (%d bytes)", path, len(data))
}

func mustJSON(v any) []byte {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	return b
}

func newWBSource(t *testing.T, cfg config.Config) Source {
	t.Helper()
	registry, err := NewRegistry(cfg, zerolog.Nop())
	require.NoError(t, err)
	src := registry[domain.SourceWildberries]
	require.NotNil(t, src)
	return src
}

func closeWBSource(t *testing.T, src Source) {
	t.Helper()
	if c, ok := src.(interface{ Close() }); ok {
		c.Close()
	}
}

func wbBrowserProfileLabel(cfg config.Config) string {
	if cfg.Wildberries.BrowserProfileDir != "" {
		return cfg.Wildberries.BrowserProfileDir
	}
	return wbScraper.BrowserProfileDir()
}

type wbSessionMeta struct {
	UpdatedAt time.Time
	Token     string
	Cookies   []struct {
		Name string `json:"name"`
	}
}

func requireSessionFile(t *testing.T, cfg config.Config) wbSessionMeta {
	t.Helper()
	path := cfg.Scraping.WBSessionPath
	if path == "" {
		t.Fatal("wb_session_path empty in config")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("session file missing: %s\nrun: go run ./cmd/wbsession -config cmd/scraper/config.wb-smoke.toml", path)
	}
	var raw wbSessionMeta
	if err := json.Unmarshal(data, &raw); err != nil || raw.UpdatedAt.IsZero() {
		t.Fatalf("invalid session file: %s", path)
	}
	return raw
}
