// Package ports defines interfaces for catalog persistence strategies.
//
// See monorepo/docs/catalog-pipeline-architecture.md for the incremental catalog design.
package ports

import (
	"context"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/catalog-builder/internal/domain"
)

// CatalogReconciler applies catalog changes without TRUNCATE when incremental mode is enabled.
//
// Stub implementation (catalogreconciler.Stub) delegates to legacy TRUNCATE+INSERT.
type CatalogReconciler interface {
	// Reconcile computes and applies incremental changes.
	// Stub: UsedLegacyTruncate=true, delegates to WriteCatalogLegacy.
	Reconcile(ctx context.Context, catalog *domain.Catalog) (*domain.ReconcileResult, error)
}

// CatalogWriterLegacy is the current truncate-based writer.
type CatalogWriterLegacy interface {
	WriteCatalogLegacy(ctx context.Context, catalog *domain.Catalog) error
}
