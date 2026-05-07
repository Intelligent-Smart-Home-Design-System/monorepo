package config

type Config struct {
	Database              DatabaseConfig      `mapstructure:"database"`
	IdentifyingAttributes map[string][]string `mapstructure:"identifying_attributes"`
	TaxonomySchemaPath    string              `mapstructure:"taxonomy_schema_path"`
	StrictSchema          bool                `mapstructure:"strict_schema"`
	Ecosystems            EcosystemsConfig    `mapstructure:"ecosystems"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

type EcosystemsConfig struct {
	// "cloud-integration" ecosystems like yandex, sber, vk
	// that have direct compatibility lists in their documentation
	Cloud []string `mapstructure:"cloud"`
	// matter enabled ecosystems that support most devices
	// using matter protocols (matter-over-thread or matter-over-wifi)
	Matter []string `mapstructure:"matter"`
	// matter protocol ids
	MatterProtocols []string `mapstructure:"matter_protocols"`
}
