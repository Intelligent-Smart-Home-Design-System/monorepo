from __future__ import annotations

import pytest

from device_selection.core.model import (
    BridgeCompat,
    Device,
    DeviceRequirement,
    DeviceSelectionRequest,
    DirectCompat,
    Filter,
    FilterOp,
    ParetoPoint,
    SolutionItem,
)
from device_selection.core.pareto import ParetoArchive
from device_selection.data.catalog import InMemoryCatalog
from device_selection.solvers.enum_repair import SolverConfig, solve_enum_repair


# Ecosystems:
#   main = "google"
#   bridge = "aqara" (cloud bridge to google)
#
# Devices:
#   wifi lamp      - direct google/wifi                          cheap, low quality
#   zigbee lamp    - direct aqara/zigbee, bridge aqara->google   mid price, mid quality
#   matter lamp    - direct google/matter-over-wifi              expensive, high quality
#   wifi sensor    - direct google/wifi                          cheap, low quality
#   zigbee sensor  - direct aqara/zigbee, bridge aqara->google   mid price, mid quality
#   aqara hub      - direct aqara/wifi + aqara/zigbee


def _device(
    device_id: int,
    device_type: str,
    price: float,
    quality: float,
    direct_compat: tuple[DirectCompat, ...] = (),
    bridge_compat: tuple[BridgeCompat, ...] = (),
    attributes: dict | None = None,
) -> Device:
    return Device(
        device_id=device_id,
        device_type=device_type,
        brand="test",
        model=f"model-{device_id}",
        attributes=attributes or {},
        price=price,
        quality=quality,
        source_listing_id=device_id,
        direct_compat=direct_compat,
        bridge_compat=bridge_compat,
    )


WIFI_LAMP = _device(
    1, "smart_lamp", price=500.0, quality=0.5,
    direct_compat=(DirectCompat("google", "wifi"),),
    attributes={"protocol": ["wifi"], "ecosystem": ["google"]},
)

ZIGBEE_LAMP = _device(
    2, "smart_lamp", price=1000.0, quality=0.7,
    direct_compat=(DirectCompat("aqara", "zigbee"),),
    bridge_compat=(BridgeCompat("aqara", "google", "cloud"),),
    attributes={"protocol": ["zigbee"], "ecosystem": ["aqara", "google"]},
)

MATTER_LAMP = _device(
    3, "smart_lamp", price=2000.0, quality=0.9,
    direct_compat=(DirectCompat("google", "matter-over-wifi"),),
    attributes={"protocol": ["matter-over-wifi"], "ecosystem": ["google"]},
)

WIFI_SENSOR = _device(
    4, "motion_sensor", price=600.0, quality=0.5,
    direct_compat=(DirectCompat("google", "wifi"),),
    attributes={"protocol": ["wifi"], "ecosystem": ["google"]},
)

ZIGBEE_SENSOR = _device(
    5, "motion_sensor", price=1100.0, quality=0.75,
    direct_compat=(DirectCompat("aqara", "zigbee"),),
    bridge_compat=(BridgeCompat("aqara", "google", "cloud"),),
    attributes={"protocol": ["zigbee"], "ecosystem": ["aqara", "google"]},
)

AQARA_HUB = _device(
    10, "smart_hub", price=3000.0, quality=0.8,
    direct_compat=(
        DirectCompat("aqara", "wifi"),
        DirectCompat("aqara", "zigbee"),
    ),
    attributes={"protocol": ["wifi", "zigbee"], "ecosystem": ["aqara"]},
)

ALL_DEVICES = [WIFI_LAMP, ZIGBEE_LAMP, MATTER_LAMP, WIFI_SENSOR, ZIGBEE_SENSOR, AQARA_HUB]


@pytest.fixture()
def catalog() -> InMemoryCatalog:
    return InMemoryCatalog(ALL_DEVICES)


