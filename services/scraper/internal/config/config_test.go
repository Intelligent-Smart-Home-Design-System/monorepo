package config_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalConfigFromTOML(t *testing.T) {
	v := viper.New()
	v.SetConfigFile(filepath.Join("testdata", "config.test.toml"))
	v.SetConfigType("toml")

	require.NoError(t, v.ReadInConfig())

	var cfg config.Config
	require.NoError(t, v.Unmarshal(&cfg))

	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "scraper", cfg.Database.User)
	assert.Equal(t, "scraper_pass", cfg.Database.Password)
	assert.Equal(t, "scraper_db", cfg.Database.DBName)
	assert.Equal(t, "disable", cfg.Database.SSLMode)

	assert.Equal(t, 3, cfg.Scraping.MaxRetries)
	assert.InDelta(t, 2.0, cfg.Scraping.RateLimitRps, 0.000001)
	assert.Equal(t, 30*time.Second, cfg.Scraping.Timeout)
}
