"""Port: decide whether a listing needs LLM extraction."""

from typing import Protocol

from extractor.domain.gate import ExtractionDecision, ListingProcessingContext


class PreLLMGate(Protocol):
    """
    Orchestrates pre-LLM checks for a parsed_listing_snapshot.

    Planned checks (see docs/catalog-pipeline-architecture.md):
      1. ContentHashGate — skip if content_hash unchanged
      2. CatalogCoverageGate — skip if in_stock and catalog fields covered by text
      3. CatalogConfirmationGate [FUTURE] — external catalog confirmation required

    Implementations must be pure regarding side effects; persistence is Worker's job.
    """

    async def evaluate(self, ctx: ListingProcessingContext) -> ExtractionDecision:
        ...
