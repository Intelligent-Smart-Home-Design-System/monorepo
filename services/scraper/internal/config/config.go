package config

import (
	"time"
)

type Config struct {
	Database DatabaseConfig `mapstructure:"database"`
	Scraping ScrapingConfig `mapstructure:"scraping"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

type ScrapingConfig struct {
    MaxRetries   int           `mapstructure:"max_retries"`
    RateLimitRps float64       `mapstructure:"rate_limit_rps"`
    Timeout      time.Duration `mapstructure:"timeout"`
    UserAgent    string        `mapstructure:"user_agent"`
    Proxy        string        `mapstructure:"proxy"`
    WBCardBasket  string 	   `mapstructure:"wb_card_basket"`
    WBSessionPath string       `mapstructure:"wb_session_path"`
    WBRPS         float64      `mapstructure:"wb_rps"`
}