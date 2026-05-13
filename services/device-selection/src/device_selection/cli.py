from __future__ import annotations

import json

from device_selection.core.model import (
    DeviceRequirement,
    DeviceSelectionRequest,
    Filter,
    FilterOp,
)
from device_selection.solvers.enum_repair import SolverConfig, solve_enum_repair
from device_selection.solvers.greedy import solve_greedy_cheapest, solve_greedy_quality

import asyncio
from pathlib import Path

import asyncpg
import structlog
import typer

import time
from typing import Any, Optional, Callable

from device_selection.config import Settings
from device_selection.data.loader import CatalogLoader
from device_selection.core.model import ConnectionInfo, ParetoPoint
from device_selection.core.pareto import ParetoArchive
from device_selection.core.validate import validate_solution
from device_selection.data.json_loader import load_test_case
from device_selection.solvers.brute_force import solve_brute_force
from device_selection.solvers.enum_repair import SolverConfig, solve_enum_repair
from device_selection.data.export import export_catalog

import csv
from device_selection.core.metrics import (
    best_known_front,
    hypervolume,
    igd,
    igd_plus,
)
from device_selection.core.pareto import ObjectiveBounds

SOLVERS: list[tuple[str, Callable[[DeviceSelectionRequest, Catalog], ParetoArchive]]] = [
    ("enum_repair",      lambda req, cat: solve_enum_repair(req, cat, SolverConfig())),
    ("greedy_cheapest",  solve_greedy_cheapest),
    ("greedy_quality",   solve_greedy_quality),
]
_HV_REF = (1.05, 1.05, 1.05)

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
        for i in range(1000, 60000, 1000):
            req = DeviceSelectionRequest(
                main_ecosystem="yandex",
                budget=25000,
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
                exclude_ecosystems=[],
                max_solutions=10,
                time_budget_seconds=60.0,
            )

            solver_cfg = SolverConfig(
                max_bridge_ecosystems=settings.solver.max_bridge_ecosystems,
                max_hub_types=settings.solver.max_hub_types,
                max_candidates_per_type=settings.solver.max_candidates_per_type,
            )

            #log.info("running selection")
            archive = solve_enum_repair(req, catalog, solver_cfg)
            points = list(archive.points)

            log.info("selection done", num_solutions=len(points))
            # enrich with listing urls
            result = await _build_result(points, pool, req)

            (output_path.with_suffix('.' + str(i) + '.json')).write_text(json.dumps(result, ensure_ascii=False, indent=2))
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


def _conn_to_dict(info: Optional[ConnectionInfo]) -> Optional[dict[str, Any]]:
    if info is None:
        return None
    return {
        "ecosystem": info.ecosystem,
        "protocol": info.protocol,
        "hub_solution_item_id": info.hub_solution_item_id,
    }
 
 
def _point_to_dict(point: ParetoPoint) -> dict[str, Any]:
    return {
        "total_cost": point.total_cost,
        "avg_quality": point.avg_quality,
        "num_ecosystems": point.num_ecosystems,
        "num_hubs": point.num_hubs,
        "items": [
            {
                "id": item.id,
                "device_id": item.device.device_id,
                "device_type": item.device.device_type,
                "brand": item.device.brand,
                "model": item.device.model,
                "requirement_id": item.requirement_id,
                "quantity": item.quantity,
                "price": item.device.price,
                "quality": item.device.quality,
                "connection": {
                    "direct": _conn_to_dict(item.connection.connection_direct),
                    "final": _conn_to_dict(item.connection.connection_final),
                },
            }
            for item in point.items
        ],
    }
 
 
def _dump_results(
    results_dir: Path,
    test_id: str,
    solver_name: str,
    archive: ParetoArchive,
    runtime_s: float,
) -> None:
    results_dir.mkdir(parents=True, exist_ok=True)
    points = list(archive.points)
    payload = {
        "test_id": test_id,
        "solver": solver_name,
        "runtime_s": runtime_s,
        "num_solutions": len(points),
        "pareto_front": [_point_to_dict(p) for p in points],
    }
    out_path = results_dir / f"{test_id}_{solver_name}.json"
    out_path.write_text(json.dumps(payload, ensure_ascii=False, indent=2))
 
 
def _print_front(solver_name: str, req: DeviceSelectionRequest, archive: ParetoArchive, runtime_s: float) -> None:
    points = sorted(archive.points, key=lambda p: p.total_cost)
    print(f"  [{solver_name}] {len(points)} solutions in {runtime_s:.2f}s")
    for p in points:
        print(
            f"    cost={p.total_cost:>10.2f}  quality={p.avg_quality:.4f}  "
            f"ecos={p.num_ecosystems}  hubs={p.num_hubs}"
        )
        errs = validate_solution(req, p)
        if len(errs) > 0:
            print(f"Errors: {errs}")
 

def bounds_from_catalog(catalog: Catalog) -> ObjectiveBounds:
    """
    Derive objective bounds from a catalog instance.

    eco_max  = total number of distinct ecosystems available.
    hub_max  = number of ecosystems that have at least one hub type
               (since we pick at most one hub per ecosystem).
    hub_min  = 0 always (no hubs is a valid solution).
    eco_min  = 1 always (at least one ecosystem needed to place any device).

    If no ecosystem has a hub type at all, hub_min == hub_max == 0,
    which _norm() handles safely (returns 0.0 for every point, i.e.
    all points are equal in that dimension — correct, since hubs are
    irrelevant for that catalog).
    """
    ecosystems = catalog.available_ecosystems()
    eco_max    = len(ecosystems)

    hub_max = sum(
        1
        for eco in ecosystems
        if len(catalog.available_hub_types_for_ecosystem(eco)) > 0
    )

    return ObjectiveBounds(
        q_min   = 0.0,
        q_max   = 1.0,
        eco_min = 1,
        eco_max = max(eco_max, 1),   # avoid lo==hi if catalog is empty
        hub_min = 0,
        hub_max = hub_max,           # 0 is fine — _norm() returns 0.0 when lo==hi
    )


