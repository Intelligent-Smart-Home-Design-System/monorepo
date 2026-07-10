package domain

import "time"

// DeviceIdentity is the stable natural key for a catalog device.
// Used for UPSERT instead of TRUNCATE+INSERT.
type DeviceIdentity struct {
	ClusterKey string // e.g. "aqara:smart_lamp:JT-BZ-03AQ/A" or dedup secondary key
	Category   string
	Brand      string
	Model      *string
}

// DeviceOffer binds a device to a marketplace listing (tracked_page).
// Price and stock are read from LatestParsedListingSnapshotID, not stored here.
type DeviceOffer struct {
	DeviceID                     int
	TrackedPageID                int
	LatestParsedListingSnapshotID int
	SourceName                   string // wildberries, yandex, ...
	URL                          string
	LinkedAt                     time.Time
}

// ReconcileAction describes one change in a reconcile plan.
type ReconcileAction string

const (
	ReconcileActionCreateDevice   ReconcileAction = "create_device"
	ReconcileActionUpdateDevice   ReconcileAction = "update_device"
	ReconcileActionDeactivateDevice ReconcileAction = "deactivate_device"
	ReconcileActionLinkOffer      ReconcileAction = "link_offer"
	ReconcileActionUnlinkOffer    ReconcileAction = "unlink_offer"
	ReconcileActionUpdateOffer    ReconcileAction = "update_offer"
)

// ReconcilePlanEntry is a single planned change (stub builds an empty plan).
type ReconcilePlanEntry struct {
	Action ReconcileAction
	Identity DeviceIdentity
	Offer    *DeviceOffer
	DeviceID int
}

// ReconcilePlan is the diff between desired catalog state and DB state.
type ReconcilePlan struct {
	Entries []ReconcilePlanEntry
}

// ReconcileResult reports what the reconciler did.
type ReconcileResult struct {
	UsedLegacyTruncate bool
	DevicesCreated     int
	DevicesUpdated     int
	DevicesDeactivated int
	OffersLinked       int
	OffersUnlinked     int
}
