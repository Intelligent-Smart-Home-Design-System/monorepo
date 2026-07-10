package config

import (
	"time"
)

type Config struct {
    Database    DatabaseConfig    `mapstructure:"database"`
    Scraping    ScrapingConfig    `mapstructure:"scraping"`
    Wildberries WildberriesConfig `mapstructure:"wildberries"`
    Yandex      YandexConfig      `mapstructure:"yandex"`
    Dns         DnsConfig         `mapstructure:"dns"`
    Example     ExampleConfig     `mapstructure:"example"`
    Jobs        JobsConfig        `mapstructure:"jobs"`
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
    Proxy        string        `mapstructure:"proxy"` // http(s):// or socks5:// proxy for all HTTP and browser requests
    WBCardBasket  string 	   `mapstructure:"wb_card_basket"`
    WBSessionPath string       `mapstructure:"wb_session_path"`
    WBSessionMaxAge time.Duration `mapstructure:"wb_session_max_age"` // freshness for discovery API (default 30m)
    WBRPS         float64      `mapstructure:"wb_rps"`
}

type WildberriesDiscoveryConfig struct {
    DiscoveryTextQueries []string `mapstructure:"discovery_text_queries"`
    MaxPages             int      `mapstructure:"max_pages"`
    URLTemplate          string   `mapstructure:"url_template"`
}

type WildberriesConfig struct {
    Discovery               WildberriesDiscoveryConfig `mapstructure:"discovery"`
    Category                WildberriesCategoryConfig  `mapstructure:"category"`
    BrandAliases            map[string]string          `mapstructure:"brand_aliases"`
    SmartHomeDeviceMarkers  []string                   `mapstructure:"smart_home_device_markers"`
    BrowserUserMode         *bool                      `mapstructure:"browser_user_mode"` // nil = auto (true on Windows/macOS)
    BrowserProfileDir       string                     `mapstructure:"browser_profile_dir"` // empty = %LOCALAPPDATA%/rod/wildberries-chrome
}

type WildberriesCategoryConfig struct {
    CategoryURL string `mapstructure:"category_url"`
}

type YandexConfig struct {
    SupportedZigbeeDevicesURL string `mapstructure:"supported_zigbee_devices_url"`
}

type DnsConfig struct {
	DiscoverySeeds           []string          `mapstructure:"discovery_seeds"`
	SearchQueries            []string          `mapstructure:"search_queries"`
	MaxPages                 int               `mapstructure:"max_pages"`
	MaxBFSFetches            int               `mapstructure:"max_bfs_fetches"` // 0 = unlimited BFS page fetches
	UserAgent                string            `mapstructure:"user_agent"`
	BrowserUserMode          *bool             `mapstructure:"browser_user_mode"` // nil = auto (true on macOS)
	BrandAliases             map[string]string `mapstructure:"brand_aliases"`
	SmartHomeDeviceMarkers   []string          `mapstructure:"smart_home_device_markers"`
}

// ExampleConfig — шаблон настроек нового источника (см. internal/parsers/example/doc.go).
//
// Два допустимых способа задать точку входа для discovery (можно комбинировать):
//   - discovery_seeds — готовые URL (как DNS catalog/search)
//   - discovery_text_queries + discovery_url_template — текст + шаблон API (как Wildberries)
//
// Без полного discovery-пайплайна: category_url и/или listing_urls (см. BootstrapRegularTasks).
type ExampleConfig struct {
	// Pattern A (DNS): готовые URL → page_type=discovery или старт BFS.
	DiscoverySeeds []string `mapstructure:"discovery_seeds"`

	// Pattern B (WB): текст запроса → виртуальный URL example://discovery/{query};
	// скрейпер подставляет query/page в discovery_url_template.
	DiscoveryTextQueries []string `mapstructure:"discovery_text_queries"`
	DiscoveryURLTemplate string   `mapstructure:"discovery_url_template"`
	MaxPages             int      `mapstructure:"max_pages"`

	// Pattern A (DNS search): текст → URL по search_url_template (плейсхолдеры {query}, {page}).
	SearchQueries     []string `mapstructure:"search_queries"`
	SearchURLTemplate string   `mapstructure:"search_url_template"`

	// (опционально) Без discovery: фиксированные category / listing в tracked_pages.
	CategoryURLs []string `mapstructure:"category_urls"`
	ListingURLs  []string `mapstructure:"listing_urls"`

	BrandAliases map[string]string `mapstructure:"brand_aliases"`
}
