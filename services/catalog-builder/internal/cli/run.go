package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/catalog-builder/internal/catalogbuilder"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/catalog-builder/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/catalog-builder/internal/repository"
)

func NewRunCmd() *cobra.Command {
	var cfgFile string

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the catalog building job",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
			defer cancel()

			return run(ctx, cfgFile)
		},
	}

	cmd.Flags().StringVar(&cfgFile, "config", "./config.toml", "config file")

	return cmd
}

func run(ctx context.Context, cfgFile string) error {
	log := zerolog.New(os.Stderr).With().Timestamp().Logger()

	cfg, err := loadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	repo, err := repository.NewPostgresRepository(cfg.Database, log)
	if err != nil {
		return fmt.Errorf("connect to db: %w", err)
	}
	defer repo.Close()

	log.Info().Msg("fetching listings")
	listings, err := repo.GetLatestExtractedListings(ctx)
	if err != nil {
		return fmt.Errorf("get listings: %w", err)
	}
	log.Info().Int("count", len(listings)).Msg("listings fetched")

	log.Info().Msg("fetching direct compatibility")
	compat, err := repo.GetLatestDirectCompatibility(ctx)
	if err != nil {
		return fmt.Errorf("get direct compat: %w", err)
	}
	log.Info().Int("count", len(compat)).Msg("direct compat fetched")

	builder, err := catalogbuilder.NewBuilder(catalogbuilder.BuilderConfig{
		IdentifyingAttributes: cfg.IdentifyingAttributes,
		Ecosystems:            cfg.Ecosystems,
		TaxonomySchemaPath:    cfg.TaxonomySchemaPath,
		StrictSchema:          cfg.StrictSchema,
	}, log)
	if err != nil {
		return fmt.Errorf("init builder: %w", err)
	}

	catalog := builder.Build(listings, compat)
	log.Info().Int("devices", len(catalog.Devices)).Msg("catalog built")

	log.Info().Msg("writing catalog")
	if err := repo.WriteCatalog(ctx, catalog); err != nil {
		return fmt.Errorf("write catalog: %w", err)
	}
	log.Info().Msg("done")

	return nil
}

func loadConfig(cfgFile string) (*config.Config, error) {
	v := viper.New()
	v.SetConfigFile(cfgFile)

	v.SetEnvPrefix("CATALOG_BUILDER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg config.Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
}
