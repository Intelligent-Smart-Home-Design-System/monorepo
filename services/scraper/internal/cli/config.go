package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/scraper/internal/config"
)

func readConfig(cfgFile string, cfg *config.Config) error {
	viper.SetConfigFile(cfgFile)

	viper.SetEnvPrefix("SCRAPER")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("reading config: %w", err)
		}
		fmt.Fprintln(os.Stderr, "No config file found, using defaults and environment variables")
	}

	if err := viper.Unmarshal(cfg); err != nil {
		return fmt.Errorf("unmarshaling config: %w", err)
	}

	return nil
}
