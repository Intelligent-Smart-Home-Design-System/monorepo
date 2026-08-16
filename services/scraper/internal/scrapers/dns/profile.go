package dns

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func dnsBrowserSharedProfileDir() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "rod", "dns-shop-chrome")
}

var (
	isolatedProfileOnce sync.Once
	isolatedProfilePath string
)

func dnsBrowserIsolatedProfileDir() string {
	isolatedProfileOnce.Do(func() {
		dir, err := os.MkdirTemp("", "dns-shop-chrome-")
		if err != nil {
			dir = filepath.Join(os.TempDir(), "rod", fmt.Sprintf("dns-shop-chrome-%d-%d", os.Getpid(), time.Now().UnixNano()))
			_ = os.MkdirAll(dir, 0o700)
		}
		isolatedProfilePath = dir
	})
	return isolatedProfilePath
}

func envTruthy(key string) bool {
	v := strings.TrimSpace(os.Getenv(key))
	return v == "1" || strings.EqualFold(v, "true") || v == "yes"
}

// resolveBrowserProfileDir picks a Chrome user-data directory for this scraper process.
//
// Priority:
//  1. DNS_BROWSER_PROFILE — explicit path
//  2. DNS_BROWSER_SHARED_PROFILE=1 — persistent cache (dev warmup)
//  3. Docker / DNS_BROWSER_ISOLATE_PROFILE=1 — unique temp dir (no SingletonLock conflicts)
//  4. Host default — shared cache dir
func resolveBrowserProfileDir() string {
	if p := strings.TrimSpace(os.Getenv("DNS_BROWSER_PROFILE")); p != "" {
		return p
	}
	if envTruthy("DNS_BROWSER_SHARED_PROFILE") {
		return dnsBrowserSharedProfileDir()
	}
	if isContainerRuntime() || envTruthy("DNS_BROWSER_ISOLATE_PROFILE") {
		return dnsBrowserIsolatedProfileDir()
	}
	return dnsBrowserSharedProfileDir()
}
