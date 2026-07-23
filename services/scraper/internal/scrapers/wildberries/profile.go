package wildberries

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func wbBrowserSharedProfileDir() string {
	return wbBrowserProfileDir()
}

var (
	wbIsolatedProfileOnce sync.Once
	wbIsolatedProfilePath string
)

func wbBrowserIsolatedProfileDir() string {
	wbIsolatedProfileOnce.Do(func() {
		dir, err := os.MkdirTemp("", "wildberries-chrome-")
		if err != nil {
			dir = filepath.Join(os.TempDir(), "rod", fmt.Sprintf("wildberries-chrome-%d-%d", os.Getpid(), time.Now().UnixNano()))
			_ = os.MkdirAll(dir, 0o700)
		}
		wbIsolatedProfilePath = dir
	})
	return wbIsolatedProfilePath
}

func wbEnvTruthy(key string) bool {
	v := strings.TrimSpace(os.Getenv(key))
	return v == "1" || strings.EqualFold(v, "true") || v == "yes"
}

// resolveWBBrowserProfile picks Chrome user-data dir for this scraper process.
//
// Priority:
//  1. config browser_profile_dir (non-empty)
//  2. WB_BROWSER_PROFILE env
//  3. WB_BROWSER_SHARED_PROFILE=1 or default on host — %LOCALAPPDATA%/rod/wildberries-chrome
//  4. WB_BROWSER_ISOLATE_PROFILE=1 — unique temp dir (parallel tests / no SingletonLock)
func resolveWBBrowserProfile(configDir string) string {
	if configDir != "" {
		return configDir
	}
	if p := strings.TrimSpace(os.Getenv("WB_BROWSER_PROFILE")); p != "" {
		return p
	}
	if wbEnvTruthy("WB_BROWSER_ISOLATE_PROFILE") {
		return wbBrowserIsolatedProfileDir()
	}
	return wbBrowserSharedProfileDir()
}
