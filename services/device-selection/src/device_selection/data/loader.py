from __future__ import annotations

import json
from typing import Any, Optional

import asyncpg

from device_selection.core.model import BridgeCompat, Device, DirectCompat
from device_selection.core.quality import bayesian_quality
from device_selection.data.catalog import InMemoryCatalog


_LOAD_DEVICES_QUERY = """
SELECT
    d.id                            AS device_id,
    d.brand,
    d.model,
    d.category,
    d.device_attributes,

    best.id                         AS source_listing_id,
    best.extracted_price            AS price,

    quality.total_reviews,
    quality.avg_rating

FROM devices d

JOIN LATERAL (
    SELECT
        l.id,
        pls.extracted_price,
        pls.extracted_review_count
    FROM listing_device_links ldl
    JOIN llm_extracted_listings l ON l.id = ldl.llm_extracted_listing_id
    JOIN parsed_listing_snapshots pls ON pls.id = l.parsed_listing_snapshot_id
    WHERE ldl.device_id = d.id
      AND pls.extracted_price IS NOT NULL
      AND pls.extracted_in_stock = TRUE
    ORDER BY pls.extracted_review_count DESC
    LIMIT 1
) best ON TRUE

JOIN LATERAL (
    SELECT
        SUM(pls.extracted_review_count)::int AS total_reviews,
        CASE
            WHEN SUM(pls.extracted_review_count) = 0 THEN 0
            ELSE SUM(pls.extracted_rating * pls.extracted_review_count)
                 / SUM(pls.extracted_review_count)
        END::float AS avg_rating
    FROM listing_device_links ldl
    JOIN llm_extracted_listings l ON l.id = ldl.llm_extracted_listing_id
    JOIN parsed_listing_snapshots pls ON pls.id = l.parsed_listing_snapshot_id
    WHERE ldl.device_id = d.id
) quality ON TRUE
"""

_LOAD_DEVICES_WITH_QUALITY_QUERY = """
SELECT
    d.id                            AS device_id,
    d.brand,
    d.model,
    d.category,
    d.device_attributes,
    d.quality,

    best.id                         AS source_listing_id,
    best.extracted_price            AS price

FROM devices d

JOIN LATERAL (
    SELECT
        l.id,
        pls.extracted_price,
        pls.extracted_review_count
    FROM listing_device_links ldl
    JOIN llm_extracted_listings l ON l.id = ldl.llm_extracted_listing_id
    JOIN parsed_listing_snapshots pls ON pls.id = l.parsed_listing_snapshot_id
    WHERE ldl.device_id = d.id
      AND pls.extracted_price IS NOT NULL
      AND pls.extracted_in_stock = TRUE
    ORDER BY pls.extracted_review_count DESC
    LIMIT 1
) best ON TRUE
"""

_LOAD_DIRECT_COMPAT_QUERY = """
SELECT device_id, ecosystem, protocol
FROM direct_compatibility
"""

_LOAD_BRIDGE_COMPAT_QUERY = """
SELECT device_id, ecosystem_source, ecosystem_target, protocol
FROM bridge_ecosystem_compatibility
"""


class CatalogLoader:
    def __init__(
        self,
        pool: asyncpg.Pool,
        calculate_quality: bool = True,
        min_reviews: int = 10,
        global_avg: float = 4.0,
        rating_floor: float = 4.0,
    ):
        self._pool = pool
        self._calculate_quality = calculate_quality
        self._min_reviews = min_reviews
        self._global_avg = global_avg
        self._rating_floor = rating_floor

    async def load(self) -> InMemoryCatalog:
        async with self._pool.acquire() as conn:
            if self._calculate_quality:
                device_rows = await conn.fetch(_LOAD_DEVICES_QUERY)
            else:
                device_rows = await conn.fetch(_LOAD_DEVICES_WITH_QUALITY_QUERY)
            direct_rows = await conn.fetch(_LOAD_DIRECT_COMPAT_QUERY)
            bridge_rows = await conn.fetch(_LOAD_BRIDGE_COMPAT_QUERY)

        direct_by_device: dict[int, list[DirectCompat]] = {}
        for row in direct_rows:
            direct_by_device.setdefault(row["device_id"], []).append(
                DirectCompat(ecosystem=row["ecosystem"], protocol=row["protocol"])
            )

        bridge_by_device: dict[int, list[BridgeCompat]] = {}
        for row in bridge_rows:
            bridge_by_device.setdefault(row["device_id"], []).append(
                BridgeCompat(
                    source_ecosystem=row["ecosystem_source"],
                    target_ecosystem=row["ecosystem_target"],
                    protocol=row["protocol"],
                )
            )

        devices: list[Device] = []
        for row in device_rows:
            device_id = row["device_id"]

            if self._calculate_quality:
                total_reviews = row["total_reviews"] or 0
                avg_rating = float(row["avg_rating"] or 0.0)
                quality = bayesian_quality(
                    total_reviews=total_reviews,
                    avg_rating=avg_rating,
                    min_reviews=self._min_reviews,
                    global_avg=self._global_avg,
                    rating_floor=self._rating_floor,
                )
            else:
                quality = float(row["quality"] or 0.0)

            attributes: dict[str, Any] = json.loads(row["device_attributes"]) if row["device_attributes"] else {}

            devices.append(Device(
                device_id=device_id,
                device_type=row["category"],
                brand=row["brand"] or None,
                model=row["model"] or None,
                attributes=attributes,
                price=float(row["price"]),
                quality=quality,
                source_listing_id=row["source_listing_id"],
                direct_compat=tuple(direct_by_device.get(device_id, [])),
                bridge_compat=tuple(bridge_by_device.get(device_id, [])),
            ))

        return InMemoryCatalog(devices)
