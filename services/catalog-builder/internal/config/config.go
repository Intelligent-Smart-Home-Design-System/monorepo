package config

import "slices"

type Config struct {
	Database              DatabaseConfig             `mapstructure:"database"`
	IdentifyingAttributes map[string][]string        `mapstructure:"identifying_attributes"`
	TaxonomySchemaPath    string                     `mapstructure:"taxonomy_schema_path"`
	StrictSchema          bool                       `mapstructure:"strict_schema"`
	Ecosystems            map[string]EcosystemConfig `mapstructure:"ecosystems"`
	SupportedHubProtocols []string                   `mapstructure:"supported_hub_protocols"`
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
	SupportsExternalIntegrations bool     `mapstructure:"supports_external_integrations"`
	SupportedMatterProtocols     []string `mapstructure:"supported_matter_protocols"`
	SupportedMatterDeviceTypes   []string `mapstructure:"supported_matter_device_types"`
}

func (eco *EcosystemConfig) SupportsMatterDeviceType(deviceType string) bool {
	if len(eco.SupportedMatterDeviceTypes) == 1 && eco.SupportedMatterDeviceTypes[0] == "*" {
		return true
	}
	return slices.Contains(eco.SupportedMatterDeviceTypes, deviceType)
}
