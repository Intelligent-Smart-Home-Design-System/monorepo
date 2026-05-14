from __future__ import annotations

import time
from dataclasses import dataclass

import asyncpg

import structlog
from temporalio import activity

from device_selection.config import Settings
from device_selection.data.loader import CatalogLoader
from device_selection.data.catalog import Catalog

log = structlog.get_logger()


# --------------------------------------------------------------------------- #
# Shared state                                                                 #
# --------------------------------------------------------------------------- #

@dataclass
class ActivityState:
    pool: asyncpg.Pool
    settings: Settings


_state: ActivityState | None = None


def init_activity_state(state: ActivityState) -> None:
    global _state
    _state = state


def _get_state() -> ActivityState:
    if _state is None:
        raise RuntimeError("Activity state not initialised")
    return _state


# --------------------------------------------------------------------------- #
# Catalog cache                                                                #
# --------------------------------------------------------------------------- #

@dataclass
class _CatalogCache:
    catalog: Catalog
    loaded_at: float


_catalog_cache: _CatalogCache | None = None


async def _get_catalog(state: ActivityState) -> object:
    global _catalog_cache
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


# --------------------------------------------------------------------------- #
# Input / output                                                               #
# --------------------------------------------------------------------------- #

@dataclass
class SolveInput:
    request_proto_bytes: bytes


@dataclass
class SolveOutput:
    response_proto_bytes: bytes


# --------------------------------------------------------------------------- #
# Activity                                                                     #
# --------------------------------------------------------------------------- #

@activity.defn(name="select_devices")
async def select_devices(inp: SolveInput) -> SolveOutput:
    from device_selection.proto import iot_opt_pb2 as pb
    from device_selection.temporal.codec import request_from_proto, response_to_proto
    from device_selection.solvers.enum_repair import SolverConfig, solve_enum_repair

    state = _get_state()
    s = state.settings

    proto_req = pb.DeviceSelectionRequest()
    proto_req.ParseFromString(inp.request_proto_bytes)
    req = request_from_proto(proto_req)

    activity.logger.info(
        "selection request received",
        main_ecosystem=req.main_ecosystem,
        budget=req.budget,
        num_requirements=len(req.requirements),
    )

    catalog = await _get_catalog(state)

    solver_cfg = SolverConfig(
        max_bridge_ecosystems=s.solver.max_bridge_ecosystems,
        max_hub_types=s.solver.max_hub_types,
        max_candidates_per_type=s.solver.max_candidates_per_type,
    )
    archive = solve_enum_repair(req, catalog, solver_cfg)
    points = sorted(archive.points, key=lambda p: p.total_cost)

    activity.logger.info("solver finished", num_solutions=len(points))

    response_proto = response_to_proto(points)
    return SolveOutput(response_proto_bytes=response_proto.SerializeToString())