def _req(
    requirement_id: int,
    device_type: str,
    count: int = 1,
    connect_to_main: bool = True,
    filters: tuple[Filter, ...] = (),
) -> DeviceRequirement:
    return DeviceRequirement(
        requirement_id=requirement_id,
        device_type=device_type,
        count=count,
        connect_to_main_ecosystem=connect_to_main,
        filters=filters,
    )


def _request(
    requirements: list[DeviceRequirement],
    budget: float = 100_000.0,
    main: str = "google",
    include: frozenset[str] = frozenset(),
    exclude: frozenset[str] = frozenset(),
    max_solutions: int = 10,
    time_budget: float = 30.0,
) -> DeviceSelectionRequest:
    return DeviceSelectionRequest(
        main_ecosystem=main,
        budget=budget,
        requirements=tuple(requirements),
        include_ecosystems=include,
        exclude_ecosystems=exclude,
        max_solutions=max_solutions,
        time_budget_seconds=time_budget,
    )


DEFAULT_CFG = SolverConfig(max_bridge_ecosystems=3, max_hub_types=2)


def _all_points(archive: ParetoArchive) -> list[ParetoPoint]:
    return list(archive.points)


def _item_for_req(point: ParetoPoint, req_id: int) -> list[SolutionItem]:
    return [it for it in point.items if it.requirement_id == req_id]


def _hub_items(point: ParetoPoint) -> list[SolutionItem]:
    return [it for it in point.items if it.requirement_id is None]


def _device_ids_in_point(point: ParetoPoint) -> set[int]:
    return {it.device.device_id for it in point.items}


class TestBasicFeasibility:
    def test_single_wifi_lamp_finds_solution(self, catalog):
        req = _request([_req(1, "smart_lamp")])
        archive = solve_enum_repair(req, catalog, DEFAULT_CFG)
        assert len(_all_points(archive)) > 0

    def test_single_wifi_sensor_finds_solution(self, catalog):
        req = _request([_req(1, "motion_sensor")])
        archive = solve_enum_repair(req, catalog, DEFAULT_CFG)
        assert len(_all_points(archive)) > 0

    def test_lamp_and_sensor_finds_solution(self, catalog):
        req = _request([_req(1, "smart_lamp"), _req(2, "motion_sensor")])
        archive = solve_enum_repair(req, catalog, DEFAULT_CFG)
        assert len(_all_points(archive)) > 0

    def test_unknown_device_type_returns_empty(self, catalog):
        req = _request([_req(1, "smart_toaster")])
        archive = solve_enum_repair(req, catalog, DEFAULT_CFG)
        assert len(_all_points(archive)) == 0

    def test_budget_too_small_returns_empty(self, catalog):
        req = _request([_req(1, "smart_lamp")], budget=100.0)
        archive = solve_enum_repair(req, catalog, DEFAULT_CFG)
        assert len(_all_points(archive)) == 0


class TestRequirementsSatisfied:
    def test_each_requirement_has_one_item(self, catalog):
        req = _request([_req(1, "smart_lamp"), _req(2, "motion_sensor")])
        archive = solve_enum_repair(req, catalog, DEFAULT_CFG)
        for point in _all_points(archive):
            assert len(_item_for_req(point, 1)) == 1
            assert len(_item_for_req(point, 2)) == 1

    def test_item_device_type_matches_requirement(self, catalog):
        req = _request([_req(1, "smart_lamp"), _req(2, "motion_sensor")])
        archive = solve_enum_repair(req, catalog, DEFAULT_CFG)
        for point in _all_points(archive):
            for item in _item_for_req(point, 1):
                assert item.device.device_type == "smart_lamp"
            for item in _item_for_req(point, 2):
                assert item.device.device_type == "motion_sensor"

    def test_quantity_matches_requirement(self, catalog):
        req = _request([_req(1, "smart_lamp", count=2)])
        archive = solve_enum_repair(req, catalog, DEFAULT_CFG)
        for point in _all_points(archive):
            items = _item_for_req(point, 1)
            assert sum(it.quantity for it in items) == 2

    def test_filter_respected(self, catalog):
        # only want rgb lamps - but none in catalog have rgb_support=True
        # so should return empty
        req = _request([
            _req(1, "smart_lamp", filters=(
                Filter("rgb_support", FilterOp.EQ, True),
            )),
        ])
        archive = solve_enum_repair(req, catalog, DEFAULT_CFG)
        assert len(_all_points(archive)) == 0


