from __future__ import annotations
from dataclasses import dataclass

from typing import Any, Optional, Protocol, Sequence

from device_selection.core.model import (
    Device,
    DeviceId,
    DeviceType,
    DeviceRequirement,
    EcosystemId,
    HubType,
    Filter,
    FilterOp,
)


class Catalog(Protocol):
    def devices_for_requirement(self, req: DeviceRequirement) -> Sequence[Device]: ...
    def get_device(self, device_id: DeviceId) -> Optional[Device]: ...
    def available_ecosystems(self) -> frozenset[EcosystemId]: ...
    def available_hub_types_for_ecosystem(self, ecosystem: EcosystemId) -> list[HubType]: ...


@dataclass(slots=True)
class InMemoryCatalog:
    _devices_by_id: dict[DeviceId, Device]
    _devices_by_type: dict[DeviceType, list[Device]]
    _ecosystems: frozenset[EcosystemId]
    # ecosystem -> set of hub types available across hubs in that ecosystem
    _hub_types_by_ecosystem: dict[EcosystemId, list[HubType]]


    def __init__(self, devices: Sequence[Device]) -> None:
        self._devices_by_id = {}
        self._devices_by_type = {}
        self._ecosystems = frozenset()
        self._hub_types_by_ecosystem = {}

        ecosystems: set[EcosystemId] = set()

        for d in devices:
            self._devices_by_id[d.device_id] = d
            self._devices_by_type.setdefault(d.device_type, []).append(d)

            for dc in d.direct_compat:
                ecosystems.add(dc.ecosystem)

            if d.device_type == "smart_hub":
                for dc in d.direct_compat:
                    # collect all protocols this hub supports in this ecosystem
                    # by finding all direct_compat records for the same ecosystem
                    ecosystem_protocols = frozenset(
                        c.protocol for c in d.direct_compat if c.ecosystem == dc.ecosystem and c.protocol != "wifi"
                    )
                    hub_type = HubType(ecosystem=dc.ecosystem, protocols=ecosystem_protocols)
                    existing = self._hub_types_by_ecosystem.setdefault(dc.ecosystem, [])
                    if hub_type not in existing:
                        existing.append(hub_type)

        self._ecosystems = frozenset(ecosystems)

    def devices_for_requirement(self, req: DeviceRequirement) -> Sequence[Device]:
        candidates = self._devices_by_type.get(req.device_type, [])
        if not req.filters:
            return candidates
        return [d for d in candidates if _matches_filters(d, req.filters)]

    def get_device(self, device_id: DeviceId) -> Optional[Device]:
        return self._devices_by_id.get(device_id)

    def available_ecosystems(self) -> frozenset[EcosystemId]:
        return self._ecosystems

    def available_hub_types_for_ecosystem(self, ecosystem: EcosystemId) -> list[HubType]:
        return list(self._hub_types_by_ecosystem.get(ecosystem, []))


def _matches_filters(device: Device, filters: tuple[Filter, ...]) -> bool:
    for f in filters:
        attr = device.attributes.get(f.field)
        if not _apply_filter(attr, f):
            return False
    return True

def _numeric_compatible(a: Any, b: Any) -> bool:
    return isinstance(a, (int, float)) and isinstance(b, (int, float)) and not isinstance(a, bool) and not isinstance(b, bool)

def _apply_filter(attr: Any, f: Filter) -> bool:
    if f.op == FilterOp.EXISTS:
        return attr is not None
    if attr is None:
        return False
    if f.op != FilterOp.CONTAINS:
        if type(attr) is not type(f.value) and not _numeric_compatible(attr, f.value):
            raise TypeError(
                f"filter field '{f.field}': type mismatch, "
                f"attribute is {type(attr).__name__} but filter value is {type(f.value).__name__}"
            )
    if f.op == FilterOp.CONTAINS:
        if not isinstance(attr, list):
            raise TypeError(
                f"filter field '{f.field}': CONTAINS requires a list attribute, "
                f"got {type(attr).__name__}"
            )
    match f.op:
        # type mismatches would raise TypeError
        case FilterOp.EQ:
            return attr == f.value
        case FilterOp.NEQ:
            return attr != f.value
        case FilterOp.GT:
            return attr > f.value
        case FilterOp.GTE:
            return attr >= f.value
        case FilterOp.LT:
            return attr < f.value
        case FilterOp.LTE:
            return attr <= f.value
        case FilterOp.CONTAINS:
            return f.value in attr
        case _:
            raise ValueError(f"unknown filter op: {f.op}")

