from __future__ import annotations

from pathlib import Path

import asyncpg
import pytest
from testcontainers.postgres import PostgresContainer

from quality_calculator.adapters.postgres_repository import PostgresQualityRepository

# services/quality-calculator/tests/conftest.py -> вверх до корня монорепо -> db/catalog/migrations
MIGRATIONS_DIR = Path(__file__).parent.parent.parent.parent / "db/catalog/migrations"


@pytest.fixture(scope="session")
def postgres_container():
    with PostgresContainer("postgres:16-alpine") as pg:
        yield pg


@pytest.fixture(scope="session")
async def db_pool(postgres_container):
    pool = await asyncpg.create_pool(
        host=postgres_container.get_container_host_ip(),
        port=postgres_container.get_exposed_port(5432),
        user=postgres_container.username,
        password=postgres_container.password,
        database=postgres_container.dbname,
    )
    async with pool.acquire() as conn:
        for migration_file in sorted(MIGRATIONS_DIR.glob("*.up.sql")):
            await conn.execute(migration_file.read_text())
    yield pool
    await pool.close()


@pytest.fixture
async def repository(db_pool):
    yield PostgresQualityRepository(db_pool, None)


@pytest.fixture
async def clean_db(db_pool):
    async with db_pool.acquire() as conn:
        await conn.execute(
            """
            TRUNCATE TABLE
                listing_device_links,
                direct_compatibility,
                bridge_ecosystem_compatibility,
                devices,
                llm_extracted_listings,
                parsed_listing_snapshots,
                page_snapshots,
                tracked_pages
            RESTART IDENTITY CASCADE
            """
        )
    yield