# ---------------------------------------------------------------------------
# Connection validity
# ---------------------------------------------------------------------------

class TestConnectionValidity:
    def test_wifi_device_has_no_hub(self, catalog):
        # with tight budget only wifi lamp fits, no hub
        req = _request([_req(1, "smart_lamp")], budget=1000.0)
        archive = solve_enum_repair(req, catalog, DEFAULT_CFG)
        for point in _all_points(archive):
            for item in _item_for_req(point, 1):
                if item.device.device_id == WIFI_LAMP.device_id:
                    assert item.connection.connection_direct.hub_solution_item_id is None

    def test_zigbee_device_has_hub_in_solution(self, catalog):
        req = _request([_req(1, "smart_lamp")], budget=10_000.0)
        archive = solve_enum_repair(req, catalog, DEFAULT_CFG)
        # find a point that uses zigbee lamp
        zigbee_points = [
            p for p in _all_points(archive)
            if any(it.device.device_id == ZIGBEE_LAMP.device_id for it in p.items)
        ]
        for point in zigbee_points:
            lamp_item = next(it for it in point.items if it.device.device_id == ZIGBEE_LAMP.device_id)
            # direct connection should reference a hub
            hub_item_id = lamp_item.connection.connection_direct.hub_solution_item_id
            assert hub_item_id is not None
            # that hub item must exist in the solution
            hub_item = next((it for it in point.items if it.id == hub_item_id), None)
            assert hub_item is not None
            assert hub_item.device.device_type == "smart_hub"

    def test_bridge_connection_has_connection_final(self, catalog):
        req = _request([_req(1, "smart_lamp")], budget=10_000.0)
        archive = solve_enum_repair(req, catalog, DEFAULT_CFG)
        bridge_points = [
            p for p in _all_points(archive)
            if any(
                it.device.device_id == ZIGBEE_LAMP.device_id and it.connection.connection_final is not None
                for it in p.items
            )
        ]
        for point in bridge_points:
            lamp_item = next(
                it for it in point.items
                if it.device.device_id == ZIGBEE_LAMP.device_id
            )
            assert lamp_item.connection.connection_final is not None
            assert lamp_item.connection.connection_final.ecosystem == "google"
            assert lamp_item.connection.connection_direct.ecosystem == "aqara"

    def test_all_hub_solution_item_ids_reference_valid_items(self, catalog):
        req = _request([_req(1, "smart_lamp"), _req(2, "motion_sensor")], budget=20_000.0)
        archive = solve_enum_repair(req, catalog, DEFAULT_CFG)
        for point in _all_points(archive):
            item_ids = {it.id for it in point.items}
            for item in point.items:
                direct = item.connection.connection_direct
                if direct.hub_solution_item_id is not None:
                    assert direct.hub_solution_item_id in item_ids
                if item.connection.connection_final is not None:
                    final = item.connection.connection_final
                    if final.hub_solution_item_id is not None:
                        assert final.hub_solution_item_id in item_ids


# ---------------------------------------------------------------------------
# Ecosystem filtering
# ---------------------------------------------------------------------------

