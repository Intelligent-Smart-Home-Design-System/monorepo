import pytest
from device_selection.core.model import (
    BridgeCompat,
    ConnectionInfo,
    ConnectionPlan,
    Device,
    DeviceRequirement,
    DeviceSelectionRequest,
    DirectCompat,
    Filter,
    FilterOp,
    ParetoPoint,
    SolutionItem,
)
from device_selection.core.objectives import compute_objectives
from device_selection.core.validate import ValidationError, validate_solution


# -- builders --

def make_device(
    device_id: int = 1,
    device_type: str = "smart_lamp",
    price: float = 1000.0,
    quality: float = 0.8,
    direct_compat: list[dict] | None = None,
    bridge_compat: list[dict] | None = None,
    attributes: dict | None = None,
) -> Device:
    dc = tuple(
        DirectCompat(ecosystem=d["ecosystem"], protocol=d["protocol"])
        for d in (direct_compat or [{"ecosystem": "yandex", "protocol": "wifi"}])
    )
    bc = tuple(
        BridgeCompat(
            source_ecosystem=b["source_ecosystem"],
            target_ecosystem=b["target_ecosystem"],
            protocol=b["protocol"],
        )
        for b in (bridge_compat or [])
    )
    return Device(
        device_id=device_id,
        device_type=device_type,
        brand="testco",
        model=f"M{device_id}",
        attributes=attributes or {},
        price=price,
        quality=quality,
        source_listing_id=device_id * 100,
        direct_compat=dc,
        bridge_compat=bc,
    )


def make_request(
    requirements: list[DeviceRequirement] | None = None,
    budget: float = 99999.0,
    main_ecosystem: str = "yandex",
    include_ecosystems: frozenset[str] = frozenset(),
    exclude_ecosystems: frozenset[str] = frozenset(),
) -> DeviceSelectionRequest:
    return DeviceSelectionRequest(
        main_ecosystem=main_ecosystem,
        budget=budget,
        requirements=tuple(
            requirements
            or [
                DeviceRequirement(
                    requirement_id=1,
                    device_type="smart_lamp",
                    count=1,
                    connect_to_main_ecosystem=True,
                )
            ]
        ),
        include_ecosystems=include_ecosystems,
        exclude_ecosystems=exclude_ecosystems,
    )


def make_solution_item(
    item_id: int,
    device: Device,
    requirement_id: int | None,
    quantity: int = 1,
    direct: ConnectionInfo | None = None,
    final: ConnectionInfo | None = None,
) -> SolutionItem:
    if direct is None:
        direct = ConnectionInfo(ecosystem="yandex", protocol="wifi")
    return SolutionItem(
        id=item_id,
        device=device,
        requirement_id=requirement_id,
        quantity=quantity,
        connection=ConnectionPlan(
            connection_direct=direct,
            connection_final=final,
        ),
    )


def make_point(
    items: tuple[SolutionItem, ...],
) -> ParetoPoint:
    obj = compute_objectives(items)
    return ParetoPoint(
        items=items,
        total_cost=obj.total_cost,
        avg_quality=obj.avg_quality,
        num_ecosystems=obj.num_ecosystems,
        num_hubs=obj.num_hubs,
    )


def error_codes(errors: list[ValidationError]) -> set[str]:
    return {e.code for e in errors}


# -- happy path --

def test_valid_direct_wifi_solution():
    device = make_device(device_id=1, device_type="smart_lamp", price=1000.0)
    item = make_solution_item(1, device, requirement_id=1)
    req = make_request(
        requirements=[DeviceRequirement(1, "smart_lamp", 1, True)],
        budget=2000.0,
    )
    point = make_point((item,))
    assert validate_solution(req, point) == []


def test_valid_two_requirements():
    lamp = make_device(1, "smart_lamp", price=1000.0)
    sensor = make_device(2, "motion_sensor", price=800.0)
    items = (
        make_solution_item(1, lamp, requirement_id=1),
        make_solution_item(2, sensor, requirement_id=2),
    )
    req = make_request(
        requirements=[
            DeviceRequirement(1, "smart_lamp", 1, True),
            DeviceRequirement(2, "motion_sensor", 1, True),
        ],
        budget=5000.0,
    )
    point = make_point(items)
    assert validate_solution(req, point) == []


