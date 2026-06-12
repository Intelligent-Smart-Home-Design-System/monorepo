from __future__ import annotations

import json

import pytest

from quality_calculator.domain.models import DeviceRecord
from quality_calculator.evaluator import QualityEvaluator
from quality_calculator.worker.worker import Worker

pytestmark = pytest.mark.asyncio


# --- helper: вставка устройства с листингами через всю цепочку слоёв ---

async def insert_device(
    conn,
    *,
    category: str,
    device_attributes: dict,
    listings: list[tuple[float, int]],  # (rating, review_count) на листинг
    quality: float | None = None,
    taxonomy_version: str = "v1",
) -> int:
    page = await conn.fetchval(
        "INSERT INTO tracked_pages (source_name, page_type, url) "
        "VALUES ('yandex', 'listing', 'https://market.yandex.ru/' || gen_random_uuid()) RETURNING id"
    )
    snap = await conn.fetchval(
        "INSERT INTO page_snapshots (tracked_page) VALUES ($1) RETURNING id", page
    )
    device_id = await conn.fetchval(
        "INSERT INTO devices (brand, category, device_attributes, taxonomy_version) "
        "VALUES ('brand', $1, $2::jsonb, $3) RETURNING id",
        category, json.dumps(device_attributes), taxonomy_version,
    )
    if quality is not None:
        await conn.execute("UPDATE devices SET quality = $1 WHERE id = $2", quality, device_id)

    for rating, count in listings:
        pls = await conn.fetchval(
            "INSERT INTO parsed_listing_snapshots "
            "(page_snapshot_id, extracted_in_stock, extracted_text, extracted_name, "
            " extracted_brand, extracted_rating, extracted_review_count) "
            "VALUES ($1, TRUE, 'text', 'name', 'brand', $2, $3) RETURNING id",
            snap, rating, count,
        )
        lel = await conn.fetchval(
            "INSERT INTO llm_extracted_listings "
            "(parsed_listing_snapshot_id, brand, category, device_attributes, taxonomy_version, llm_model) "
            "VALUES ($1, 'brand', $2, $3::jsonb, $4, 'test') RETURNING id",
            pls, category, json.dumps(device_attributes), taxonomy_version,
        )
        await conn.execute(
            "INSERT INTO listing_device_links (llm_extracted_listing_id, device_id) VALUES ($1, $2)",
            lel, device_id,
        )
    return device_id


# --- минимальная схема/стратегия для e2e (не зависим от файлов в репо) ---

TECH_SCHEMA = {
    "traits": {"lighting": {"properties": {"brightness_lm": {"minimum": 400, "maximum": 1500}}}},
    "types": {"smart_lamp": {"traits": ["lighting"]}},
}
STRATEGY = {"lighting": {"brightness_lm": {"weight": 1.0}}}


def make_evaluator() -> QualityEvaluator:
    return QualityEvaluator(TECH_SCHEMA, STRATEGY, weights=(0.5, 0.3, 0.2), reputation_mode="bayesian")


class TestGetPendingDevices:
    async def test_returns_empty_when_no_devices(self, repository, clean_db):
        assert await repository.get_pending_devices(limit=10) == []

    async def test_returns_device_with_aggregated_reputation(self, repository, clean_db, db_pool):
        async with db_pool.acquire() as conn:
            dev_id = await insert_device(
                conn, category="smart_lamp",
                device_attributes={"brightness_lm": 1000, "protocol": ["zigbee"]},
                listings=[(4.8, 100)],
            )
        result = await repository.get_pending_devices(limit=10)
        assert len(result) == 1
        rec = result[0]
        assert rec.id == dev_id
        assert rec.category == "smart_lamp"
        assert rec.device_attributes["brightness_lm"] == 1000
        assert rec.review_count == 100
        assert rec.rating == pytest.approx(4.8)

    async def test_weighted_rating_across_listings(self, repository, clean_db, db_pool):
        # 5.0*10 + 4.0*90 = 50 + 360 = 410; /100 = 4.1; count = 100
        async with db_pool.acquire() as conn:
            await insert_device(
                conn, category="smart_lamp",
                device_attributes={"brightness_lm": 800},
                listings=[(5.0, 10), (4.0, 90)],
            )
        rec = (await repository.get_pending_devices(limit=10))[0]
        assert rec.review_count == 100
        assert rec.rating == pytest.approx(4.1)

    async def test_ignores_scored_unless_recompute(self, repository, clean_db, db_pool):
        async with db_pool.acquire() as conn:
            await insert_device(
                conn, category="smart_lamp", device_attributes={"brightness_lm": 800},
                listings=[(4.5, 50)], quality=0.7,
            )
        assert await repository.get_pending_devices(limit=10) == []
        assert len(await repository.get_pending_devices(limit=10, recompute_all=True)) == 1

    async def test_keyset_pagination(self, repository, clean_db, db_pool):
        async with db_pool.acquire() as conn:
            ids = [
                await insert_device(conn, category="smart_lamp",
                                    device_attributes={"brightness_lm": 800}, listings=[(4.5, 50)])
                for _ in range(3)
            ]
        first = await repository.get_pending_devices(limit=2, after_id=0)
        assert [r.id for r in first] == ids[:2]
        rest = await repository.get_pending_devices(limit=2, after_id=ids[1])
        assert [r.id for r in rest] == [ids[2]]


class TestSaveQuality:
    async def test_save_quality_updates_column(self, repository, clean_db, db_pool):
        async with db_pool.acquire() as conn:
            dev_id = await insert_device(
                conn, category="smart_lamp", device_attributes={"brightness_lm": 800},
                listings=[(4.5, 50)],
            )
        await repository.save_quality(dev_id, 0.83)
        async with db_pool.acquire() as conn:
            q = await conn.fetchval("SELECT quality FROM devices WHERE id = $1", dev_id)
        assert q == pytest.approx(0.83)


class TestWorkerEndToEnd:
    async def test_fills_quality_for_pending_devices(self, repository, clean_db, db_pool):
        async with db_pool.acquire() as conn:
            scoreable = await insert_device(
                conn, category="smart_lamp",
                device_attributes={"brightness_lm": 1400, "protocol": ["zigbee"]},
                listings=[(4.9, 500)],
            )
            # без отзывов и без распознаваемых спеков -> quality посчитать нельзя
            unscoreable = await insert_device(
                conn, category="smart_lamp", device_attributes={}, listings=[],
            )

        worker = Worker(make_evaluator(), repository, batch_size=10)
        stats = await worker.run()

        assert stats["scored"] == 1
        assert stats["skipped"] == 1
        async with db_pool.acquire() as conn:
            q_ok = await conn.fetchval("SELECT quality FROM devices WHERE id = $1", scoreable)
            q_none = await conn.fetchval("SELECT quality FROM devices WHERE id = $1", unscoreable)
        assert q_ok is not None and 0.0 < q_ok <= 1.0
        assert q_none is None  # остаётся NULL — честно, сигналов нет

    async def test_idempotent_second_run_scores_nothing(self, repository, clean_db, db_pool):
        async with db_pool.acquire() as conn:
            await insert_device(
                conn, category="smart_lamp",
                device_attributes={"brightness_lm": 1000}, listings=[(4.7, 200)],
            )
        worker = Worker(make_evaluator(), repository, batch_size=10)
        assert (await worker.run())["scored"] == 1
        # второй прогон: устройство уже с quality -> не выбирается
        assert (await worker.run())["scored"] == 0