class TestEcosystemFiltering:
    def test_excluded_ecosystem_not_used_as_bridge(self, catalog):
        req = _request(
            [_req(1, "smart_lamp")],
            exclude=frozenset({"aqara"}),
            budget=10_000.0,
        )
        archive = solve_enum_repair(req, catalog, DEFAULT_CFG)
        for point in _all_points(archive):
            for item in point.items:
                direct = item.connection.connection_direct
                assert direct.ecosystem != "aqara"
                if item.connection.connection_final:
                    # aqara as bridge source should not appear
                    assert item.connection.connection_direct.ecosystem != "aqara"

    def test_excluded_ecosystem_forces_only_direct_solutions(self, catalog):
        # excluding aqara means only wifi/matter lamps work
        req = _request(
            [_req(1, "smart_lamp")],
            exclude=frozenset({"aqara"}),
            budget=10_000.0,
        )
        archive = solve_enum_repair(req, catalog, DEFAULT_CFG)
        for point in _all_points(archive):
            for item in _item_for_req(point, 1):
                assert item.device.device_id in {WIFI_LAMP.device_id, MATTER_LAMP.device_id}


class TestHubSelection:
    def test_all_wifi_solution_has_no_hub(self, catalog):
        # budget only allows wifi devices
        req = _request(
            [_req(1, "smart_lamp"), _req(2, "motion_sensor")],
            budget=1500.0,
        )
        archive = solve_enum_repair(req, catalog, DEFAULT_CFG)
        for point in _all_points(archive):
            assert len(_hub_items(point)) == 0

    def test_zigbee_solution_includes_hub(self, catalog):
        req = _request(
            [_req(1, "smart_lamp"), _req(2, "motion_sensor")],
            budget=20_000.0,
        )
        archive = solve_enum_repair(req, catalog, DEFAULT_CFG)
        zigbee_points = [
            p for p in _all_points(archive)
            if any(it.device.device_id == ZIGBEE_LAMP.device_id for it in p.items)
        ]
        assert len(zigbee_points) > 0
        for point in zigbee_points:
            assert len(_hub_items(point)) >= 1

    def test_hub_counted_in_num_hubs(self, catalog):
        req = _request([_req(1, "smart_lamp")], budget=20_000.0)
        archive = solve_enum_repair(req, catalog, DEFAULT_CFG)
        for point in _all_points(archive):
            hub_count = len(_hub_items(point))
            assert point.num_hubs == hub_count


class TestParetoFront:
    def test_no_solution_dominated_by_another(self, catalog):
        req = _request([_req(1, "smart_lamp"), _req(2, "motion_sensor")], budget=20_000.0)
        archive = solve_enum_repair(req, catalog, DEFAULT_CFG)
        points = _all_points(archive)
        for i, a in enumerate(points):
            for j, b in enumerate(points):
                if i == j:
                    continue
                dominated = (
                    b.avg_quality >= a.avg_quality
                    and b.num_ecosystems <= a.num_ecosystems
                    and b.num_hubs <= a.num_hubs
                    and (
                        b.avg_quality > a.avg_quality
                        or b.num_ecosystems < a.num_ecosystems
                        or b.num_hubs < a.num_hubs
                    )
                )
                assert not dominated, f"point {i} is dominated by point {j}"

    def test_avg_quality_includes_hubs(self, catalog):
        req = _request([_req(1, "smart_lamp")], budget=20_000.0)
        archive = solve_enum_repair(req, catalog, DEFAULT_CFG)
        for point in _all_points(archive):
            expected = sum(it.device.quality for it in point.items) / len(point.items)
            assert abs(point.avg_quality - expected) < 1e-9

    def test_num_ecosystems_correct(self, catalog):
        req = _request([_req(1, "smart_lamp")], budget=20_000.0)
        archive = solve_enum_repair(req, catalog, DEFAULT_CFG)
        for point in _all_points(archive):
            ecosystems: set[str] = set()
            for item in point.items:
                ecosystems.add(item.connection.connection_direct.ecosystem)
                if item.connection.connection_final:
                    ecosystems.add(item.connection.connection_final.ecosystem)
            assert point.num_ecosystems == len(ecosystems)
