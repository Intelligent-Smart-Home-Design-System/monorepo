"""Port: read catalog state for a marketplace listing."""

from typing import Protocol

from extractor.domain.gate import CatalogListingState


class CatalogReader(Protocol):
    """
    Reads the current gold-layer device attributes linked to a tracked_page.

    Used by CatalogCoverageGate to compare catalog fields against listing text.
    Stub implementation always returns CatalogListingState.empty().
    """

    async def get_by_tracked_page(self, tracked_page_id: int) -> CatalogListingState:
        ...
