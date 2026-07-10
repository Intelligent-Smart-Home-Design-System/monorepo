package wildberries

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/go-rod/rod/lib/launcher"
	"github.com/rs/zerolog"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/netproxy"
)

// BrowserProfileDir is the dedicated Chrome user-data directory for WB user-mode scraping.
func BrowserProfileDir() string {
	return wbBrowserProfileDir()
}

func wbBrowserProfileDir() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "rod", "wildberries-chrome")
}

func newBrowserLauncher(log zerolog.Logger, userMode bool, proxyURL, profileDir string) *launcher.Launcher {
	var l *launcher.Launcher
	if userMode {
		profile := profileDir
		if profile == "" {
			profile = wbBrowserProfileDir()
		}
		log.Info().Str("profile", profile).Msg("wb browser: using system Chrome (dedicated profile)")
		// Kill stale Chrome still bound to this profile (e.g. after wbsession) so a new
		// --proxy-server forwarder port is picked up instead of a dead local tunnel.
		k := launcher.New().UserDataDir(profile)
		if path, ok := launcher.LookPath(); ok {
			k = k.Bin(path)
		}
		k.Kill()
		l = launcher.New().
			UserDataDir(profile).
			Set("disable-blink-features", "AutomationControlled").
			Set("lang", "ru-RU").
			Set("window-size", "1920,1080")
		if path, ok := launcher.LookPath(); ok {
			l = l.Bin(path)
		}
	} else {
		log.Info().Msg("wb browser: using headless Chrome")
		l = launcher.New().
			Headless(true).
			Set("no-sandbox").
			Set("disable-setuid-sandbox").
			Set("disable-blink-features", "AutomationControlled").
			Set("headless", "new").
			Set("lang", "ru-RU").
			Set("window-size", "1920,1080")
		if path, ok := launcher.LookPath(); ok {
			l = l.Bin(path)
		}
	}
	return applyBrowserProxy(log, l, proxyURL)
}

func applyBrowserProxy(log zerolog.Logger, l *launcher.Launcher, proxyURL string) *launcher.Launcher {
	if proxyURL == "" {
		return l
	}
	proxyServer, err := netproxy.BrowserProxyServer(proxyURL)
	if err != nil {
		log.Warn().Err(err).Str("proxy", netproxy.RedactURL(proxyURL)).Msg("wb browser: invalid proxy URL")
		return l
	}
	if proxyServer == "" {
		return l
	}
	log.Info().Str("proxy", netproxy.RedactURL(proxyURL)).Str("chrome_proxy", proxyServer).Msg("wb browser: using proxy")
	return l.Set("proxy-server", proxyServer)
}

func defaultBrowserUserMode(cfgValue *bool) bool {
	if cfgValue != nil {
		return *cfgValue
	}
	if v := os.Getenv("WB_BROWSER_USER_MODE"); v == "1" || v == "true" {
		return true
	}
	if v := os.Getenv("WB_BROWSER_USER_MODE"); v == "0" || v == "false" {
		return false
	}
	// Headless token is often rejected by __internal APIs on Windows/Linux dev machines.
	return runtime.GOOS == "windows" || runtime.GOOS == "darwin"
}
