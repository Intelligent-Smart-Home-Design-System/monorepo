from __future__ import annotations

import pytest

from device_selection.core.model import (
    Device,
    DirectCompat,
)
from device_selection.data.catalog import InMemoryCatalog
from device_selection.temporal.activities import (
    ActivityState,
    SolveInput,
    SolveOutput,
    init_activity_state,
    solve_device_selection,
)
from device_selection.config import Settings
from device_selection.proto import iot_opt_pb2 as pb
from google.protobuf.struct_pb2 import Value
from temporalio.testing import ActivityEnvironment


# --------------------------------------------------------------------------- #
# Minimal catalog — one wifi lamp, directly on google/wifi                    #
# --------------------------------------------------------------------------- #

WIFI_LAMP = Device(
    device_id=1,
    device_type="smart_lamp",
    brand="test",
    model="wifi-lamp-1",
    attributes={"socket_type": "E27"},
    price=500.0,
    quality=0.6,
    source_listing_id=1,
    direct_compat=(DirectCompat("google", "wifi"),),
    bridge_compat=(),
)


@pytest.fixture()
def catalog() -> InMemoryCatalog:
    return InMemoryCatalog([WIFI_LAMP])


# --------------------------------------------------------------------------- #
# Fake Settings — no real DB or Temporal needed                               #
# --------------------------------------------------------------------------- #

@pytest.fixture()
def settings() -> Settings:
    return Settings(
        database=dict(host="localhost", port=5432, name="x", user="x", password="x"),
    )


# --------------------------------------------------------------------------- #
# Wire up the activity state with a fake pool that never gets called          #
# --------------------------------------------------------------------------- #

class _FakePool:
    """Stands in for asyncpg.Pool. The activity uses the pre-warmed catalog
    cache so pool.acquire() is never actually called in these tests."""
    pass


@pytest.fixture(autouse=True)
def _patch_activity_state(catalog, settings, monkeypatch):
    """
    Initialise the module-level activity state and pre-warm the catalog cache
    so the activity never tries to hit the database.
    """
    import time
    import device_selection.temporal.activities as act_module

    init_activity_state(ActivityState(pool=_FakePool(), settings=settings))

    # pre-warm the cache so _get_catalog() returns our InMemoryCatalog
    act_module._catalog_cache = act_module._CatalogCache(
        catalog=catalog,
        loaded_at=time.monotonic(),
    )
    yield
    # reset between tests
    act_module._catalog_cache = None
    act_module._state = None


# --------------------------------------------------------------------------- #
# Helpers                                                                     #
# --------------------------------------------------------------------------- #

def _make_request(
    *,
    budget: float = 10_000.0,
    requirements: list[pb.DeviceRequirement],
    max_solutions: int = 5,
    time_budget_seconds: float = 10.0,
) -> bytes:
    req = pb.DeviceSelectionRequest(
        main_ecosystem="google",
        budget=budget,
        device_requirements=requirements,
        max_solutions=max_solutions,
        time_budget_seconds=time_budget_seconds,
    )
    return req.SerializeToString()


def _parse_response(raw: bytes) -> pb.DeviceSelectionResponse:
    resp = pb.DeviceSelectionResponse()
    resp.ParseFromString(raw)
    return resp


# --------------------------------------------------------------------------- #
# Tests                                                                       #
# --------------------------------------------------------------------------- #

