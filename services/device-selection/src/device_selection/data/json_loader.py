"""JSON loader for catalogs and test cases."""
from __future__ import annotations

import json
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from device_selection.core.model import (
    BridgeCompat,
    Device,
    DeviceRequirement,
    DeviceSelectionRequest,
    DirectCompat,
    Filter,
    FilterOp,
)
from device_selection.data.catalog import Catalog, InMemoryCatalog  # adjust import if InMemoryCatalog lives elsewhere


@dataclass
class TestCase:
    id: str
    description: str
    run_brute_force: bool
    catalog: Catalog
    request: DeviceSelectionRequest


def _load_direct_compat(data: dict[str, Any]) -> DirectCompat:
    return DirectCompat(ecosystem=data["ecosystem"], protocol=data["protocol"])


def _load_bridge_compat(data: dict[str, Any]) -> BridgeCompat:
    return BridgeCompat(
        source_ecosystem=data["source_ecosystem"],
        target_ecosystem=data["target_ecosystem"],
        protocol=data["protocol"],
    )


def _load_device(data: dict[str, Any]) -> Device:
    direct_compat = tuple(_load_direct_compat(dc) for dc in data.get("direct_compat", []))
    attrs = data.get("attributes", {})
    attrs["protocol"] = list(set([d.protocol for d in direct_compat]))
    attrs["ecosystem"] = list(set([d.ecosystem for d in direct_compat]))
    attrs["brand"] = data.get("brand", "")
    return Device(
        device_id=data["device_id"],
        device_type=data["device_type"],
        brand=data.get("brand"),
        model=data.get("model"),
        attributes=attrs,
        price=data["price"],
        quality=data["quality"],
        source_listing_id=data["source_listing_id"],
        direct_compat=direct_compat,
        bridge_compat=tuple(_load_bridge_compat(bc) for bc in data.get("bridge_compat", [])),
    )


def load_catalog(data: dict[str, Any]) -> Catalog:
    """Build a Catalog from a parsed JSON dict shaped {"devices": [...]}."""
    devices = [_load_device(d) for d in data["devices"]]
    return InMemoryCatalog(devices)


def _load_filter(data: dict[str, Any]) -> Filter:
    return Filter(
        field=data["field"],
        op=FilterOp(data["op"]),
        value=data.get("value"),
    )


def _load_requirement(data: dict[str, Any]) -> DeviceRequirement:
    return DeviceRequirement(
        requirement_id=data["requirement_id"],
        device_type=data["device_type"],
        count=data["count"],
        connect_to_main_ecosystem=data.get("connect_to_main_ecosystem", True),
        filters=tuple(_load_filter(f) for f in data.get("filters", [])),
    )


def load_request(data: dict[str, Any]) -> DeviceSelectionRequest:
    """Build a DeviceSelectionRequest from a parsed JSON dict."""
    return DeviceSelectionRequest(
        main_ecosystem=data["main_ecosystem"],
        budget=data["budget"],
        requirements=tuple(_load_requirement(r) for r in data["requirements"]),
        include_ecosystems=frozenset(data.get("include_ecosystems", [])),
        exclude_ecosystems=frozenset(data.get("exclude_ecosystems", [])),
        max_solutions=data.get("max_solutions", 7),
        random_seed=data.get("random_seed"),
        time_budget_seconds=data.get("time_budget_seconds", 180.0),
    )


def load_test_case(path: Path) -> TestCase:
    """Load a test case. Catalog filename in the test JSON is resolved relative to the test file."""
    data = json.loads(path.read_text())

    catalog_path = path.parent / data["catalog"]
    catalog_data = json.loads(catalog_path.read_text())
    catalog = load_catalog(catalog_data)

    return TestCase(
        id=data["id"],
        description=data["description"],
        run_brute_force=data.get("run_brute_force", False),
        catalog=catalog,
        request=load_request(data["request"]),
    )
