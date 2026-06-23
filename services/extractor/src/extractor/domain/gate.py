"""Domain models for pre-LLM gate and catalog lookup."""

from enum import Enum
from typing import Any

from pydantic import BaseModel, Field

from extractor.domain.models import ListingSnapshot


class SkipReason(str, Enum):
    """Why LLM extraction was skipped."""

    UNCHANGED_CONTENT = "unchanged_content"
    """content_hash matches the previous snapshot for the same tracked_page."""

    CATALOG_COVERS_FIELDS = "catalog_covers_fields"
    """In-stock listing text already contains all catalog attribute values; re-extract not needed."""

    REUSED_PREVIOUS_EXTRACTION = "reused_previous_extraction"
    """Previous llm_extracted_listings row copied to the new parsed_listing_snapshot."""

    # FUTURE: CatalogConfirmationGate may introduce:
    # PENDING_CATALOG_CONFIRMATION = "pending_catalog_confirmation"
    # """External catalog source requires manual confirmation before trusting attributes."""


class ExtractionDecision(BaseModel):
    """Result of PreLLMGate.evaluate()."""

    requires_llm: bool = True
    skip_reason: SkipReason | None = None
    reuse_extraction_id: int | None = None
    details: dict[str, Any] = Field(default_factory=dict)


class CatalogListingState(BaseModel):
    """
    Snapshot of catalog data for a marketplace listing (tracked_page).

    Used by CatalogCoverageGate to check whether LLM re-run is necessary.
    Stub CatalogReader returns empty defaults (catalog_not_found=True).
    """

    tracked_page_id: int | None = None
    device_id: int | None = None
    category: str | None = None
    brand: str | None = None
    model: str | None = None
    device_attributes: dict[str, Any] = Field(default_factory=dict)
    taxonomy_version: str | None = None
    catalog_not_found: bool = True

    @classmethod
    def empty(cls, tracked_page_id: int | None = None) -> "CatalogListingState":
        return cls(tracked_page_id=tracked_page_id, catalog_not_found=True)


class ListingProcessingContext(BaseModel):
    """
    Input to PreLLMGate: silver snapshot + metadata for hash/catalog checks.

    Repository will populate content_hash and tracked_page_id in a later iteration.
    """

    listing: ListingSnapshot
    content_hash: str | None = None
    tracked_page_id: int | None = None
    previous_content_hash: str | None = None
    previous_extraction_id: int | None = None
