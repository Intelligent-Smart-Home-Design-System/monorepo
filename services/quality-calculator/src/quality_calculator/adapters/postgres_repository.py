from __future__ import annotations

import json

import asyncpg

from quality_calculator.config import DatabaseConfig
from quality_calculator.domain.models import DeviceRecord
from quality_calculator.ports.repository import QualityRepository


class PostgresQualityRepository(QualityRepository):
    def __init__(self, pool: asyncpg.Pool, logger):
        self.pool = pool
        self.logger = logger

    @classmethod
    async def create(cls, config: DatabaseConfig, logger) -> "PostgresQualityRepository":
        pool = await asyncpg.create_pool(config.dsn)
        return cls(pool, logger)

    async def close(self) -> None:
        await self.pool.close()

    async def get_pending_devices(
        self, limit: int, after_id: int = 0, recompute_all: bool = False
    ) -> list[DeviceRecord]:
        # Репутация устройства = агрегация по всем его листингам:
        #   review_count = сумма числа отзывов,
        #   rating       = средневзвешенный по числу отзывов рейтинг.
        quality_clause = "" if recompute_all else "AND d.quality IS NULL"
        query = f"""
            SELECT
                d.id,
                d.category,
                d.device_attributes,
                COALESCE(SUM(pls.extracted_review_count), 0) AS review_count,
                CASE
                    WHEN SUM(pls.extracted_review_count) > 0
                    THEN SUM(pls.extracted_rating * pls.extracted_review_count)
                         / SUM(pls.extracted_review_count)
                    ELSE NULL
                END AS rating
            FROM devices d
            LEFT JOIN listing_device_links ldl ON ldl.device_id = d.id
            LEFT JOIN llm_extracted_listings lel ON lel.id = ldl.llm_extracted_listing_id
            LEFT JOIN parsed_listing_snapshots pls ON pls.id = lel.parsed_listing_snapshot_id
            WHERE d.id > $1 {quality_clause}
            GROUP BY d.id
            ORDER BY d.id
            LIMIT $2
        """
        async with self.pool.acquire() as conn:
            rows = await conn.fetch(query, after_id, limit)

        records: list[DeviceRecord] = []
        for row in rows:
            attrs = row["device_attributes"]
            if isinstance(attrs, str):  # JSONB приходит строкой, если не настроен codec
                attrs = json.loads(attrs)
            rating = row["rating"]
            records.append(
                DeviceRecord(
                    id=row["id"],
                    category=row["category"],
                    device_attributes=attrs or {},
                    rating=float(rating) if rating is not None else None,
                    review_count=int(row["review_count"] or 0),
                )
            )
        return records

    async def save_quality(self, device_id: int, quality: float) -> None:
        async with self.pool.acquire() as conn:
            await conn.execute(
                "UPDATE devices SET quality = $1 WHERE id = $2",
                quality,
                device_id,
            )
