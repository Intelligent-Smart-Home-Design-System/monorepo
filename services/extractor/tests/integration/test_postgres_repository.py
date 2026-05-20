import pytest
from datetime import datetime, timezone
from extractor.domain.models import ExtractionSnapshot

pytestmark = pytest.mark.asyncio


class TestGetPendingSnapshots:
    
    async def test_returns_empty_when_no_snapshots(
        self, repository, clean_db
    ):
        result = await repository.get_pending_snapshots(limit=10)
        assert result == []

    async def test_returns_unprocessed_snapshots(
        self, repository, clean_db, db_pool
    ):
        # Arrange
        async with db_pool.acquire() as conn:
            await conn.execute("""
                INSERT INTO tracked_pages (id, source_name, page_type, url)
                VALUES (1, 'amazon', 'listing', 'https://amazon.com/product/123')
            """)
            await conn.execute("""
                INSERT INTO page_snapshots (id, tracked_page, scraped_at)
                VALUES (1, 1, NOW())
            """)
            await conn.execute("""
                INSERT INTO parsed_listing_snapshots (
                    id, page_snapshot_id, extracted_name, extracted_in_stock, extracted_text, extracted_brand, extracted_rating, extracted_review_count, parsed_at, processed
                )
                VALUES (1, 1, 'Smart Bulb', TRUE, 'A great smart bulb...', 'Philips', 4, 10, '2024-01-01', FALSE)
            """)

        # Act
        result = await repository.get_pending_snapshots(limit=10)

        # Assert
        assert len(result) == 1
        assert result[0].id == 1
        assert result[0].name == "Smart Bulb"
        assert result[0].brand == "Philips"

    async def test_ignores_already_processed(
        self, repository, clean_db, db_pool
    ):
        # Arrange
        async with db_pool.acquire() as conn:
            await conn.execute("""
                INSERT INTO tracked_pages (id, source_name, page_type, url)
                VALUES (1, 'amazon', 'listing', 'https://amazon.com/product/123')
            """)
            await conn.execute("""
                INSERT INTO page_snapshots (id, tracked_page, scraped_at)
                VALUES (1, 1, NOW())
            """)
            await conn.execute("""
                INSERT INTO parsed_listing_snapshots (
                    id, page_snapshot_id, extracted_name, extracted_in_stock, extracted_text, extracted_brand, extracted_rating, extracted_review_count, parsed_at, processed
                )
                VALUES (1, 1, 'Already done', TRUE, 'Something', 'IKEA', 4, 10, '2024-01-01', TRUE)
            """)

        # Act
        result = await repository.get_pending_snapshots(limit=10)

        # Assert
        assert result == []

    async def test_returns_multiple_snapshots(
        self, repository, clean_db, db_pool
    ):
        # Arrange: two snapshots for same page, different dates
        async with db_pool.acquire() as conn:
            await conn.execute("""
                INSERT INTO tracked_pages (id, source_name, page_type, url)
                VALUES (1, 'amazon', 'listing', 'https://amazon.com/product/123')
            """)
            await conn.execute("""
                INSERT INTO page_snapshots (id, tracked_page, scraped_at)
                VALUES 
                    (1, 1, '2024-01-01'),
                    (2, 1, '2024-01-02')
            """)
            await conn.execute("""
                INSERT INTO parsed_listing_snapshots (
                    id, page_snapshot_id, extracted_name, extracted_in_stock, extracted_text, extracted_brand, extracted_rating, extracted_review_count, parsed_at, processed
                )
                VALUES 
                    (1, 1, 'Old Snapshot', TRUE, 'Something', 'IKEA', 4, 10, '2024-01-01', FALSE),
                    (2, 2, 'New Snapshot', TRUE, 'Something', 'IKEA', 4, 10, '2024-01-02', FALSE)
            """)

        # Act
        result = await repository.get_pending_snapshots(limit=10)

        # Assert
        assert len(result) == 2
        assert result[0].name == "New Snapshot"
        assert result[1].name == "Old Snapshot"


class TestSaveExtraction:

    async def test_saves_extraction_and_marks_processed(
        self, repository, clean_db, db_pool
    ):
        # Arrange
        async with db_pool.acquire() as conn:
            await conn.execute("""
                INSERT INTO tracked_pages (id, source_name, page_type, url)
                VALUES (1, 'amazon', 'listing', 'https://amazon.com/product/123')
            """)
            await conn.execute("""
                INSERT INTO page_snapshots (id, tracked_page, scraped_at)
                VALUES (1, 1, NOW())
            """)
            await conn.execute("""
                INSERT INTO parsed_listing_snapshots (
                    id, page_snapshot_id, extracted_name, processed, extracted_text, extracted_in_stock, extracted_brand, extracted_rating, extracted_review_count
                )
                VALUES (1, 1, 'Smart Bulb', FALSE, 'Something', TRUE, 'Philips', 5, 100)
            """)

        extraction = ExtractionSnapshot(
            parsed_listing_snapshot_id=1,
            extracted_at=datetime.now(timezone.utc),
            brand="Philips",
            model="Hue White",
            category="smart_lamp",
            category_confidence=0.95,
            device_attributes={"brightness_lumens": 800, "rgb_support": False},
            llm_model="mistral-7b",
        )

        # Act
        await repository.save_extraction(extraction)

        # Assert: extraction saved
        async with db_pool.acquire() as conn:
            row = await conn.fetchrow("""
                SELECT * FROM llm_extracted_listings WHERE parsed_listing_snapshot_id = 1
            """)
            assert row is not None
            assert row["brand"] == "Philips"
            assert row["category"] == "smart_lamp"

            # Assert: marked as processed
            processed = await conn.fetchval("""
                SELECT processed FROM parsed_listing_snapshots WHERE id = 1
            """)
            assert processed is True

