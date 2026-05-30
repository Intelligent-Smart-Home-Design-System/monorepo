from __future__ import annotations

from dataclasses import dataclass
from typing import Any


@dataclass
class DeviceRecord:
    """
    Устройство золотого слоя в виде, достаточном для оценки качества.

    rating/review_count агрегированы по всем листингам устройства
    (через listing_device_links -> llm_extracted_listings -> parsed_listing_snapshots).
    Цена не нужна: в devices.quality пишется только Q; Value считает downstream-подборщик.
    """

    id: int
    category: str
    device_attributes: dict[str, Any]
    rating: float | None
    review_count: int

    def to_eval_record(self) -> dict[str, Any]:
        """Преобразование в формат, который ожидает QualityEvaluator.evaluate_device."""
        attrs = self.device_attributes or {}
        return {
            "id": self.id,
            "category": self.category,
            "specs": attrs,
            "protocol": attrs.get("protocol") or [],
            "reviews": {"rating": self.rating, "count": self.review_count},
        }