def test_valid_quantity_two():
    device = make_device(1, "smart_lamp", price=1000.0)
    item = make_solution_item(1, device, requirement_id=1, quantity=2)
    req = make_request(
        requirements=[DeviceRequirement(1, "smart_lamp", 2, True)],
        budget=5000.0,
    )
    point = make_point((item,))
    assert validate_solution(req, point) == []


# -- requirement coverage --

def test_wrong_device_type():
    device = make_device(1, "motion_sensor")
    item = make_solution_item(1, device, requirement_id=1)
    req = make_request(
        requirements=[DeviceRequirement(1, "smart_lamp", 1, True)]
    )
    point = make_point((item,))
    assert "REQ_TYPE_MISMATCH" in error_codes(validate_solution(req, point))


def test_wrong_quantity():
    device = make_device(1, "smart_lamp", price=1000.0)
    item = make_solution_item(1, device, requirement_id=1, quantity=1)
    req = make_request(
        requirements=[DeviceRequirement(1, "smart_lamp", 2, True)]
    )
    point = make_point((item,))
    assert "REQ_COUNT_MISMATCH" in error_codes(validate_solution(req, point))


def test_unknown_requirement_id():
    device = make_device(1, "smart_lamp")
    item = make_solution_item(1, device, requirement_id=99)
    req = make_request(
        requirements=[DeviceRequirement(1, "smart_lamp", 1, True)]
    )
    point = make_point((item,))
    codes = error_codes(validate_solution(req, point))
    assert "UNKNOWN_REQUIREMENT_ID" in codes


def test_filter_match_passes():
    device = make_device(1, "smart_lamp", attributes={"socket_type": "E27"})
    item = make_solution_item(1, device, requirement_id=1)
    req = make_request(
        requirements=[
            DeviceRequirement(
                1,
                "smart_lamp",
                1,
                True,
                filters=(Filter("socket_type", FilterOp.EQ, "E27"),),
            )
        ]
    )
    point = make_point((item,))
    assert validate_solution(req, point) == []


def test_filter_mismatch():
    device = make_device(1, "smart_lamp", attributes={"socket_type": "E14"})
    item = make_solution_item(1, device, requirement_id=1)
    req = make_request(
        requirements=[
            DeviceRequirement(
                1,
                "smart_lamp",
                1,
                True,
                filters=(Filter("socket_type", FilterOp.EQ, "E27"),),
            )
        ]
    )
    point = make_point((item,))
    assert "REQ_FILTER_MISMATCH" in error_codes(validate_solution(req, point))


# -- budget --

def test_budget_exceeded():
    device = make_device(1, "smart_lamp", price=5000.0)
    item = make_solution_item(1, device, requirement_id=1)
    req = make_request(
        requirements=[DeviceRequirement(1, "smart_lamp", 1, True)],
        budget=4999.0,
    )
    obj = compute_objectives((item,))
    # construct point with real cost so the budget check fires, not objective mismatch
    point = ParetoPoint(
        items=(item,),
        total_cost=obj.total_cost,
        avg_quality=obj.avg_quality,
        num_ecosystems=obj.num_ecosystems,
        num_hubs=obj.num_hubs,
    )
    assert "BUDGET_EXCEEDED" in error_codes(validate_solution(req, point))


def test_budget_exact_boundary_passes():
    device = make_device(1, "smart_lamp", price=5000.0)
    item = make_solution_item(1, device, requirement_id=1)
    req = make_request(
        requirements=[DeviceRequirement(1, "smart_lamp", 1, True)],
        budget=5000.0,
    )
    point = make_point((item,))
    assert not any(e.code == "BUDGET_EXCEEDED" for e in validate_solution(req, point))


# -- recorded objectives --

def test_cost_mismatch_detected():
    device = make_device(1, "smart_lamp", price=1000.0)
    item = make_solution_item(1, device, requirement_id=1)
    req = make_request(requirements=[DeviceRequirement(1, "smart_lamp", 1, True)])
    obj = compute_objectives((item,))
    point = ParetoPoint(
        items=(item,),
        total_cost=obj.total_cost + 500.0,  # deliberate mismatch
        avg_quality=obj.avg_quality,
        num_ecosystems=obj.num_ecosystems,
        num_hubs=obj.num_hubs,
    )
    assert "TOTAL_COST_MISMATCH" in error_codes(validate_solution(req, point))


