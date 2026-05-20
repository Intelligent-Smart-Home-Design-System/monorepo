import pytest
import asyncio
import asyncpg
from pathlib import Path
from testcontainers.postgres import PostgresContainer
from extractor.adapters.postgres_repository import PostgresExtractionRepository

MIGRATIONS_DIR = Path(__file__).parent.parent.parent.parent / "db/catalog/migrations"


@pytest.fixture(scope="session")
def postgres_container():
    """Spin up Postgres once per test session"""
    with PostgresContainer("postgres:16-alpine") as pg:
        yield pg


@pytest.fixture(scope="session")
async def db_pool(postgres_container):
    """Create pool and run migrations"""
    pool = await asyncpg.create_pool(
        host=postgres_container.get_container_host_ip(),
        port=postgres_container.get_exposed_port(5432),
        user=postgres_container.username,
        password=postgres_container.password,
        database=postgres_container.dbname,
    )

    files = sorted(MIGRATIONS_DIR.glob("*.up.sql"))
    print("migration files:", [f.name for f in files])

    async with pool.acquire() as conn:
        for migration_file in sorted(MIGRATIONS_DIR.glob("*.up.sql")):
            sql = migration_file.read_text()
            await conn.execute(sql)
    
    yield pool
    await pool.close()


@pytest.fixture
async def repository(db_pool):
    """Fresh repository per test"""
    repo = PostgresExtractionRepository(db_pool, None)
    yield repo


@pytest.fixture
async def clean_db(db_pool):
    """Clean tables before each test"""
    async with db_pool.acquire() as conn:
        await conn.execute("""
            TRUNCATE TABLE 
                llm_extracted_listings,
                parsed_listing_snapshots,
                page_snapshots,
                tracked_pages
            CASCADE
        """)
    yield

