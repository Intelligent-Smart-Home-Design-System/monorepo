package netproxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrowserProxyServer_NoAuth(t *testing.T) {
	t.Parallel()
	got, err := BrowserProxyServer("http://proxy.example:8080")
	require.NoError(t, err)
	assert.Equal(t, "http://proxy.example:8080", got)
}

func TestBrowserProxyServer_WithAuth_StartsLocalForwarder(t *testing.T) {
	// Uses a non-routable upstream; we only verify local forwarder URL is returned.
	got, err := BrowserProxyServer("http://user:pass@127.0.0.1:9")
	require.NoError(t, err)
	assert.Contains(t, got, "http://127.0.0.1:")
}

func TestBrowserProxyServer_ReusesForwarder(t *testing.T) {
	in := "http://user:pass@127.0.0.1:9"
	a, err := BrowserProxyServer(in)
	require.NoError(t, err)
	b, err := BrowserProxyServer(in)
	require.NoError(t, err)
	assert.Equal(t, a, b)
}