def test_quality_mismatch_detected():
    device = make_device(1, "smart_lamp", quality=0.8)
    item = make_solution_item(1, device, requirement_id=1)
    req = make_request(requirements=[DeviceRequirement(1, "smart_lamp", 1, True)])
    obj = compute_objectives((item,))
    point = ParetoPoint(
        items=(item,),
        total_cost=obj.total_cost,
        avg_quality=0.5,  # deliberate mismatch
        num_ecosystems=obj.num_ecosystems,
        num_hubs=obj.num_hubs,
    )
    assert "AVG_QUALITY_MISMATCH" in error_codes(validate_solution(req, point))


# -- hub self-connectivity --

def test_hub_valid_wifi():
    hub = make_device(
        10,
        "smart_hub",
        price=2000.0,
        direct_compat=[{"ecosystem": "yandex", "protocol": "wifi"}, {"ecosystem": "yandex", "protocol": "zigbee"}],
    )
    lamp = make_device(1, "smart_lamp", direct_compat=[{"ecosystem": "yandex", "protocol": "zigbee"}])
    hub_item = make_solution_item(
        10,
        hub,
        requirement_id=None,
        direct=ConnectionInfo("yandex", "wifi"),
    )
    lamp_item = make_solution_item(
        1,
        lamp,
        requirement_id=1,
        direct=ConnectionInfo("yandex", "zigbee", 10),
    )
    req = make_request(requirements=[DeviceRequirement(1, "smart_lamp", 1, True)])
    point = make_point((hub_item, lamp_item))
    assert validate_solution(req, point) == []


def test_hub_connecting_via_hub_required_protocol_is_invalid():
    hub = make_device(
        10,
        "smart_hub",
        price=2000.0,
        direct_compat=[{"ecosystem": "yandex", "protocol": "zigbee"}],
    )
    lamp = make_device(1, "smart_lamp")
    hub_item = make_solution_item(
        10,
        hub,
        requirement_id=None,
        direct=ConnectionInfo("yandex", "zigbee"),
    )
    lamp_item = make_solution_item(1, lamp, requirement_id=1)
    req = make_request(requirements=[DeviceRequirement(1, "smart_lamp", 1, True)])
    point = make_point((hub_item, lamp_item))
    assert "HUB_REQUIRES_HUB" in error_codes(validate_solution(req, point))


def test_hub_no_direct_compat_entry():
    hub = make_device(
        10,
        "smart_hub",
        price=2000.0,
        direct_compat=[{"ecosystem": "aqara", "protocol": "wifi"}],
    )
    lamp = make_device(1, "smart_lamp")
    hub_item = make_solution_item(
        10,
        hub,
        requirement_id=None,
        direct=ConnectionInfo("yandex", "wifi"),  # not in direct_compat
    )
    lamp_item = make_solution_item(1, lamp, requirement_id=1)
    req = make_request(requirements=[DeviceRequirement(1, "smart_lamp", 1, True)])
    point = make_point((hub_item, lamp_item))
    assert "HUB_NO_DIRECT_COMPAT" in error_codes(validate_solution(req, point))


# -- direct device connections --

def test_no_direct_compat_for_device():
    device = make_device(
        1,
        "smart_lamp",
        direct_compat=[{"ecosystem": "aqara", "protocol": "wifi"}],
    )
    item = make_solution_item(
        1,
        device,
        requirement_id=1,
        direct=ConnectionInfo("yandex", "wifi"),  # not in device's compat
    )
    req = make_request(requirements=[DeviceRequirement(1, "smart_lamp", 1, True)])
    point = make_point((item,))
    assert "NO_DIRECT_COMPAT" in error_codes(validate_solution(req, point))


