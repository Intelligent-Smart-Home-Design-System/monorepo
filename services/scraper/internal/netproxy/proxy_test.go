package netproxy

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChromeProxyServer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"http://proxy.example:8080", "http://proxy.example:8080"},
		{"http://user:pass@proxy.example:8080", "http://proxy.example:8080"},
		{"socks5://127.0.0.1:1080", "socks5://127.0.0.1:1080"},
		{"127.0.0.1:3128", "http://127.0.0.1:3128"},
	}

	for _, tt := range tests {
		got, err := ChromeProxyServer(tt.in)
		require.NoError(t, err)
		assert.Equal(t, tt.want, got)
	}
}

func TestRedactURL(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "http://user:****@proxy:8080", RedactURL("http://user:secret@proxy:8080"))
	assert.Equal(t, "http://proxy:8080", RedactURL("http://proxy:8080"))
}

func TestValidateProxyURL(t *testing.T) {
	t.Parallel()
	require.NoError(t, ValidateProxyURL("http://iparchitect_123:secret@188.143.169.27:30151"))
	require.Error(t, ValidateProxyURL("http://USER:PASS@188.143.169.27:30151"))
	require.Error(t, ValidateProxyURL(""))
}

func TestConfigureTransport(t *testing.T) {
	t.Parallel()
	transport := &http.Transport{}
	require.NoError(t, ConfigureTransport(transport, "http://127.0.0.1:8888"))
	require.NotNil(t, transport.Proxy)
}
