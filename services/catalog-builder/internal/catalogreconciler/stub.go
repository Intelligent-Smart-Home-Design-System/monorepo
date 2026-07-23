// Package catalogreconciler implements incremental catalog persistence.
//
// StubCatalogReconciler delegates to legacy TRUNCATE until incremental UPSERT is implemented.
// See monorepo/docs/catalog-pipeline-architecture.md.
package catalogreconciler

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/catalog-builder/internal/domain"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/catalog-builder/internal/ports"
)

// StubCatalogReconciler is the default reconciler: legacy TRUNCATE path only.
type StubCatalogReconciler struct {
	legacy ports.CatalogWriterLegacy
	log    zerolog.Logger
}

func NewStubCatalogReconciler(legacy ports.CatalogWriterLegacy, log zerolog.Logger) *StubCatalogReconciler {
	return &StubCatalogReconciler{legacy: legacy, log: log}
}

// Reconcile stub: logs mode=legacy_truncate and calls WriteCatalogLegacy.
func (r *StubCatalogReconciler) Reconcile(ctx context.Context, catalog *domain.Catalog) (*domain.ReconcileResult, error) {
	r.log.Info().
		Int("devices", len(catalog.Devices)).
		Str("mode", "legacy_truncate").
		Bool("stub_mode", true).
		Msg("catalog_reconcile_started")

	r.log.Info().
		Bool("stub_mode", true).
		Msg("catalog_reconcile_build_plan_start")

	plan := BuildPlan(ctx, catalog)
	r.log.Info().
		Int("plan_entries", len(plan.Entries)).
		Bool("stub_mode", true).
		Msg("catalog_reconcile_build_plan_stub_empty")

	// TODO incremental path:
	// r.applyPlan(ctx, plan)             // UPSERT devices, reconcile device_offers

	if err := r.legacy.WriteCatalogLegacy(ctx, catalog); err != nil {
		return nil, err
	}

	result := &domain.ReconcileResult{
		UsedLegacyTruncate: true,
		DevicesCreated:     len(catalog.Devices),
	}
	r.log.Info().
		Bool("used_legacy_truncate", result.UsedLegacyTruncate).
		Int("devices_created", result.DevicesCreated).
		Int("devices_updated", result.DevicesUpdated).
		Int("offers_linked", result.OffersLinked).
		Bool("stub_mode", true).
		Msg("catalog_reconcile_completed")

	return result, nil
}

// BuildPlan stub — returns empty plan. Real impl compares cluster keys + tracked_page links.
func BuildPlan(_ context.Context, _ *domain.Catalog) *domain.ReconcilePlan {
	return &domain.ReconcilePlan{Entries: nil}
}
