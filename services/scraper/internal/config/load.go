package config

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/pathnorm"
)

const defaultWBSessionMaxAge = 30 * time.Minute

// LoadFile reads a TOML config, applies SCRAPER_* env overrides, and normalizes file paths
// relative to the config file directory (not the process cwd).
func LoadFile(path string) (Config, error) {
	var cfg Config

	absConfig, err := pathnorm.Abs(path)
	if err != nil {
		return cfg, fmt.Errorf("config path: %w", err)
	}

	v := viper.New()
	v.SetConfigFile(absConfig)
	v.SetEnvPrefix("SCRAPER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return cfg, fmt.Errorf("read config %s: %w", absConfig, err)
	}
	if err := v.Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("unmarshal config: %w", err)
	}

	cfgDir := filepath.Dir(absConfig)
	if err := NormalizePaths(cfgDir, &cfg); err != nil {
		return cfg, err
	}
	applyDefaults(&cfg)
	return cfg, nil
}

// NormalizePaths resolves scraping.wb_session_path and wildberries.browser_profile_dir
// relative to the directory containing the config file.
func NormalizePaths(cfgDir string, cfg *Config) error {
	if cfg.Scraping.WBSessionPath != "" {
		abs, err := pathnorm.AbsIn(cfgDir, cfg.Scraping.WBSessionPath)
		if err != nil {
			return fmt.Errorf("wb_session_path: %w", err)
		}
		cfg.Scraping.WBSessionPath = abs
	}
	if cfg.Wildberries.BrowserProfileDir != "" {
		abs, err := pathnorm.AbsIn(cfgDir, cfg.Wildberries.BrowserProfileDir)
		if err != nil {
			return fmt.Errorf("browser_profile_dir: %w", err)
		}
		cfg.Wildberries.BrowserProfileDir = abs
	}
	return nil
}

func applyDefaults(cfg *Config) {
	if cfg.Scraping.WBSessionMaxAge <= 0 {
		cfg.Scraping.WBSessionMaxAge = defaultWBSessionMaxAge
	}
	if cfg.Dns.UserAgent == "" && cfg.Scraping.UserAgent != "" {
		cfg.Dns.UserAgent = cfg.Scraping.UserAgent
	}
}
