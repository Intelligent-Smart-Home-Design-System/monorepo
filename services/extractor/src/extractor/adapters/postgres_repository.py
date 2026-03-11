import json
import asyncpg
from extractor.ports.repository import ExtractionRepository
from extractor.domain.models import ExtractionSnapshot, ListingSnapshot
from extractor.config import DatabaseConfig


class PostgresExtractionRepository(ExtractionRepository):
    def __init__(self, pool: asyncpg.Pool, logger):
        self.pool = pool
        self.logger = logger

    @classmethod
    async def create(cls, config: DatabaseConfig, logger) -> PostgresExtractionRepository:
        pool = await asyncpg.create_pool(config.dsn)
        return cls(pool, logger)

    async def close(self):
        await self.pool.close()

    async def get_pending_snapshots(self, limit: int, offset: int) -> list[ListingSnapshot]:
        async with self.pool.acquire() as conn:
            rows = await conn.fetch("""
                WITH latest_per_page AS (
                    SELECT DISTINCT ON (ps.tracked_page)
                        pls.*
                    FROM parsed_listing_snapshots pls
                    JOIN page_snapshots ps ON ps.id = pls.page_snapshot_id
                    WHERE pls.processed = FALSE
                    ORDER BY ps.tracked_page, pls.parsed_at DESC
                )
                SELECT * FROM latest_per_page
                LIMIT $1
                OFFSET $2
            """, limit, offset)

            return [
                ListingSnapshot(
                    id=row["id"],
                    name=row["extracted_name"],
                    in_stock=row["extracted_in_stock"],
                    text=row["extracted_text"],
                    brand=row["extracted_brand"],
                    rating=row["extracted_rating"],
                    review_count=row["extracted_review_count"],
                    price=row["extracted_price"],
                    currency=row["extracted_currency"],
                    model_number=row["extracted_model_number"],
                    category=row["extracted_category"],
                    quantity=row["extracted_quantity"],
                )
                for row in rows
            ]

    async def save_extraction(self, extraction: ExtractionSnapshot) -> None:
        async with self.pool.acquire() as conn:
            async with conn.transaction():
                await conn.execute("""
                    INSERT INTO llm_extracted_listings (
                        parsed_listing_snapshot_id,
                        extracted_at,
                        brand,
                        model,
                        category,
                        category_confidence,
                        device_attributes,
                        llm_model
                    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
                """,
                    extraction.parsed_listing_snapshot_id,
                    extraction.extracted_at,
                    extraction.brand,
                    extraction.model,
                    extraction.category,
                    extraction.category_confidence,
                    json.dumps(extraction.device_attributes),
                    extraction.llm_model,
                )

                await conn.execute("""
                    UPDATE parsed_listing_snapshots 
                    SET processed = TRUE 
                    WHERE id = $1
                """, extraction.parsed_listing_snapshot_id)
