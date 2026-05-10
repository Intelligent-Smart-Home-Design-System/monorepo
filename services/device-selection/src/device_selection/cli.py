from __future__ import annotations

import json

from device_selection.core.model import (
    DeviceRequirement,
    DeviceSelectionRequest,
    Filter,
    FilterOp,
)
from device_selection.solvers.enum_repair import SolverConfig, solve_enum_repair

import asyncio
from pathlib import Path

import asyncpg
import structlog
import typer

from device_selection.config import Settings
from device_selection.data.loader import CatalogLoader

app = typer.Typer()


def _setup_logging(settings: Settings) -> None:
    structlog.configure(
        processors=[
            structlog.processors.TimeStamper(fmt="iso"),
            structlog.processors.JSONRenderer()
            if settings.logging.format == "json"
            else structlog.dev.ConsoleRenderer(),
        ]
    )


@app.command()
def run(
    config_path: Path = typer.Option(
        Path("config.toml"),
        "--config", "-c",
        help="Path to config file",
    ),
) -> None:
    """Run the device selection service."""
    settings = Settings.from_toml(config_path)
    _setup_logging(settings)
    asyncio.run(_run(settings))


async def _run(settings: Settings) -> None:
    log = structlog.get_logger()

    pool = await asyncpg.create_pool(settings.database.dsn)
    try:
        loader = CatalogLoader(
            pool,
            calculate_quality=settings.quality.calculate,
            min_reviews=settings.quality.min_reviews,
            global_avg=settings.quality.global_avg_rating,
        )
        catalog = await loader.load()
        log.info("catalog loaded", device_count=len(catalog._devices_by_id))
    finally:
        await pool.close()


@app.command()
def test_selection(
    config_path: Path = typer.Option(
        Path("config.toml"),
        "--config", "-c",
        help="Path to config file",
    ),
    output_path: Path = typer.Option(
        Path("selection_result.json"),
        "--output", "-o",
        help="Path to output JSON file",
    ),
) -> None:
    """Run a test device selection request and export results to JSON."""
    settings = Settings.from_toml(config_path)
    _setup_logging(settings)
    asyncio.run(_test_selection(settings, output_path))


_GET_LISTING_URL_QUERY = """
SELECT
    tp.url,
    pls.extracted_price,
    pls.extracted_name,
    pls.extracted_brand,
    pls.extracted_review_count,
    pls.extracted_rating
FROM llm_extracted_listings l
JOIN parsed_listing_snapshots pls ON pls.id = l.parsed_listing_snapshot_id
JOIN page_snapshots ps ON ps.id = pls.page_snapshot_id
JOIN tracked_pages tp ON tp.id = ps.tracked_page
WHERE l.id = $1
"""


async def _test_selection(settings: Settings, output_path: Path) -> None:
    log = structlog.get_logger()

    pool = await asyncpg.create_pool(settings.database.dsn)
    try:
        loader = CatalogLoader(
            pool,
            calculate_quality=settings.quality.calculate,
            min_reviews=settings.quality.min_reviews,
            global_avg=settings.quality.global_avg_rating,
            rating_floor=settings.quality.rating_floor,
        )
        catalog = await loader.load()
        log.info("catalog loaded", device_count=len(catalog._devices_by_id))

        req = DeviceSelectionRequest(
            main_ecosystem="yandex",
            budget=10_000.0,
            requirements=(
                DeviceRequirement(
                    requirement_id=1,
                    device_type="smart_lamp",
                    count=3,
                    connect_to_main_ecosystem=True,
                    filters=(
                        Filter(field="socket_type", op=FilterOp.EQ, value="E27"),
                    ),
                ),
                DeviceRequirement(
                    requirement_id=2,
                    device_type="motion_sensor",
                    count=2,
                    connect_to_main_ecosystem=True,
                ),
                DeviceRequirement(
                    requirement_id=3,
                    device_type="water_leak_sensor",
                    count=5,
                    connect_to_main_ecosystem=True,
                ),
                DeviceRequirement(
                    requirement_id=4,
                    device_type="door_window_sensor",
                    count=3,
                    connect_to_main_ecosystem=True,
                ),
                DeviceRequirement(
                   requirement_id=5,
                   device_type="smart_lock",
                   count=1,
                   connect_to_main_ecosystem=False,
                ),
            ),
            exclude_ecosystems=["xiaomi", "digma"],
            max_solutions=10,
            time_budget_seconds=60.0,
        )

        solver_cfg = SolverConfig(
            max_bridge_ecosystems=settings.solver.max_bridge_ecosystems,
            max_hub_types=settings.solver.max_hub_types,
            max_candidates_per_type=settings.solver.max_candidates_per_type,
        )

        log.info("running selection")
        archive = solve_enum_repair(req, catalog, solver_cfg)
        points = list(archive.points)
        log.info("selection done", num_solutions=len(points))

        # enrich with listing urls
        result = await _build_result(points, pool, req)

        output_path.write_text(json.dumps(result, ensure_ascii=False, indent=2))
        log.info("results written", path=str(output_path))

    finally:
        await pool.close()


async def _build_result(
    points: list,
    pool: asyncpg.Pool,
    req: DeviceSelectionRequest,
) -> dict:
    async with pool.acquire() as conn:
        pareto_points = []
        for i, point in enumerate(points):
            items_out = []
            for item in point.items:
                listing_row = await conn.fetchrow(_GET_LISTING_URL_QUERY, item.device.source_listing_id)

                conn_direct = item.connection.connection_direct
                conn_final = item.connection.connection_final

                connection_out: dict = {
                    "direct": {
                        "ecosystem": conn_direct.ecosystem,
                        "protocol": conn_direct.protocol,
                        "hub_solution_item_id": conn_direct.hub_solution_item_id,
                    }
                }
                if conn_final is not None:
                    connection_out["final"] = {
                        "ecosystem": conn_final.ecosystem,
                        "protocol": conn_final.protocol,
                        "hub_solution_item_id": conn_final.hub_solution_item_id,
                    }

                items_out.append({
                    "item_id": item.id,
                    "requirement_id": item.requirement_id,
                    "device_id": item.device.device_id,
                    "device_type": item.device.device_type,
                    "brand": item.device.brand,
                    "model": item.device.model,
                    "quantity": item.quantity,
                    "unit_price": item.device.price,
                    "total_price": item.device.price * item.quantity,
                    "quality": round(item.device.quality, 4),
                    "connection": connection_out,
                    "listing": {
                        "url": listing_row["url"] if listing_row else None,
                        "name": listing_row["extracted_name"] if listing_row else None,
                        "brand": listing_row["extracted_brand"] if listing_row else None,
                        "price": listing_row["extracted_price"] if listing_row else None,
                        "rating": float(listing_row["extracted_rating"]) if listing_row else None,
                        "review_count": listing_row["extracted_review_count"] if listing_row else None,
                    } if listing_row else None,
                })

            pareto_points.append({
                "rank": i + 1,
                "total_cost": round(point.total_cost, 2),
                "avg_quality": round(point.avg_quality, 4),
                "num_ecosystems": point.num_ecosystems,
                "num_hubs": point.num_hubs,
                "items": items_out,
            })

        return {
            "request": {
                "main_ecosystem": req.main_ecosystem,
                "budget": req.budget,
                "requirements": [
                    {
                        "requirement_id": r.requirement_id,
                        "device_type": r.device_type,
                        "count": r.count,
                        "connect_to_main_ecosystem": r.connect_to_main_ecosystem,
                    }
                    for r in req.requirements
                ],
            },
            "num_solutions": len(points),
            "pareto_front": pareto_points,
        }


if __name__ == "__main__":
    app()
