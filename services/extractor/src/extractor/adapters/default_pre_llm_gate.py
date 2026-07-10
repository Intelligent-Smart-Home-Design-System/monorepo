"""
Default PreLLMGate — stub that always requires LLM.

Check pipeline (when implemented):

  if no_dups_check:
      return requires_llm=True

  if content_hash == previous_content_hash:
      return requires_llm=False, reuse_extraction_id=previous_extraction_id

  if listing.in_stock and catalog covers all device_attributes fields in listing.text:
      return requires_llm=False, skip_reason=CATALOG_COVERS_FIELDS

  # FUTURE: CatalogConfirmationGate
  # if catalog_requires_confirmation(device) and not confirmed:
  #     return requires_llm=True  # or queue for manual review

  return requires_llm=True
"""

from extractor.domain.gate import ExtractionDecision, ListingProcessingContext
from extractor.ports.catalog_reader import CatalogReader
from extractor.ports.pre_llm_gate import PreLLMGate
import structlog


class DefaultPreLLMGate:
    def __init__(self, catalog_reader: CatalogReader, *, no_dups_check: bool = False):
        self._catalog_reader = catalog_reader
        self._no_dups_check = no_dups_check

    async def evaluate(self, ctx: ListingProcessingContext) -> ExtractionDecision:
        log = structlog.get_logger()
        log.info(
            "pre_llm_gate_check_started",
            listing_id=ctx.listing.id,
            tracked_page_id=ctx.tracked_page_id,
            no_dups_check=self._no_dups_check,
            stub_mode=True,
        )

        if self._no_dups_check:
            decision = ExtractionDecision(
                requires_llm=True,
                details={"no_dups_check": True, "stub_mode": True},
            )
            log.info(
                "pre_llm_gate_decision",
                listing_id=ctx.listing.id,
                requires_llm=decision.requires_llm,
                skip_reason=None,
                stub_mode=True,
                reason="no_dups_check flag set; checks bypassed",
            )
            return decision

        # --- ContentHashGate (TODO) ---
        log.debug(
            "pre_llm_gate_check_skipped",
            listing_id=ctx.listing.id,
            gate="content_hash",
            stub_mode=True,
        )

        # --- CatalogCoverageGate (TODO) ---
        if ctx.tracked_page_id is not None:
            catalog = await self._catalog_reader.get_by_tracked_page(ctx.tracked_page_id)
            log.debug(
                "pre_llm_gate_catalog_lookup",
                listing_id=ctx.listing.id,
                tracked_page_id=ctx.tracked_page_id,
                catalog_not_found=catalog.catalog_not_found,
                stub_mode=True,
            )
        else:
            _ = self._catalog_reader

        # --- CatalogConfirmationGate (FUTURE, comment only) ---

        decision = ExtractionDecision(
            requires_llm=True,
            details={"stub_mode": True, "reason": "hardcoded default until gates implemented"},
        )
        log.info(
            "pre_llm_gate_decision",
            listing_id=ctx.listing.id,
            requires_llm=decision.requires_llm,
            skip_reason=None,
            stub_mode=True,
            reason=decision.details.get("reason"),
        )
        return decision


def _text_covers_attributes(text: str, attributes: dict) -> bool:
    """
    Planned helper for CatalogCoverageGate.

    Returns True when every non-null scalar attribute value appears in listing text,
    and every enum/set value is a substring match (case-insensitive).
    """
    normalized = text.casefold()
    for _key, value in attributes.items():
        if value is None:
            continue
        if isinstance(value, list):
            if not all(str(v).casefold() in normalized for v in value):
                return False
        elif str(value).casefold() not in normalized:
            return False
    return True
