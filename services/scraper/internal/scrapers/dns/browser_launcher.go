package dns

import (
	"os"
	"runtime"

	"github.com/go-rod/rod/lib/launcher"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/netproxy"
)

func isContainerRuntime() bool {
	_, err := os.Stat("/.dockerenv")
	return err == nil
}

func containerChromiumBin() string {
	for _, p := range []string{
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"/usr/bin/google-chrome",
		"/usr/bin/google-chrome-stable",
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func (s *Scraper) newBrowserLauncher() *launcher.Launcher {
	if s.forceBrowser {
		profile := resolveBrowserProfileDir()
		mode := "user"
		if isContainerRuntime() {
			mode = "user-xvfb"
		}
		s.log.Info().
			Str("profile", profile).
			Str("mode", mode).
			Bool("isolated", profile != dnsBrowserSharedProfileDir()).
			Msg("dns browser: using Chrome profile")
		// Chrome locks user-data-dir; kill stale instance (e.g. previous pipeline job).
		k := launcher.New().UserDataDir(profile)
		if isContainerRuntime() {
			if bin := containerChromiumBin(); bin != "" {
				k = k.Bin(bin)
			}
		} else if path, ok := launcher.LookPath(); ok {
			k = k.Bin(path)
		}
		k.Kill()
		l := launcher.New().
			UserDataDir(profile).
			Set("disable-blink-features", "AutomationControlled").
			Set("lang", "ru-RU").
			Set("window-size", "1920,1080")
		if isContainerRuntime() {
			l = l.
				Set("no-sandbox", "").
				Set("disable-dev-shm-usage", "").
				Set("disable-gpu", "")
			if bin := containerChromiumBin(); bin != "" {
				l = l.Bin(bin)
			}
		} else if path, ok := launcher.LookPath(); ok {
			l = l.Bin(path)
		}
		return s.applyBrowserProxy(l)
	}

	s.log.Info().Msg("dns browser: using headless Chrome")
	l := launcher.New().
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
	return s.applyBrowserProxy(l)
}

func (s *Scraper) applyBrowserProxy(l *launcher.Launcher) *launcher.Launcher {
	if s.proxyURL == "" {
		return l
	}
	proxyServer, err := netproxy.BrowserProxyServer(s.proxyURL)
	if err != nil {
		s.log.Warn().Err(err).Str("proxy", netproxy.RedactURL(s.proxyURL)).Msg("dns browser: invalid proxy URL")
		return l
	}
	if proxyServer == "" {
		return l
	}
	s.log.Info().Str("proxy", netproxy.RedactURL(s.proxyURL)).Str("chrome_proxy", proxyServer).Msg("dns browser: using proxy")
	return l.Set("proxy-server", proxyServer)
}

func defaultBrowserUserMode(cfgValue *bool) bool {
	if cfgValue != nil {
		return *cfgValue
	}
	if v := os.Getenv("DNS_BROWSER_USER_MODE"); v == "1" || v == "true" {
		return true
	}
	// Headless Chromium is often blocked by Qrator; real Chrome profile works on dev machines.
	return runtime.GOOS == "darwin"
}
