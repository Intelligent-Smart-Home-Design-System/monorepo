package dns

import (
	"os"
	"runtime"

	"github.com/go-rod/rod/lib/launcher"
)

func (s *Scraper) newBrowserLauncher() *launcher.Launcher {
	if s.browserUserMode {
		s.log.Info().Msg("dns browser: using system Chrome (user profile)")
		return launcher.NewUserMode()
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
	return l
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
