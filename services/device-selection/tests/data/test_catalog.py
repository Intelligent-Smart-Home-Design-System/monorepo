from __future__ import annotations

import pytest

from device_selection.core.model import (
    BridgeCompat,
    Device,
    DeviceRequirement,
    DirectCompat,
    HubType,
    Filter,
    FilterOp,
)
from device_selection.data.catalog import InMemoryCatalog, _apply_filter


def make_device(
    device_id: int = 1,
    device_type: str = "smart_lamp",
    attributes: dict | None = None,
    direct_compat: tuple[DirectCompat, ...] = (),
    bridge_compat: tuple[BridgeCompat, ...] = (),
    price: float = 1000.0,
    quality: float = 0.8,
) -> Device:
    return Device(
        device_id=device_id,
        device_type=device_type,
        brand="test",
        model="test-model",
        attributes=attributes or {},
        price=price,
        quality=quality,
        source_listing_id=1,
        direct_compat=direct_compat,
        bridge_compat=bridge_compat,
    )


class TestApplyFilter:
    def test_eq_match(self):
        assert _apply_filter("zigbee", Filter("protocol", FilterOp.EQ, "zigbee")) is True

    def test_eq_no_match(self):
        assert _apply_filter("wifi", Filter("protocol", FilterOp.EQ, "zigbee")) is False

    def test_neq_match(self):
        assert _apply_filter("wifi", Filter("protocol", FilterOp.NEQ, "zigbee")) is True

    def test_eq_type_mismatch_raises(self):
        with pytest.raises(TypeError, match="type mismatch"):
            _apply_filter("5", Filter("wattage", FilterOp.EQ, 5))

    def test_neq_type_mismatch_raises(self):
        with pytest.raises(TypeError, match="type mismatch"):
            _apply_filter(5, Filter("wattage", FilterOp.NEQ, "5"))

    def test_gt(self):
        assert _apply_filter(10.0, Filter("wattage_w", FilterOp.GT, 5)) is True
        assert _apply_filter(3, Filter("wattage_w", FilterOp.GT, 5.0)) is False

    def test_gte(self):
        assert _apply_filter(5.0, Filter("wattage_w", FilterOp.GTE, 5.0)) is True
        assert _apply_filter(4.9, Filter("wattage_w", FilterOp.GTE, 5.0)) is False

    def test_lt(self):
        assert _apply_filter(3.0, Filter("wattage_w", FilterOp.LT, 5.0)) is True
        assert _apply_filter(5.0, Filter("wattage_w", FilterOp.LT, 5.0)) is False

    def test_lte(self):
        assert _apply_filter(5.0, Filter("wattage_w", FilterOp.LTE, 5.0)) is True
        assert _apply_filter(5.1, Filter("wattage_w", FilterOp.LTE, 5.0)) is False

    def test_comparison_type_mismatch_raises(self):
        with pytest.raises(TypeError, match="type mismatch"):
            _apply_filter("big", Filter("wattage_w", FilterOp.GT, 5.0))

    def test_contains_match(self):
        assert _apply_filter(["zigbee", "wifi"], Filter("protocol", FilterOp.CONTAINS, "zigbee")) is True

    def test_contains_no_match(self):
        assert _apply_filter(["wifi"], Filter("protocol", FilterOp.CONTAINS, "zigbee")) is False

    def test_contains_non_list_raises(self):
        with pytest.raises(TypeError, match="CONTAINS requires a list"):
            _apply_filter("zigbee", Filter("protocol", FilterOp.CONTAINS, "zigbee"))

    def test_exists_present(self):
        assert _apply_filter("something", Filter("field", FilterOp.EXISTS, None)) is True

    def test_exists_none(self):
        assert _apply_filter(None, Filter("field", FilterOp.EXISTS, None)) is False

    def test_none_attr_non_exists_returns_false(self):
        assert _apply_filter(None, Filter("field", FilterOp.EQ, "value")) is False

    def test_bool_int_type_mismatch_raises(self):
        with pytest.raises(TypeError, match="type mismatch"):
            _apply_filter(True, Filter("field", FilterOp.EQ, 1))


