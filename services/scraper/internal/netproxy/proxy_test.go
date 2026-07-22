package netproxy

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
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

func TestProxy_RotationAndLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Пропущено в коротком режиме")
	}

	cfg := config.Config{}

	// TODO remove vulnerable data
	if cfg.Scraping.Proxy == "" {
		t.Skip("Тест пропущен: PROXY URL не задан в конфигурации")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Minute)
	defer cancel()

	startIP, _ := fetchIP(context.Background(), cfg.Scraping.Proxy)
	require.NotEmpty(t, startIP, "Должен определиться стартовый IP через прокси")
	t.Logf("[PROXY TEST] Стартовый IP: %s. Запускаем ожидание авто-ротации...", startIP)

	startTime := time.Now()

	RotateSharedProxy(ctx, cfg.Scraping.Proxy, func(msg string) {
		t.Logf("[PROXY ROTATION LOG]: %s", msg)
	})

	require.NoError(t, ctx.Err(), "Метод должен завершиться по успешной смене IP, а не по таймауту контекста")

	finalIP, _ := fetchIP(context.Background(), cfg.Scraping.Proxy)
	t.Logf("[PROXY TEST] Ротация завершена за %s. Новый IP: %s", time.Since(startTime).Round(time.Second), finalIP)

	assert.NotEmpty(t, finalIP, "Финальный IP должен успешно определиться")
	assert.NotEqual(t, startIP, finalIP, "Финальный IP должен отличаться от стартового после работы RotateSharedProxy")
}

func TestProxy_APIRotationAndLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Пропущено в коротком режиме")
	}

	cfg := config.Config{}

	// TODO remove vulnerable data
	if cfg.Scraping.Proxy == "" {
		t.Skip("Тест пропущен: PROXY URL не задан в конфигурации")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Minute)
	defer cancel()

	startIP, _ := fetchIP(context.Background(), cfg.Scraping.Proxy)
	require.NotEmpty(t, startIP, "Должен определиться стартовый IP через прокси")
	t.Logf("[PROXY TEST] Стартовый IP: %s. Запускаем ожидание авто-ротации...", startIP)

	startTime := time.Now()
	// TODO remove
	apiURL := ""

	err := RotateProxyViaAPI(ctx, cfg.Scraping.Proxy, apiURL, func(msg string) {
		t.Logf("[PROXY ROTATION LOG]: %s", msg)
	})
	assert.NoError(t, err)

	require.NoError(t, ctx.Err(), "Метод должен завершиться по успешной смене IP, а не по таймауту контекста")

	finalIP, _ := fetchIP(context.Background(), cfg.Scraping.Proxy)
	t.Logf("[PROXY TEST] Ротация завершена за %s. Новый IP: %s", time.Since(startTime).Round(time.Second), finalIP)

	assert.NotEmpty(t, finalIP, "Финальный IP должен успешно определиться")
	assert.NotEqual(t, startIP, finalIP, "Финальный IP должен отличаться от стартового после работы RotateSharedProxy")
}