def _compute_metrics_row(
    test_id: str,
    solver_name: str,
    archive: ParetoArchive,
    runtime_s: float,
    ref_archive: ParetoArchive,
    ref_source: str,
    bounds: ObjectiveBounds,
) -> dict:
    hv_val       = hypervolume(archive, bounds, _HV_REF)
    igd_val      = igd(archive, ref_archive, bounds)      if ref_archive.points else None
    igd_plus_val = igd_plus(archive, ref_archive, bounds) if ref_archive.points else None

    return {
        "test_id":          test_id,
        "solver":           solver_name,
        "num_solutions":    archive.front_size(),
        "runtime_s":        round(runtime_s, 4),
        "hv":               round(hv_val, 6),
        "igd":              round(igd_val,      6) if igd_val      is not None else "",
        "igd_plus":         round(igd_plus_val, 6) if igd_plus_val is not None else "",
        "reference_source": ref_source,
    }


_CSV_FIELDS = [
    "test_id",
    "solver",
    "num_solutions",
    "runtime_s",
    "hv",
    "igd",
    "igd_plus",
    "reference_source",
]


def _write_summary_csv(rows: list[dict], out_path: Path) -> None:
    out_path.parent.mkdir(parents=True, exist_ok=True)
    with out_path.open("w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fieldnames=_CSV_FIELDS)
        writer.writeheader()
        writer.writerows(rows)
    print(f"\nSummary CSV written to {out_path}")


@app.command()
def evaluate(
    eval_dir: Path = typer.Option(
        Path("evaluation"),
        "--eval-dir", "-d",
        help="Directory containing test_*.json files",
    ),
    results_dir: Path = typer.Option(
        Path("evaluation/results"),
        "--results-dir", "-r",
        help="Directory to write per-test result JSONs",
    ),
    pattern: str = typer.Option(
        "test_*.json",
        "--pattern", "-p",
        help="Glob pattern for test files",
    ),
) -> None:
    """Run enum_repair (and brute_force where enabled) on all test cases."""
    test_files = sorted(eval_dir.glob(pattern))
    if not test_files:
        print(f"No test files matching {pattern!r} in {eval_dir}")
        return

    all_rows: list[dict] = []

    for test_file in test_files:
        tc = load_test_case(test_file)
        bounds = bounds_from_catalog(tc.catalog)
        print(f"\n=== {tc.id} ===")
        print(f"  {tc.description}")

        # ---- run regular solvers ----
        solver_results: list[tuple[str, ParetoArchive, float]] = []
        for name, solve_fn in SOLVERS:
            t0 = time.perf_counter()
            archive = solve_fn(tc.request, tc.catalog)
            elapsed = time.perf_counter() - t0
            _print_front(name, tc.request, archive, elapsed)
            _dump_results(results_dir, tc.id, name, archive, elapsed)
            solver_results.append((name, archive, elapsed))

        # ---- brute force (optional ground truth) ----
        bf_archive: ParetoArchive | None = None
        if tc.run_brute_force:
            t0 = time.perf_counter()
            bf_archive = solve_brute_force(tc.request, tc.catalog)
            bf_time = time.perf_counter() - t0
            _print_front("brute_force", tc.request, bf_archive, bf_time)
            _dump_results(results_dir, tc.id, "brute_force", bf_archive, bf_time)

        # ---- reference front for IGD ----
        if bf_archive is not None:
            ref_archive = bf_archive
            ref_source  = "brute_force"
        else:
            ref_archive = best_known_front([arch for _, arch, _ in solver_results])
            ref_source  = "best_known_union"

        # ---- metrics rows ----
        for name, archive, elapsed in solver_results:
            all_rows.append(_compute_metrics_row(
                tc.id, name, archive, elapsed, ref_archive, ref_source, bounds
            ))
        if bf_archive is not None:
            all_rows.append(_compute_metrics_row(
                tc.id, "brute_force", bf_archive, bf_time, ref_archive, ref_source, bounds
            ))

    # ---- write summary ----
    _write_summary_csv(all_rows, results_dir / "summary.csv")


async def _export_catalog(settings: Settings, output_path: Path) -> None:
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
        n = export_catalog(catalog, output_path)
        log.info("catalog exported", path=str(output_path), num_devices=n)
    finally:
        await pool.close()


@app.command()
def export_real_catalog(
    config_path: Path = typer.Option(
        Path("config.toml"),
        "--config", "-c",
        help="Path to config file",
    ),
    output_path: Path = typer.Option(
        Path("real_catalog.json"),
        "--output", "-o",
        help="Path for the output catalog JSON",
    ),
) -> None:
    """Export the full device catalog to JSON for use in evaluation test cases."""
    settings = Settings.from_toml(config_path)
    _setup_logging(settings)
    asyncio.run(_export_catalog(settings, output_path))


if __name__ == "__main__":
    app()
