package catalogbuilder

import (
	"fmt"
	"slices"
	"sort"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/catalog-builder/internal/domain"

	"github.com/rs/zerolog"
)

type BuilderConfig struct {
	IdentifyingAttributes map[string][]string
	CloudEcosystems       []string
	MatterEcosystems      []string
	MatterProtocols       []string
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
		device.DirectCompatibility = append(device.DirectCompatibility, &domain.DirectCompatibility{
			Ecosystem: c.Ecosystem,
			Protocol:  c.Protocol,
		})
	}

	for _, d := range devices {
		b.buildCompatibilityLinks(d)
	}

	return &domain.Catalog{Devices: devices}
}

func (b *Builder) buildCompatibilityLinks(d *domain.Device) {
	ecosystems := getStringSet(d.DeviceAttributes, "ecosystem")
	protocols := getStringSet(d.DeviceAttributes, "protocol")

	alreadyDirectCompat := make(map[string]bool)
	for _, dc := range d.DirectCompatibility {
		alreadyDirectCompat[dc.Ecosystem] = true
	}

	var vendorEcosystems []string
	for _, eco := range ecosystems {
		if !slices.Contains(b.cfg.CloudEcosystems, eco) && !slices.Contains(b.cfg.MatterEcosystems, eco) {
			vendorEcosystems = append(vendorEcosystems, eco)
			for _, proto := range protocols {
				d.DirectCompatibility = append(d.DirectCompatibility, &domain.DirectCompatibility{
					Ecosystem: eco,
					Protocol:  proto,
				})
			}
			alreadyDirectCompat[eco] = true
		}
	}

	hasMatter := false
	for _, proto := range protocols {
		if slices.Contains(b.cfg.MatterProtocols, proto) {
			hasMatter = true
			break
		}
	}

	for _, eco := range ecosystems {
		if !slices.Contains(b.cfg.MatterEcosystems, eco) {
			continue
		}
		if hasMatter {
			for _, proto := range protocols {
				if slices.Contains(b.cfg.MatterProtocols, proto) {
					d.DirectCompatibility = append(d.DirectCompatibility, &domain.DirectCompatibility{
						Ecosystem: eco,
						Protocol:  proto,
					})
				}
			}
		} else {
			for _, vendor := range vendorEcosystems {
				d.BridgeCompatibility = append(d.BridgeCompatibility, &domain.BridgeCompatibility{
					SourceEcosystem: vendor,
					TargetEcosystem: eco,
					Protocol:        "cloud",
				})
			}
		}
	}

	for _, eco := range ecosystems {
		if !slices.Contains(b.cfg.CloudEcosystems, eco) {
			continue
		}
		if alreadyDirectCompat[eco] {
			continue
		}
		for _, vendor := range vendorEcosystems {
			d.BridgeCompatibility = append(d.BridgeCompatibility, &domain.BridgeCompatibility{
				SourceEcosystem: vendor,
				TargetEcosystem: eco,
				Protocol:        "cloud",
			})
		}
	}
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