def test_hub_required_protocol_missing_hub_reference():
    device = make_device(
        1,
        "smart_lamp",
        direct_compat=[{"ecosystem": "yandex", "protocol": "zigbee"}],
    )
    item = make_solution_item(
        1,
        device,
        requirement_id=1,
        direct=ConnectionInfo("yandex", "zigbee", hub_solution_item_id=None),
    )
    req = make_request(requirements=[DeviceRequirement(1, "smart_lamp", 1, True)])
    point = make_point((item,))
    assert "MISSING_HUB_REFERENCE" in error_codes(validate_solution(req, point))


def test_hub_reference_points_to_nonexistent_item():
    device = make_device(
        1,
        "smart_lamp",
        direct_compat=[{"ecosystem": "yandex", "protocol": "zigbee"}],
    )
    item = make_solution_item(
        1,
        device,
        requirement_id=1,
        direct=ConnectionInfo("yandex", "zigbee", hub_solution_item_id=999),
    )
    req = make_request(requirements=[DeviceRequirement(1, "smart_lamp", 1, True)])
    point = make_point((item,))
    assert "HUB_ITEM_NOT_FOUND" in error_codes(validate_solution(req, point))


def test_hub_reference_points_to_non_hub_item():
    lamp1 = make_device(1, "smart_lamp", direct_compat=[{"ecosystem": "yandex", "protocol": "zigbee"}])
    lamp2 = make_device(2, "smart_lamp")
    item1 = make_solution_item(
        1,
        lamp1,
        requirement_id=1,
        direct=ConnectionInfo("yandex", "zigbee", hub_solution_item_id=2),
    )
    item2 = make_solution_item(2, lamp2, requirement_id=2)
    req = make_request(
        requirements=[
            DeviceRequirement(1, "smart_lamp", 1, True),
            DeviceRequirement(2, "smart_lamp", 1, True),
        ]
    )
    point = make_point((item1, item2))
    assert "HUB_ITEM_WRONG_TYPE" in error_codes(validate_solution(req, point))


def test_hub_missing_protocol_for_device():
    hub = make_device(
        10,
        "smart_hub",
        price=2000.0,
        direct_compat=[
            {"ecosystem": "yandex", "protocol": "wifi"},
            {"ecosystem": "yandex", "protocol": "matter"},
        ],
    )
    device = make_device(
        1,
        "smart_lamp",
        direct_compat=[{"ecosystem": "yandex", "protocol": "zigbee"}],
    )
    hub_item = make_solution_item(
        10, hub, requirement_id=None,
        direct=ConnectionInfo("yandex", "wifi"),
    )
    lamp_item = make_solution_item(
        1,
        device,
        requirement_id=1,
        direct=ConnectionInfo("yandex", "zigbee", hub_solution_item_id=10),
    )
    req = make_request(requirements=[DeviceRequirement(1, "smart_lamp", 1, True)])
    point = make_point((hub_item, lamp_item))
    # hub supports matter but not zigbee, so device can't use it
    assert "HUB_MISSING_PROTOCOL" in error_codes(validate_solution(req, point))


def test_valid_zigbee_with_hub():
    hub = make_device(
        10,
        "smart_hub",
        price=2000.0,
        direct_compat=[
            {"ecosystem": "yandex", "protocol": "wifi"},
            {"ecosystem": "yandex", "protocol": "zigbee"},
        ],
    )
    device = make_device(
        1,
        "smart_lamp",
        direct_compat=[{"ecosystem": "yandex", "protocol": "zigbee"}],
    )
    hub_item = make_solution_item(
        10, hub, requirement_id=None,
        direct=ConnectionInfo("yandex", "wifi"),
    )
    lamp_item = make_solution_item(
        1,
        device,
        requirement_id=1,
        direct=ConnectionInfo("yandex", "zigbee", hub_solution_item_id=10),
    )
    req = make_request(requirements=[DeviceRequirement(1, "smart_lamp", 1, True)])
    point = make_point((hub_item, lamp_item))
    assert validate_solution(req, point) == []


# -- bridge connections --

