from __future__ import annotations

from dataclasses import dataclass
from enum import Enum
from typing import Any, FrozenSet, Optional


EcosystemId = str
DeviceId = int
DeviceType = str
ProtocolId = str


@dataclass(frozen=True, slots=True)
class DirectCompat:
    ecosystem: EcosystemId
    protocol: ProtocolId


@dataclass(frozen=True, slots=True)
class BridgeCompat:
    source_ecosystem: EcosystemId
    target_ecosystem: EcosystemId
    protocol: ProtocolId


@dataclass(frozen=True, slots=True)
class HubType:
    ecosystem: EcosystemId
    protocols: FrozenSet[ProtocolId]


@dataclass(frozen=True, slots=True)
class Device:
    device_id: DeviceId
    device_type: DeviceType
    brand: Optional[str]
    model: Optional[str]
    attributes: dict[str, Any]

    price: float
    quality: float
    source_listing_id: int

    direct_compat: tuple[DirectCompat, ...]
    bridge_compat: tuple[BridgeCompat, ...]


class FilterOp(Enum):
    EQ = "eq"
    NEQ = "neq"
    GT = "gt"
    GTE = "gte"
    LT = "lt"
    LTE = "lte"
    CONTAINS = "contains"
    EXISTS = "exists"


@dataclass(frozen=True, slots=True)
class Filter:
    field: str
    op: FilterOp
    value: str | int | float | None


@dataclass(frozen=True, slots=True)
class DeviceRequirement:
    requirement_id: int
    device_type: DeviceType
    count: int
    connect_to_main_ecosystem: bool = True
    filters: tuple[Filter, ...] = ()


@dataclass(frozen=True, slots=True)
class DeviceSelectionRequest:
    main_ecosystem: EcosystemId
    budget: float
    requirements: tuple[DeviceRequirement, ...]
    include_ecosystems: FrozenSet[EcosystemId] = frozenset()
    exclude_ecosystems: FrozenSet[EcosystemId] = frozenset()
    max_solutions: int = 7
    random_seed: Optional[int] = None
    time_budget_seconds: float = 180.0


@dataclass(frozen=True, slots=True)
class ConnectionInfo:
    ecosystem: EcosystemId
    protocol: ProtocolId
    hub_solution_item_id: Optional[int] = None


@dataclass(frozen=True, slots=True)
class ConnectionPlan:
    connection_direct: ConnectionInfo
    connection_final: Optional[ConnectionInfo] = None


@dataclass(frozen=True, slots=True)
class SolutionItem:
    id: int
    device: Device
    requirement_id: Optional[int]
    quantity: int
    connection: ConnectionPlan


@dataclass(frozen=True, slots=True)
class ParetoPoint:
    items: tuple[SolutionItem, ...]
    total_cost: float
    avg_quality: float
    num_ecosystems: int
    num_hubs: int
