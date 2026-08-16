package dns

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveBrowserProfileDir_explicit(t *testing.T) {
	t.Setenv("DNS_BROWSER_PROFILE", filepath.Join(t.TempDir(), "custom"))
	t.Setenv("DNS_BROWSER_SHARED_PROFILE", "")
	t.Setenv("DNS_BROWSER_ISOLATE_PROFILE", "")

	got := resolveBrowserProfileDir()
	want := os.Getenv("DNS_BROWSER_PROFILE")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveBrowserProfileDir_isolated(t *testing.T) {
	t.Setenv("DNS_BROWSER_PROFILE", "")
	t.Setenv("DNS_BROWSER_SHARED_PROFILE", "")
	t.Setenv("DNS_BROWSER_ISOLATE_PROFILE", "1")

	got := resolveBrowserProfileDir()
	if got == dnsBrowserSharedProfileDir() {
		t.Fatalf("expected isolated profile, got shared %q", got)
	}
}

func TestResolveBrowserProfileDir_shared(t *testing.T) {
	t.Setenv("DNS_BROWSER_PROFILE", "")
	t.Setenv("DNS_BROWSER_ISOLATE_PROFILE", "")
	t.Setenv("DNS_BROWSER_SHARED_PROFILE", "1")

	got := resolveBrowserProfileDir()
	if got != dnsBrowserSharedProfileDir() {
		t.Fatalf("got %q want shared %q", got, dnsBrowserSharedProfileDir())
	}
}
