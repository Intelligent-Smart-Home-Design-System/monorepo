package catalogbuilder

import (
	"fmt"
	"slices"
	"sort"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/catalog-builder/internal/config"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/catalog-builder/internal/domain"

	"github.com/rs/zerolog"
)

const smartHubCategory = "smart_hub"

type BuilderConfig struct {
	IdentifyingAttributes map[string][]string
	Ecosystems            map[string]config.EcosystemConfig
	TaxonomySchemaPath    string
	// StrictSchema skips devices that fail schema validation instead of just warning
	StrictSchema bool
}

type Builder struct {
	cfg     BuilderConfig
	schemas *taxonomySchemas
	log     zerolog.Logger
}

func NewBuilder(cfg BuilderConfig, log zerolog.Logger) (*Builder, error) {
	b := &Builder{cfg: cfg, log: log}
	if cfg.TaxonomySchemaPath != "" {
		s, err := loadTaxonomySchemas(cfg.TaxonomySchemaPath)
		if err != nil {
			return nil, fmt.Errorf("load taxonomy schemas: %w", err)
		}
		b.schemas = s
	}
	return b, nil
}

func (b *Builder) Build(listings []*domain.ExtractedListing, compat []*domain.ScrapedDirectCompatibility) *domain.Catalog {
	sorted := make([]*domain.ExtractedListing, len(listings))
	copy(sorted, listings)
	sort.Slice(sorted, func(i, j int) bool {
		iHas := sorted[i].Model != nil
		jHas := sorted[j].Model != nil
		return iHas && !jHas
	})

	keyToCluster := make(map[string][]*domain.ExtractedListing)
	secondaryToPrimary := make(map[string]string)

	for _, listing := range sorted {
		primaryKey, primaryErr := getPrimaryKey(listing)

		attrs := b.cfg.IdentifyingAttributes[listing.Category]
		secondaryKey, secondaryErr := getSecondaryKey(listing, attrs)

		if primaryErr == nil {
			keyToCluster[primaryKey] = append(keyToCluster[primaryKey], listing)
			if secondaryErr == nil {
				secondaryToPrimary[secondaryKey] = primaryKey
			}
			continue
		}

		if secondaryErr != nil {
			b.log.Warn().
				Int("listing_id", listing.Id).
				Str("brand", listing.Brand).
				Str("category", listing.Category).
				Err(secondaryErr).
				Msg("could not get primary or secondary key for listing, creating isolated device")
			isolatedKey := fmt.Sprintf("isolated:%d", listing.Id)
			keyToCluster[isolatedKey] = append(keyToCluster[isolatedKey], listing)
			continue
		}

		if existingPrimary, ok := secondaryToPrimary[secondaryKey]; ok {
			keyToCluster[existingPrimary] = append(keyToCluster[existingPrimary], listing)
		} else {
			keyToCluster[secondaryKey] = append(keyToCluster[secondaryKey], listing)
		}
	}

	var devices []*domain.Device
	modelToDevice := make(map[string]*domain.Device)

	for clusterKey, cluster := range keyToCluster {
		attrs := deduplicateAttributes(cluster, b.log)
		if len(attrs) == 0 {
			b.log.Warn().Str("cluster_key", clusterKey).Msg("cluster produced empty attributes, skipping")
			continue
		}

		brand := cluster[0].Brand
		category := cluster[0].Category
		taxonomyVersion := cluster[0].TaxonomyVersion

		device := &domain.Device{
			Brand:            brand,
			Model:            cluster[0].Model,
			Category:         category,
			DeviceAttributes: attrs,
			TaxonomyVersion:  taxonomyVersion,
			Listings:         cluster,
		}

		if b.schemas != nil {
			if valid, errs := b.schemas.validate(category, attrs); !valid {
				b.log.Warn().
					Str("brand", brand).
					Str("category", category).
					Strs("errors", errs).
					Msg("device failed schema validation after merge")
				if b.cfg.StrictSchema {
					continue
				}
			}
		}

		model := clusterKey
		if cluster[0].Model != nil {
			model = fmt.Sprintf("%s:%s", brand, *cluster[0].Model)
		}

		if category == smartHubCategory {
			// special rule - if hub supports matter-over-wifi and ecosystem supports matter-over-wifi too, add support for it
			attrs := device.DeviceAttributes
			protocol := getStringSet(attrs, "protocol")
			if slices.Contains(protocol, "wifi") && !slices.Contains(protocol, "matter-over-wifi") {
				device.DeviceAttributes["protocol"] = append(protocol, "matter-over-wifi")
			}
		}

		devices = append(devices, device)
		modelToDevice[model] = device
	}

	for _, c := range compat {
		modelKey := fmt.Sprintf("%s:%s", c.Brand, c.Model)
		device, ok := modelToDevice[modelKey]
		if !ok {
			b.log.Warn().
				Str("brand", c.Brand).
				Str("model", c.Model).
				Str("ecosystem", c.Ecosystem).
				Msg("scraped compat record references unknown device")
			continue
		}
		if c.Ecosystem == device.Brand {
			continue // skip - added as generic rule
		}
		addDirect(device, c.Ecosystem, c.Protocol)
	}

	for _, d := range devices {
		b.buildCompatibilityLinks(d)
		if d.Category == smartHubCategory {
			ecosystems := []string{}
			for _, c := range d.DirectCompatibility {
				if !slices.Contains(ecosystems, c.Ecosystem) {
					ecosystems = append(ecosystems, c.Ecosystem)
				}
			}
			d.DeviceAttributes["ecosystem"] = ecosystems
		}
	}

	return &domain.Catalog{Devices: devices}
}

