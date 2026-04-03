import datetime

import structlog

from extractor.domain.models import ExtractionSnapshot
from extractor.ports.extraction import Extractor
from extractor.ports.repository import ExtractionRepository

class Worker:
    def __init__(self, extractor: Extractor, repository: ExtractionRepository, model: str):
        self._extractor = extractor
        self._repository = repository
        self._model = model

    async def run(self):
        log = structlog.get_logger()
        
        BATCH_SIZE = 10
        total_processed = 0
        total_errors = 0

        log.info("extraction_started")

        while True:
            snapshots = await self._repository.get_pending_snapshots(limit=BATCH_SIZE)
            if not snapshots:
                log.info("no_pending_snapshots", total_processed=total_processed)
                break

            log.info("processing_batch", batch_size=len(snapshots), total_processed=total_processed)

            for listing in snapshots:
                try:
                    detected = await self._extractor.detect_device_type(listing)

                    attributes = await self._extractor.extract(listing, detected.type)

                    extraction = ExtractionSnapshot(
                        parsed_listing_snapshot_id=listing.id,
                        brand=attributes.get("brand") or listing.brand,
                        model=attributes.get("model") or "unknown",
                        category=detected.type,
                        category_confidence=detected.confidence,
                        extracted_at=datetime.datetime.now(datetime.timezone.utc),
                        llm_model=self._model,
                        device_attributes=attributes,
                    )

                    await self._repository.save_extraction(extraction)
                    total_processed += 1

                except Exception as e:
                    total_errors += 1
                    log.error("extraction_failed", listing_id=listing.id, error=str(e))
                    continue

        log.info("extraction_finished", total_processed=total_processed, total_errors=total_errors)