def test_valid_bridge_connection():
    hub = make_device(
        10,
        "smart_hub",
        price=2000.0,
        direct_compat=[
            {"ecosystem": "yandex", "protocol": "wifi"},
            {"ecosystem": "aqara", "protocol": "zigbee"},
        ],
    )
    device = make_device(
        1,
        "smart_lamp",
        direct_compat=[{"ecosystem": "aqara", "protocol": "zigbee"}],
        bridge_compat=[
            {
                "source_ecosystem": "aqara",
                "target_ecosystem": "yandex",
                "protocol": "wifi",
            }
        ],
    )
    hub_item = make_solution_item(
        10, hub, requirement_id=None,
        direct=ConnectionInfo("yandex", "wifi"),
    )
    lamp_item = make_solution_item(
        1,
        device,
        requirement_id=1,
        direct=ConnectionInfo("aqara", "zigbee", hub_solution_item_id=10),
        final=ConnectionInfo("yandex", "wifi"),
    )
    req = make_request(requirements=[DeviceRequirement(1, "smart_lamp", 1, True)])
    point = make_point((hub_item, lamp_item))
    assert validate_solution(req, point) == []


def test_bridge_no_bridge_compat_record():
    device = make_device(
        1,
        "smart_lamp",
        direct_compat=[{"ecosystem": "aqara", "protocol": "wifi"}],
        bridge_compat=[],  # no bridge_compat
    )
    item = make_solution_item(
        1,
        device,
        requirement_id=1,
        direct=ConnectionInfo("aqara", "wifi"),
        final=ConnectionInfo("yandex", "wifi"),
    )
    req = make_request(requirements=[DeviceRequirement(1, "smart_lamp", 1, True)])
    point = make_point((item,))
    assert "NO_BRIDGE_COMPAT" in error_codes(validate_solution(req, point))


def test_bridge_wrong_termination():
    device = make_device(
        1,
        "smart_lamp",
        direct_compat=[{"ecosystem": "aqara", "protocol": "wifi"}],
        bridge_compat=[
            {
                "source_ecosystem": "aqara",
                "target_ecosystem": "tuya",  # goes to tuya, not yandex
                "protocol": "wifi",
            }
        ],
    )
    item = make_solution_item(
        1,
        device,
        requirement_id=1,
        direct=ConnectionInfo("aqara", "wifi"),
        final=ConnectionInfo("tuya", "wifi"),
    )
    req = make_request(requirements=[DeviceRequirement(1, "smart_lamp", 1, True)])
    point = make_point((item,))
    assert "WRONG_TERMINATION_ECOSYSTEM" in error_codes(validate_solution(req, point))


def test_bridge_source_hub_missing_protocol():
    hub = make_device(
        10,
        "smart_hub",
        price=2000.0,
        direct_compat=[
            {"ecosystem": "yandex", "protocol": "wifi"},
            # no zigbee for aqara
        ],
    )
    device = make_device(
        1,
        "smart_lamp",
        direct_compat=[{"ecosystem": "aqara", "protocol": "zigbee"}],
        bridge_compat=[
            {
                "source_ecosystem": "aqara",
                "target_ecosystem": "yandex",
                "protocol": "wifi",
            }
        ],
    )
    hub_item = make_solution_item(
        10, hub, requirement_id=None,
        direct=ConnectionInfo("yandex", "wifi"),
    )
    lamp_item = make_solution_item(
        1,
        device,
        requirement_id=1,
        direct=ConnectionInfo("aqara", "zigbee", hub_solution_item_id=10),
        final=ConnectionInfo("yandex", "wifi"),
    )
    req = make_request(requirements=[DeviceRequirement(1, "smart_lamp", 1, True)])
    point = make_point((hub_item, lamp_item))
    assert "HUB_MISSING_PROTOCOL" in error_codes(validate_solution(req, point))


def test_bridge_cloud_needs_no_target_hub():
    device = make_device(
        1,
        "smart_lamp",
        direct_compat=[{"ecosystem": "aqara", "protocol": "wifi"}],
        bridge_compat=[
            {
                "source_ecosystem": "aqara",
                "target_ecosystem": "yandex",
                "protocol": "cloud",
            }
        ],
    )
    item = make_solution_item(
        1,
        device,
        requirement_id=1,
        direct=ConnectionInfo("aqara", "wifi"),
        final=ConnectionInfo("yandex", "cloud"),
    )
    req = make_request(requirements=[DeviceRequirement(1, "smart_lamp", 1, True)])
    point = make_point((item,))
    assert validate_solution(req, point) == []


# -- dead hubs --

