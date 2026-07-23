import asyncio
import datetime

import structlog

from extractor.enrichment import enrich_from_silver_layer
from extractor.domain.gate import ExtractionDecision, ListingProcessingContext
from extractor.domain.models import ExtractionSnapshot, ListingSnapshot
from extractor.ports.extraction import Extractor
from extractor.ports.pre_llm_gate import PreLLMGate
from extractor.ports.repository import ExtractionRepository


class Worker:
    def __init__(
        self,
        extractor: Extractor,
        repository: ExtractionRepository,
        model: str,
        batch_size: int,
        pre_llm_gate: PreLLMGate | None = None,
    ):
        self._extractor = extractor
        self._repository = repository
        self._model = model
        self._batch_size = batch_size
        self._pre_llm_gate = pre_llm_gate

    async def run(self):
        log = structlog.get_logger()
        
        batch_size = self._batch_size
        total_processed = 0
        total_errors = 0
        total_skipped_llm = 0

        log.info(
            "extraction_started",
            pre_llm_gate_enabled=self._pre_llm_gate is not None,
        )

        while True:
            snapshots = await self._repository.get_pending_snapshots(limit=batch_size)
            if not snapshots:
                log.info("no_pending_snapshots", total_processed=total_processed)
                break

            log.info("processing_batch", batch_size=len(snapshots), total_processed=total_processed)

            results = await asyncio.gather(
                *[self._process_listing(listing) for listing in snapshots],
                return_exceptions=True
            )
            
            for listing, result in zip(snapshots, results):
                if isinstance(result, Exception):
                    total_errors += 1
                    log.error(
                        "extraction_failed",
                        listing_id=listing.id,
                        error=str(result),
                        error_type=type(result).__name__,
                        exc_info=result,
                    )
                    continue

                if result is None:
                    total_skipped_llm += 1
                    continue

                try:
                    await self._repository.save_extraction(result)
                    total_processed += 1
                except Exception as e:
                    total_errors += 1
                    log.error("save_failed", listing_id=listing.id, error=str(e))

        log.info(
            "extraction_finished",
            total_processed=total_processed,
            total_skipped_llm=total_skipped_llm,
            total_errors=total_errors,
        )

    async def _process_listing(self, listing: ListingSnapshot) -> ExtractionSnapshot | None:
        log = structlog.get_logger()
        decision = await self._evaluate_gate(listing)
        log.info(
            "pre_llm_gate_decision",
            listing_id=listing.id,
            requires_llm=decision.requires_llm,
            skip_reason=decision.skip_reason.value if decision.skip_reason else None,
            stub_mode=decision.details.get("stub_mode"),
            details=decision.details,
        )

        if not decision.requires_llm:
            # TODO: copy reuse_extraction_id → new llm_extracted_listings row; mark processed
            log.warning(
                "pre_llm_skip_not_implemented",
                listing_id=listing.id,
                reuse_extraction_id=decision.reuse_extraction_id,
            )
            return None

        return await self._extract(listing)

    async def _evaluate_gate(self, listing: ListingSnapshot) -> ExtractionDecision:
        if self._pre_llm_gate is None:
            structlog.get_logger().info(
                "pre_llm_gate_disabled",
                listing_id=listing.id,
                requires_llm=True,
            )
            return ExtractionDecision(requires_llm=True)

        ctx = ListingProcessingContext(listing=listing)
        return await self._pre_llm_gate.evaluate(ctx)

    async def _extract(self, listing: ListingSnapshot) -> ExtractionSnapshot:
        detected = await self._extractor.detect_device_type(listing)
        attributes = await self._extractor.extract(listing, detected.type)
        brand, model, attributes = enrich_from_silver_layer(listing, attributes)

        return ExtractionSnapshot(
            parsed_listing_snapshot_id=listing.id,
            brand=brand,
            model=model,
            category=detected.type,
            category_confidence=detected.confidence,
            extracted_at=datetime.datetime.now(datetime.timezone.utc),
            llm_model=self._model,
            device_attributes=attributes,
        )
