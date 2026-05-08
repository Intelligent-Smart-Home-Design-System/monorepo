from __future__ import annotations

from dataclasses import dataclass
from enum import Enum
from typing import FrozenSet, Optional, Sequence


EcosystemId = int
DeviceId = int
DeviceTypeId = int


class ConnectionMethod(Enum):
    DIRECT = "direct"
    VIA_ECOSYSTEM = "via_ecosystem"


@dataclass(frozen=True, slots=True)
class Device:
    device_id: DeviceId
    type_id: DeviceTypeId
    price: float
    quality: float
    bridge_ecosystem_id: Optional[EcosystemId]
    hub_type_id: Optional[DeviceTypeId]


@dataclass(frozen=True, slots=True)
class TypeCount:
    type_id: DeviceTypeId
    count: int


@dataclass(frozen=True, slots=True)
class DeviceSelectionRequest:
    main_ecosystem_id: EcosystemId
    budget: float
    type_counts: Sequence[TypeCount]
    include_ecosystem_ids: FrozenSet[EcosystemId] = frozenset()
    exclude_ecosystem_ids: FrozenSet[EcosystemId] = frozenset()
    max_solutions: int = 7
    random_seed: Optional[int] = None
    time_budget_seconds: float = 180.0


@dataclass(frozen=True, slots=True)
class ConnectionPlan:
    method: ConnectionMethod
    # if VIA_ECOSYSTEM, through which ecosystem
    bridge_ecosystem_id: Optional[EcosystemId] = None
    # if hub required, which hub is used
    hub_device_id: Optional[DeviceId] = None


@dataclass(frozen=True, slots=True)
class SolutionItem:
    device: Device
    quantity: int
    connection: ConnectionPlan


@dataclass(frozen=True, slots=True)
class ParetoPoint:
    items: Sequence[SolutionItem]
    total_cost: float
    avg_quality: float
    num_ecosystems: int
    num_hubs: int
