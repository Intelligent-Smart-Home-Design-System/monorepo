package config

type Config struct {
	Database              DatabaseConfig             `mapstructure:"database"`
	IdentifyingAttributes map[string][]string        `mapstructure:"identifying_attributes"`
	TaxonomySchemaPath    string                     `mapstructure:"taxonomy_schema_path"`
	StrictSchema          bool                       `mapstructure:"strict_schema"`
	Ecosystems            map[string]EcosystemConfig `mapstructure:"ecosystems"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

type EcosystemConfig struct {
	IsBridgeTarget             bool     `mapstructure:"is_bridge_target"`
	SupportedMatterProtocols   []string `mapstructure:"supported_matter_protocols"`
	SupportedMatterDeviceTypes []string `mapstructure:"supported_matter_device_types"`
}
