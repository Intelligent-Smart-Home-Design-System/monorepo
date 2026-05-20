import asyncio
import datetime

import structlog

from extractor.domain.models import ExtractionSnapshot, ListingSnapshot
from extractor.ports.extraction import Extractor
from extractor.ports.repository import ExtractionRepository

class Worker:
    def __init__(self, extractor: Extractor, repository: ExtractionRepository, model: str, batch_size: int):
        self._extractor = extractor
        self._repository = repository
        self._model = model
        self._batch_size = batch_size

    async def run(self):
        log = structlog.get_logger()
        
        batch_size = self._batch_size
        total_processed = 0
        total_errors = 0

        log.info("extraction_started")

        while True:
            snapshots = await self._repository.get_pending_snapshots(limit=batch_size)
            if not snapshots:
                log.info("no_pending_snapshots", total_processed=total_processed)
                break

            log.info("processing_batch", batch_size=len(snapshots), total_processed=total_processed)


            results = await asyncio.gather(
                *[self._extract(listing) for listing in snapshots],
                return_exceptions=True
            )
            
            for listing, result in zip(snapshots, results):
                if isinstance(result, Exception):
                    total_errors += 1
                    log.error("extraction_failed", listing_id=listing.id, error=str(result))
                    continue

                try:
                    await self._repository.save_extraction(result)
                    total_processed += 1
                except Exception as e:
                    total_errors += 1
                    log.error("save_failed", listing_id=listing.id, error=str(e))

        log.info("extraction_finished", total_processed=total_processed, total_errors=total_errors)

    async def _extract(self, listing: ListingSnapshot) -> ExtractionSnapshot:
        detected = await self._extractor.detect_device_type(listing)

        attributes = await self._extractor.extract(listing, detected.type)

        return ExtractionSnapshot(
            parsed_listing_snapshot_id=listing.id,
            brand=attributes.get("brand") or listing.brand,
            model=attributes.get("model") or "unknown",
            category=detected.type,
            category_confidence=detected.confidence,
            extracted_at=datetime.datetime.now(datetime.timezone.utc),
            llm_model=self._model,
            device_attributes=attributes,
        )