from __future__ import annotations

import asyncio
import json
import time
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import asyncpg
import structlog
from opentelemetry import trace
from temporalio import activity

from device_selection.config import Settings
from device_selection.data.catalog import Catalog
from device_selection.data.json_loader import load_request, load_test_case
from device_selection.data.loader import CatalogLoader
from device_selection.temporal.observability import Metrics

log = structlog.get_logger()


@dataclass
class ActivityState:
    pool: asyncpg.Pool | None
    settings: Settings
    metrics: Metrics
    semaphore: asyncio.Semaphore
    service_name: str


_state: ActivityState | None = None


def init_activity_state(state: ActivityState) -> None:
    global _state
    _state = state


def _get_state() -> ActivityState:
    if _state is None:
        raise RuntimeError("Activity state not initialised")
    return _state


@dataclass
class _CatalogCache:
    catalog: Catalog
    loaded_at: float


_catalog_cache: _CatalogCache | None = None


async def _get_catalog(state: ActivityState) -> Catalog:
    global _catalog_cache
    if state.pool is None:
        raise RuntimeError("Database pool is not configured for catalog loading")

    ttl = state.settings.catalog_ttl_seconds
    now = time.monotonic()
    if _catalog_cache is None or (now - _catalog_cache.loaded_at) > ttl:
        s = state.settings
        loader = CatalogLoader(
            state.pool,
            calculate_quality=s.quality.calculate,
            min_reviews=s.quality.min_reviews,
            global_avg=s.quality.global_avg_rating,
            rating_floor=s.quality.rating_floor,
        )
        catalog = await loader.load()
        _catalog_cache = _CatalogCache(catalog=catalog, loaded_at=now)
        log.info("catalog (re)loaded", device_count=len(catalog._devices_by_id))
    return _catalog_cache.catalog


@dataclass
class SolveInput:
    request_proto_bytes: bytes


@dataclass
class SolveOutput:
    response_proto_bytes: bytes


@dataclass
class SolveFromFileInput:
    request_id: str
    request_path: str
    output_path: str


@dataclass
class SolveFromFileOutput:
    request_id: str
    output_path: str
    solution_count: int
    best_total_cost: float
    catalog_source: str


def _bind_activity_logger(request_id: str) -> structlog.BoundLogger:
    info = activity.info()
    return log.bind(
        request_id=request_id,
        workflow_id=info.workflow_id,
        workflow_run_id=info.workflow_run_id,
        activity_id=info.activity_id,
        activity_type=info.activity_type,
        task_queue=info.task_queue,
    )


def _conn_to_dict(info) -> dict[str, Any] | None:
    if info is None:
        return None
    return {
        "ecosystem": info.ecosystem,
        "protocol": info.protocol,
        "hub_solution_item_id": info.hub_solution_item_id,
    }


def _point_to_dict(point) -> dict[str, Any]:
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


async def _solve(req, catalog: Catalog, state: ActivityState, activity_name: str, logger: structlog.BoundLogger):
    from device_selection.solvers.enum_repair import SolverConfig, solve_enum_repair

    s = state.settings
    solver_cfg = SolverConfig(
        max_bridge_ecosystems=s.solver.max_bridge_ecosystems,
        max_hub_types=s.solver.max_hub_types,
        max_candidates_per_type=s.solver.max_candidates_per_type,
    )
    archive = solve_enum_repair(req, catalog, solver_cfg)
    points = sorted(archive.points, key=lambda p: p.total_cost)
    logger.info("solver finished", activity=activity_name, num_solutions=len(points))
    return points


async def _run_activity(activity_name: str, request_id: str, handler):
    state = _get_state()
    tracer = trace.get_tracer(state.service_name)
    logger = _bind_activity_logger(request_id)
    start = time.perf_counter()
    state.metrics.concurrent_runs.labels(activity_name).inc()

    try:
        async with state.semaphore:
            with tracer.start_as_current_span(f"device_selection.{activity_name}") as span:
                span.set_attribute("request.id", request_id)
                span_context = span.get_span_context()
                logger = logger.bind(
                    trace_id=f"{span_context.trace_id:032x}",
                    span_id=f"{span_context.span_id:016x}",
                )
                result = await handler(state, logger)
                duration = time.perf_counter() - start
                state.metrics.runs_total.labels(activity_name, "success").inc()
                state.metrics.duration_seconds.labels(activity_name).observe(duration)
                return result
    except Exception:
        duration = time.perf_counter() - start
        state.metrics.runs_total.labels(activity_name, "failure").inc()
        state.metrics.duration_seconds.labels(activity_name).observe(duration)
        logger.exception("device-selection activity failed", activity=activity_name, duration_seconds=duration)
        raise
    finally:
        state.metrics.concurrent_runs.labels(activity_name).dec()


@activity.defn(name="select_devices")
async def select_devices(inp: SolveInput) -> SolveOutput:
    from device_selection.proto import iot_opt_pb2 as pb
    from device_selection.temporal.codec import request_from_proto, response_to_proto

    async def handler(state: ActivityState, logger: structlog.BoundLogger) -> SolveOutput:
        proto_req = pb.DeviceSelectionRequest()
        proto_req.ParseFromString(inp.request_proto_bytes)
        req = request_from_proto(proto_req)

        logger.info(
            "selection request received",
            activity="select_devices",
            main_ecosystem=req.main_ecosystem,
            budget=req.budget,
            num_requirements=len(req.requirements),
        )

        catalog = await _get_catalog(state)
        points = await _solve(req, catalog, state, "select_devices", logger)
        response_proto = response_to_proto(points)
        return SolveOutput(response_proto_bytes=response_proto.SerializeToString())

    return await _run_activity("select_devices", "proto-request", handler)


@activity.defn(name="device_selection.select_devices_from_file")
async def select_devices_from_file(inp: SolveFromFileInput) -> SolveFromFileOutput:
    async def handler(state: ActivityState, logger: structlog.BoundLogger) -> SolveFromFileOutput:
        request_path = Path(inp.request_path)
        output_path = Path(inp.output_path)
        output_path.parent.mkdir(parents=True, exist_ok=True)

        activity.heartbeat("started")
        payload = json.loads(request_path.read_text(encoding="utf-8"))
        if "request" in payload and "catalog" in payload:
            testcase = load_test_case(request_path)
            req = testcase.request
            catalog = testcase.catalog
            catalog_source = "test_case"
        else:
            req = load_request(payload)
            catalog = await _get_catalog(state)
            catalog_source = "database"

        logger.info(
            "selection request loaded",
            activity="select_devices_from_file",
            request_path=str(request_path),
            catalog_source=catalog_source,
            budget=req.budget,
            num_requirements=len(req.requirements),
        )

        points = await _solve(req, catalog, state, "select_devices_from_file", logger)
        artifact = {
            "request_id": inp.request_id,
            "catalog_source": catalog_source,
            "solution_count": len(points),
            "pareto_front": [_point_to_dict(point) for point in points],
        }
        output_path.write_text(json.dumps(artifact, ensure_ascii=False, indent=2), encoding="utf-8")
        activity.heartbeat("completed")

        return SolveFromFileOutput(
            request_id=inp.request_id,
            output_path=str(output_path),
            solution_count=len(points),
            best_total_cost=points[0].total_cost if points else 0.0,
            catalog_source=catalog_source,
        )

    return await _run_activity("select_devices_from_file", inp.request_id, handler)