def test_dead_hub_detected():
    hub = make_device(
        10,
        "smart_hub",
        price=2000.0,
        direct_compat=[{"ecosystem": "yandex", "protocol": "wifi"}],
    )
    lamp = make_device(1, "smart_lamp")
    hub_item = make_solution_item(
        10, hub, requirement_id=None,
        direct=ConnectionInfo("yandex", "wifi"),
    )
    # lamp connects via wifi directly, never references the hub
    lamp_item = make_solution_item(
        1, lamp, requirement_id=1,
        direct=ConnectionInfo("yandex", "wifi"),
    )
    req = make_request(requirements=[DeviceRequirement(1, "smart_lamp", 1, True)])
    point = make_point((hub_item, lamp_item))
    assert "DEAD_HUB" in error_codes(validate_solution(req, point))


def test_hub_referenced_by_bridge_source_not_dead():
    hub = make_device(
        10,
        "smart_hub",
        price=2000.0,
        direct_compat=[
            {"ecosystem": "yandex", "protocol": "wifi"},
            {"ecosystem": "aqara", "protocol": "zigbee"},
        ],
    )
    device = make_device(
        1,
        "smart_lamp",
        direct_compat=[{"ecosystem": "aqara", "protocol": "zigbee"}],
        bridge_compat=[
            {
                "source_ecosystem": "aqara",
                "target_ecosystem": "yandex",
                "protocol": "wifi",
            }
        ],
    )
    hub_item = make_solution_item(
        10, hub, requirement_id=None,
        direct=ConnectionInfo("yandex", "wifi"),
    )
    lamp_item = make_solution_item(
        1,
        device,
        requirement_id=1,
        direct=ConnectionInfo("aqara", "zigbee", hub_solution_item_id=10),
        final=ConnectionInfo("yandex", "wifi"),
    )
    req = make_request(requirements=[DeviceRequirement(1, "smart_lamp", 1, True)])
    point = make_point((hub_item, lamp_item))
    assert not any(e.code == "DEAD_HUB" for e in validate_solution(req, point))


# -- ecosystem filters --

def test_excluded_ecosystem_rejected():
    device = make_device(
        1,
        "smart_lamp",
        direct_compat=[{"ecosystem": "aqara", "protocol": "wifi"}],
        bridge_compat=[
            {
                "source_ecosystem": "aqara",
                "target_ecosystem": "yandex",
                "protocol": "cloud",
            }
        ],
    )
    item = make_solution_item(
        1,
        device,
        requirement_id=1,
        direct=ConnectionInfo("aqara", "wifi"),
        final=ConnectionInfo("yandex", "cloud"),
    )
    req = make_request(
        requirements=[DeviceRequirement(1, "smart_lamp", 1, True)],
        exclude_ecosystems=frozenset({"aqara"}),
    )
    point = make_point((item,))
    assert "ECO_EXCLUDED" in error_codes(validate_solution(req, point))


def test_include_ecosystems_allows_main():
    device = make_device(1, "smart_lamp")
    item = make_solution_item(1, device, requirement_id=1)
    req = make_request(
        requirements=[DeviceRequirement(1, "smart_lamp", 1, True)],
        include_ecosystems=frozenset({"tuya"}),  # yandex still allowed as main
    )
    point = make_point((item,))
    assert not any(e.code == "ECO_NOT_INCLUDED" for e in validate_solution(req, point))


def test_ecosystem_not_in_include_list_rejected():
    device = make_device(
        1,
        "smart_lamp",
        direct_compat=[{"ecosystem": "aqara", "protocol": "wifi"}],
        bridge_compat=[
            {
                "source_ecosystem": "aqara",
                "target_ecosystem": "yandex",
                "protocol": "cloud",
            }
        ],
    )
    item = make_solution_item(
        1,
        device,
        requirement_id=1,
        direct=ConnectionInfo("aqara", "wifi"),
        final=ConnectionInfo("yandex", "cloud"),
    )
    req = make_request(
        requirements=[DeviceRequirement(1, "smart_lamp", 1, True)],
        include_ecosystems=frozenset({"tuya"}),  # aqara not included
    )
    point = make_point((item,))
    assert "ECO_NOT_INCLUDED" in error_codes(validate_solution(req, point))
