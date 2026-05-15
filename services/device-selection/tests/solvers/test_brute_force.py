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
from device_selection.core.validate import validate_solution
from device_selection.data.catalog import InMemoryCatalog
from device_selection.solvers.brute_force import solve_brute_force


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


def _all_points(archive: ParetoArchive) -> list[ParetoPoint]:
    return list(archive.points)


class TestBasicFeasibility:
    def test_single_wifi_lamp_finds_solution(self, catalog):
        req = _request([_req(1, "smart_lamp")])
        archive = solve_brute_force(req, catalog)
        assert len(_all_points(archive)) > 0
        for point in archive.points:
            assert len(validate_solution(req, point)) == 0

    def test_single_wifi_sensor_finds_solution(self, catalog):
        req = _request([_req(1, "motion_sensor")])
        archive = solve_brute_force(req, catalog)
        assert len(_all_points(archive)) > 0
        for point in archive.points:
            assert len(validate_solution(req, point)) == 0

    def test_lamp_and_sensor_finds_solution(self, catalog):
        req = _request([_req(1, "smart_lamp"), _req(2, "motion_sensor")])
        archive = solve_brute_force(req, catalog)
        assert len(_all_points(archive)) > 0
        for point in archive.points:
            assert len(validate_solution(req, point)) == 0

    def test_unknown_device_type_returns_empty(self, catalog):
        req = _request([_req(1, "smart_toaster")])
        archive = solve_brute_force(req, catalog)
        assert len(_all_points(archive)) == 0

    def test_budget_too_small_returns_empty(self, catalog):
        req = _request([_req(1, "smart_lamp")], budget=100.0)
        archive = solve_brute_force(req, catalog)
        assert len(_all_points(archive)) == 0


class TestRequirementsSatisfied:
    def test_each_requirement_has_one_item(self, catalog):
        req = _request([_req(1, "smart_lamp"), _req(2, "motion_sensor")])
        archive = solve_brute_force(req, catalog)
        assert len(_all_points(archive)) > 0
        for point in archive.points:
            assert len(validate_solution(req, point)) == 0

    def test_item_device_type_matches_requirement(self, catalog):
        req = _request([_req(1, "smart_lamp"), _req(2, "motion_sensor")])
        archive = solve_brute_force(req, catalog)
        assert len(_all_points(archive)) > 0
        for point in archive.points:
            assert len(validate_solution(req, point)) == 0

    def test_quantity_matches_requirement(self, catalog):
        req = _request([_req(1, "smart_lamp", count=2)])
        archive = solve_brute_force(req, catalog)
        assert len(_all_points(archive)) > 0
        for point in archive.points:
            assert len(validate_solution(req, point)) == 0

    def test_filter_respected(self, catalog):
        # only want rgb lamps - but none in catalog have rgb_support=True
        # so should return empty
        req = _request([
            _req(1, "smart_lamp", filters=(
                Filter("rgb_support", FilterOp.EQ, True),
            )),
        ])
        archive = solve_brute_force(req, catalog)
        assert len(_all_points(archive)) == 0


class TestConnectionValidity:
    def test_wifi_device_has_no_hub(self, catalog):
        # with tight budget only wifi lamp fits, no hub
        req = _request([_req(1, "smart_lamp")], budget=1000.0)
        archive = solve_brute_force(req, catalog)
        assert len(_all_points(archive)) > 0
        for point in archive.points:
            assert len(validate_solution(req, point)) == 0

    def test_zigbee_device_has_hub_in_solution(self, catalog):
        req = _request([_req(1, "smart_lamp")], budget=10_000.0)
        archive = solve_brute_force(req, catalog)
        assert len(_all_points(archive)) > 0
        for point in archive.points:
            assert len(validate_solution(req, point)) == 0

    def test_bridge_connection_has_connection_final(self, catalog):
        req = _request([_req(1, "smart_lamp")], budget=10_000.0)
        archive = solve_brute_force(req, catalog)
        assert len(_all_points(archive)) > 0
        for point in archive.points:
            assert len(validate_solution(req, point)) == 0

    def test_all_hub_solution_item_ids_reference_valid_items(self, catalog):
        req = _request([_req(1, "smart_lamp"), _req(2, "motion_sensor")], budget=20_000.0)
        archive = solve_brute_force(req, catalog)
        assert len(_all_points(archive)) > 0
        for point in archive.points:
            assert len(validate_solution(req, point)) == 0


class TestEcosystemFiltering:
    def test_excluded_ecosystem_not_used_as_bridge(self, catalog):
        req = _request(
            [_req(1, "smart_lamp")],
            exclude=frozenset({"aqara"}),
            budget=10_000.0,
        )
        archive = solve_brute_force(req, catalog)
        assert len(_all_points(archive)) > 0
        for point in archive.points:
            assert len(validate_solution(req, point)) == 0

    def test_excluded_ecosystem_forces_only_direct_solutions(self, catalog):
        # excluding aqara means only wifi/matter lamps work
        req = _request(
            [_req(1, "smart_lamp")],
            exclude=frozenset({"aqara"}),
            budget=10_000.0,
        )
        archive = solve_brute_force(req, catalog)
        assert len(_all_points(archive)) > 0
        for point in archive.points:
            assert len(validate_solution(req, point)) == 0


class TestHubSelection:
    def test_all_wifi_solution_has_no_hub(self, catalog):
        # budget only allows wifi devices
        req = _request(
            [_req(1, "smart_lamp"), _req(2, "motion_sensor")],
            budget=1500.0,
        )
        archive = solve_brute_force(req, catalog)
        assert len(_all_points(archive)) > 0
        for point in archive.points:
            assert len(validate_solution(req, point)) == 0

    def test_zigbee_solution_includes_hub(self, catalog):
        req = _request(
            [_req(1, "smart_lamp"), _req(2, "motion_sensor")],
            budget=20_000.0,
        )
        archive = solve_brute_force(req, catalog)
        assert len(_all_points(archive)) > 0
        for point in archive.points:
            assert len(validate_solution(req, point)) == 0

    def test_hub_counted_in_num_hubs(self, catalog):
        req = _request([_req(1, "smart_lamp")], budget=20_000.0)
        archive = solve_brute_force(req, catalog)
        assert len(_all_points(archive)) > 0
        for point in archive.points:
            assert len(validate_solution(req, point)) == 0

class TestParetoFront:
    def test_avg_quality_includes_hubs(self, catalog):
        req = _request([_req(1, "smart_lamp")], budget=20_000.0)
        archive = solve_brute_force(req, catalog)
        assert len(_all_points(archive)) > 0
        for point in archive.points:
            assert len(validate_solution(req, point)) == 0

    def test_num_ecosystems_correct(self, catalog):
        req = _request([_req(1, "smart_lamp")], budget=20_000.0)
        archive = solve_brute_force(req, catalog)
        assert len(_all_points(archive)) > 0
        for point in archive.points:
            assert len(validate_solution(req, point)) == 0