class TestSolveDeviceSelectionActivity:

    @pytest.mark.asyncio
    async def test_returns_at_least_one_solution(self):
        """Happy path: one lamp requirement, lamp is in catalog, budget is ample."""
        raw = _make_request(
            requirements=[
                pb.DeviceRequirement(
                    requirement_id=1,
                    device_type="smart_lamp",
                    count=1,
                    connect_to_main_ecosystem=True,
                ),
            ],
        )
        env = ActivityEnvironment()
        result: SolveOutput = await env.run(solve_device_selection, SolveInput(raw))

        resp = _parse_response(result.response_proto_bytes)
        assert len(resp.pareto_front) > 0


    @pytest.mark.asyncio
    async def test_filter_matching_attribute_finds_solution(self):
        """Filter on socket_type=E27 should still find the wifi lamp."""
        raw = _make_request(
            requirements=[
                pb.DeviceRequirement(
                    requirement_id=1,
                    device_type="smart_lamp",
                    count=1,
                    connect_to_main_ecosystem=True,
                    filters=[
                        pb.Filter(
                            field="socket_type",
                            op=pb.OP_EQ,
                            value=Value(string_value="E27"),
                        )
                    ],
                ),
            ],
        )
        env = ActivityEnvironment()
        result: SolveOutput = await env.run(solve_device_selection, SolveInput(raw))

        resp = _parse_response(result.response_proto_bytes)
        assert len(resp.pareto_front) > 0

    @pytest.mark.asyncio
    async def test_filter_no_match_returns_no_solutions(self):
        """Filter on socket_type=E14 should match nothing — expect empty front."""
        raw = _make_request(
            requirements=[
                pb.DeviceRequirement(
                    requirement_id=1,
                    device_type="smart_lamp",
                    count=1,
                    connect_to_main_ecosystem=True,
                    filters=[
                        pb.Filter(
                            field="socket_type",
                            op=pb.OP_EQ,
                            value=Value(string_value="E14"),
                        )
                    ],
                ),
            ],
        )
        env = ActivityEnvironment()
        result: SolveOutput = await env.run(solve_device_selection, SolveInput(raw))

        resp = _parse_response(result.response_proto_bytes)
        assert len(resp.pareto_front) == 0

    @pytest.mark.asyncio
    async def test_budget_too_low_returns_no_solutions(self):
        """Budget of 1.0 can't cover the lamp at 500.0."""
        raw = _make_request(
            budget=1.0,
            requirements=[
                pb.DeviceRequirement(
                    requirement_id=1,
                    device_type="smart_lamp",
                    count=1,
                    connect_to_main_ecosystem=True,
                ),
            ],
        )
        env = ActivityEnvironment()
        result: SolveOutput = await env.run(solve_device_selection, SolveInput(raw))

        resp = _parse_response(result.response_proto_bytes)
        assert len(resp.pareto_front) == 0

    @pytest.mark.asyncio
    async def test_unknown_device_type_returns_no_solutions(self):
        """Requesting a device type not in the catalog yields an empty front."""
        raw = _make_request(
            requirements=[
                pb.DeviceRequirement(
                    requirement_id=1,
                    device_type="coffee_maker",
                    count=1,
                    connect_to_main_ecosystem=True,
                ),
            ],
        )
        env = ActivityEnvironment()
        result: SolveOutput = await env.run(solve_device_selection, SolveInput(raw))

        resp = _parse_response(result.response_proto_bytes)
        assert len(resp.pareto_front) == 0

    @pytest.mark.asyncio
    async def test_response_listing_fields_are_populated(self):
        """Spot-check that the proto fields we care about are non-default."""
        raw = _make_request(
            requirements=[
                pb.DeviceRequirement(
                    requirement_id=1,
                    device_type="smart_lamp",
                    count=1,
                    connect_to_main_ecosystem=True,
                ),
            ],
        )
        env = ActivityEnvironment()
        result: SolveOutput = await env.run(solve_device_selection, SolveInput(raw))

        resp = _parse_response(result.response_proto_bytes)
        listing = resp.pareto_front[0].listings[0]

        assert listing.device_id == WIFI_LAMP.device_id
        assert listing.unit_price == pytest.approx(WIFI_LAMP.price)
        assert listing.device_quality == pytest.approx(WIFI_LAMP.quality)
        assert listing.connection_direct.ecosystem == "google"
        assert listing.connection_direct.protocol == "wifi"
        assert listing.requirement_id == 1
