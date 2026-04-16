from typing import Protocol
from extractor.domain.models import ExtractionSnapshot, ListingSnapshot


class ExtractionRepository(Protocol):
    async def get_pending_snapshots(self, limit: int, offset: int) -> list[ListingSnapshot]:
        ...

    async def save_extraction(self, extraction: ExtractionSnapshot) -> None:
        ...