func (b *Builder) buildCompatibilityLinks(d *domain.Device) {
	ecosystems := getStringSet(d.DeviceAttributes, "ecosystem")
	protocols := getStringSet(d.DeviceAttributes, "protocol")

	for _, eco := range ecosystems {
		if eco == d.Brand {
			for _, proto := range protocols {
				addDirect(d, eco, proto)
			}
		}
	}

	for _, eco := range ecosystems {
		config := b.cfg.Ecosystems[eco]
		if !config.SupportsExternalIntegrations {
			for _, proto := range protocols {
				addDirect(d, eco, proto)
			}
			if d.Category == smartHubCategory {
				continue
			}
			for _, ecoTarget := range ecosystems {
				targetConfig := b.cfg.Ecosystems[ecoTarget]
				if targetConfig.SupportsExternalIntegrations {
					d.BridgeCompatibility = append(d.BridgeCompatibility, &domain.BridgeCompatibility{
						SourceEcosystem: eco,
						TargetEcosystem: ecoTarget,
						Protocol:        "cloud",
					})
				}
			}
		}

		// matter
		if d.Category == smartHubCategory && config.SupportsExternalIntegrations {
			continue
		}
		if config.SupportsMatterDeviceType(d.Category) {
			protocol := getStringSet(d.DeviceAttributes, "protocol")
			for _, matterProtocol := range config.SupportedMatterProtocols {
				if slices.Contains(protocol, matterProtocol) {
					addDirect(d, eco, matterProtocol)
				}
			}
		}
	}
}

func addDirect(d *domain.Device, ecosystem string, protocol string) {
	for _, c := range d.DirectCompatibility {
		if c.Ecosystem == ecosystem && c.Protocol == protocol {
			return
		}
	}
	d.DirectCompatibility = append(d.DirectCompatibility, &domain.DirectCompatibility{
		Ecosystem: ecosystem,
		Protocol:  protocol,
	})
}

func getStringSet(attrs map[string]any, key string) []string {
	val, ok := attrs[key]
	if !ok {
		return nil
	}
	result, ok := val.([]string)
	if !ok {
		return nil
	}
	return result
}
