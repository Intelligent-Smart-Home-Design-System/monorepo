"""
Stub CatalogReader — always reports catalog_not_found.

Real implementation will join:
  tracked_pages → listing_device_links → devices → device_attributes
using latest llm_extracted_listing per tracked_page.
"""

import structlog

from extractor.domain.gate import CatalogListingState
from extractor.ports.catalog_reader import CatalogReader


class StubCatalogReader:
    async def get_by_tracked_page(self, tracked_page_id: int) -> CatalogListingState:
        structlog.get_logger().info(
            "catalog_reader_lookup",
            tracked_page_id=tracked_page_id,
            stub_mode=True,
            catalog_not_found=True,
        )
        return CatalogListingState.empty(tracked_page_id=tracked_page_id)