class TestInMemoryCatalog:
    def _make_catalog(self, devices: list[Device]) -> InMemoryCatalog:
        return InMemoryCatalog(devices)

    def test_get_device_found(self):
        d = make_device(device_id=1)
        catalog = self._make_catalog([d])
        assert catalog.get_device(1) is d

    def test_get_device_not_found(self):
        catalog = self._make_catalog([])
        assert catalog.get_device(99) is None

    def test_devices_for_requirement_no_filters(self):
        d1 = make_device(device_id=1, device_type="smart_lamp")
        d2 = make_device(device_id=2, device_type="smart_lamp")
        d3 = make_device(device_id=3, device_type="motion_sensor")
        catalog = self._make_catalog([d1, d2, d3])
        req = DeviceRequirement(requirement_id=1, device_type="smart_lamp", count=1)
        result = catalog.devices_for_requirement(req)
        assert set(d.device_id for d in result) == {1, 2}

    def test_devices_for_requirement_with_filter(self):
        d1 = make_device(device_id=1, device_type="smart_lamp", attributes={"rgb_support": True})
        d2 = make_device(device_id=2, device_type="smart_lamp", attributes={"rgb_support": False})
        catalog = self._make_catalog([d1, d2])
        req = DeviceRequirement(
            requirement_id=1,
            device_type="smart_lamp",
            count=1,
            filters=(Filter("rgb_support", FilterOp.EQ, True),),
        )
        result = catalog.devices_for_requirement(req)
        assert len(result) == 1
        assert result[0].device_id == 1

    def test_devices_for_requirement_contains_filter(self):
        d1 = make_device(device_id=1, device_type="motion_sensor", attributes={"protocol": ["zigbee", "wifi"]})
        d2 = make_device(device_id=2, device_type="motion_sensor", attributes={"protocol": ["wifi"]})
        catalog = self._make_catalog([d1, d2])
        req = DeviceRequirement(
            requirement_id=1,
            device_type="motion_sensor",
            count=1,
            filters=(Filter("protocol", FilterOp.CONTAINS, "zigbee"),),
        )
        result = catalog.devices_for_requirement(req)
        assert len(result) == 1
        assert result[0].device_id == 1

    def test_devices_for_requirement_unknown_type_returns_empty(self):
        catalog = self._make_catalog([])
        req = DeviceRequirement(requirement_id=1, device_type="unknown_type", count=1)
        assert list(catalog.devices_for_requirement(req)) == []

    def test_available_ecosystems(self):
        d1 = make_device(device_id=1, direct_compat=(DirectCompat("aqara", "zigbee"),))
        d2 = make_device(device_id=2, direct_compat=(DirectCompat("tuya", "wifi"),))
        catalog = self._make_catalog([d1, d2])
        assert catalog.available_ecosystems() == frozenset({"aqara", "tuya"})

    def test_available_ecosystems_empty(self):
        catalog = self._make_catalog([])
        assert catalog.available_ecosystems() == frozenset()


class TestAvailableHubTypesForEcosystem:
    def test_single_hub_single_protocol(self):
        hub = make_device(
            device_id=1,
            device_type="smart_hub",
            direct_compat=(DirectCompat("aqara", "zigbee"),),
        )
        catalog = InMemoryCatalog([hub])
        hub_types = catalog.available_hub_types_for_ecosystem("aqara")
        assert len(hub_types) == 1
        assert hub_types[0] == HubType(ecosystem="aqara", protocols=frozenset({"zigbee"}))

    def test_single_hub_multiple_protocols(self):
        # one hub that supports both zigbee and matter-over-thread in aqara
        hub = make_device(
            device_id=1,
            device_type="smart_hub",
            direct_compat=(
                DirectCompat("aqara", "zigbee"),
                DirectCompat("aqara", "matter-over-thread"),
            ),
        )
        catalog = InMemoryCatalog([hub])
        hub_types = catalog.available_hub_types_for_ecosystem("aqara")
        assert len(hub_types) == 1
        assert hub_types[0] == HubType(
            ecosystem="aqara",
            protocols=frozenset({"zigbee", "matter-over-thread"}),
        )

    def test_two_hubs_different_protocol_sets_gives_two_hub_types(self):
        # hub A: zigbee+matter-over-thread, hub B: zigbee only
        hub_a = make_device(
            device_id=1,
            device_type="smart_hub",
            direct_compat=(
                DirectCompat("aqara", "zigbee"),
                DirectCompat("aqara", "matter-over-thread"),
            ),
        )
        hub_b = make_device(
            device_id=2,
            device_type="smart_hub",
            direct_compat=(DirectCompat("aqara", "zigbee"),),
        )
        catalog = InMemoryCatalog([hub_a, hub_b])
        hub_types = catalog.available_hub_types_for_ecosystem("aqara")
        assert len(hub_types) == 2
        assert HubType(ecosystem="aqara", protocols=frozenset({"zigbee", "matter-over-thread"})) in hub_types
        assert HubType(ecosystem="aqara", protocols=frozenset({"zigbee"})) in hub_types

    def test_two_hubs_same_protocol_set_gives_one_hub_type(self):
        # two different hub devices but same protocol set - same hub type
        hub_a = make_device(
            device_id=1,
            device_type="smart_hub",
            direct_compat=(DirectCompat("aqara", "zigbee"),),
        )
        hub_b = make_device(
            device_id=2,
            device_type="smart_hub",
            direct_compat=(DirectCompat("aqara", "zigbee"),),
        )
        catalog = InMemoryCatalog([hub_a, hub_b])
        hub_types = catalog.available_hub_types_for_ecosystem("aqara")
        assert len(hub_types) == 1

    def test_hub_multiple_ecosystems_indexed_separately(self):
        # hub supports both aqara and google
        hub = make_device(
            device_id=1,
            device_type="smart_hub",
            direct_compat=(
                DirectCompat("aqara", "zigbee"),
                DirectCompat("google", "matter-over-thread"),
            ),
        )
        catalog = InMemoryCatalog([hub])
        aqara_types = catalog.available_hub_types_for_ecosystem("aqara")
        google_types = catalog.available_hub_types_for_ecosystem("google")
        assert len(aqara_types) == 1
        assert aqara_types[0] == HubType(ecosystem="aqara", protocols=frozenset({"zigbee"}))
        assert len(google_types) == 1
        assert google_types[0] == HubType(ecosystem="google", protocols=frozenset({"matter-over-thread"}))

    def test_non_hub_device_not_counted(self):
        device = make_device(
            device_id=1,
            device_type="smart_lamp",
            direct_compat=(DirectCompat("aqara", "zigbee"),),
        )
        catalog = InMemoryCatalog([device])
        assert catalog.available_hub_types_for_ecosystem("aqara") == []

    def test_unknown_ecosystem_returns_empty(self):
        catalog = InMemoryCatalog([])
        assert catalog.available_hub_types_for_ecosystem("nonexistent") == []
