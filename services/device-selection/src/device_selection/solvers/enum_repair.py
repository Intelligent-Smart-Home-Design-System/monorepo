from __future__ import annotations

from collections import defaultdict
from dataclasses import dataclass
from itertools import combinations
from time import perf_counter
from typing import Iterable, Optional, TypeVar

from device_selection.core.model import (
    ConnectionMethod,
    ConnectionPlan,
    Device,
    DeviceId,
    DeviceSelectionRequest,
    EcosystemId,
    ParetoPoint,
    SolutionItem,
    DeviceTypeId,
)
from device_selection.data.catalog import Catalog
from device_selection.core.pareto import ParetoArchive

MAX_BRIDGE_ECOSYSTEMS = 5
MAX_HUBS = 4
MAX_CANDIDATES_PER_TYPE: Optional[int] = None  # None to disable


T = TypeVar('T')
def _iter_subsets(items: list[T], max_size: int) -> Iterable[frozenset[T]]:
    n = len(items)
    for r in range(0, min(max_size, n) + 1):
        for comb in combinations(items, r):
            yield frozenset(comb)


def _filter_candidates(
    devices_sorted: list[Device],
    bridge_set: frozenset[int],
    hub_set: frozenset[int],
) -> list[Device]:
    """
    Filter devices that:
      - are connectable under bridge_set
      - if require a hub (hub_type_id != None), that hub_type_id must be in hub_set
    Input devices_sorted must be sorted by price ascending.
    """
    res: list[Device] = []
    last_quality = -1
    for d in devices_sorted:
        if d.bridge_ecosystem_id is not None and d.bridge_ecosystem_id not in bridge_set:
            continue
        if d.hub_type_id is not None and d.hub_type_id not in hub_set:
            continue
        if d.quality <= last_quality:
            continue

        last_quality = d.quality
        res.append(d)
        if MAX_CANDIDATES_PER_TYPE is not None and len(res) >= MAX_CANDIDATES_PER_TYPE:
            break

    return res


def _repair_to_budget(
    budget: float,
    candidates_by_type: list[list[Device]],
    chosen_idx: list[int],
    quantities: list[int],
) -> Optional[list[Device]]:
    """
    Generic repair: we have N categories (requested types + hub types).
    Start from most expensive in each category, downgrade the most expensive category until <= budget.
    """
    n = len(candidates_by_type)

    while True:
        chosen = [candidates_by_type[i][chosen_idx[i]] for i in range(n)]
        total = sum([quantities[i] * chosen[i].price for i in range(n)])
        if total <= budget:
            return chosen

        # downgrade the category contributing most to cost that still has cheaper options
        best_i = -1
        best_cost = -1.0
        for i in range(n):
            if chosen_idx[i] <= 0:
                continue
            d = candidates_by_type[i][chosen_idx[i]]
            cost = d.price * quantities[i]
            if cost > best_cost:
                best_cost = cost
                best_i = i

        if best_i == -1:
            return None  # cannot repair further

        chosen_idx[best_i] -= 1


def _build_solution(
    chosen_devices: list[Device],
    quantities: list[int],
) -> ParetoPoint:
    items: list[SolutionItem] = []

    used_bridge_ecosystems: set[EcosystemId] = set()
    used_hubs: set[DeviceTypeId] = set()
    device_by_type: dict[DeviceTypeId, DeviceId] = {}

    for d in chosen_devices:
        device_by_type[d.type_id] = d.device_id

    # Add all chosen devices as solution items
    total_cost = 0
    for i in range(len(chosen_devices)):
        d = chosen_devices[i]
        qty = quantities[i]
        method: ConnectionMethod = ConnectionMethod.DIRECT
        bridge: EcosystemId = None
        if d.bridge_ecosystem_id is not None:
            method = ConnectionMethod.VIA_ECOSYSTEM
            bridge = d.bridge_ecosystem_id
            used_bridge_ecosystems.add(bridge)

        hub_device_id = None
        if d.hub_type_id is not None:
            hub_device_id = device_by_type.get(d.hub_type_id)
            used_hubs.add(hub_device_id)

        items.append(
            SolutionItem(
                device=d,
                quantity=qty,
                connection=ConnectionPlan(
                    method=method,
                    bridge_ecosystem_id=bridge,
                    hub_device_id=hub_device_id,
                ),
            )
        )
        total_cost += qty * d.price

    avg_quality = sum(d.quality for d in chosen_devices) / len(chosen_devices)

    num_ecosystems = 1 + len(used_bridge_ecosystems)

    num_hubs = len(used_hubs)

    return ParetoPoint(
        items=tuple(items),
        total_cost=total_cost,
        avg_quality=avg_quality,
        num_ecosystems=num_ecosystems,
        num_hubs=num_hubs,
    )


def solve_enum_repair(req: DeviceSelectionRequest, catalog: Catalog) -> ParetoArchive:
    start = perf_counter()

    requested_type_ids: list[DeviceTypeId] = [tc.type_id for tc in req.type_counts]
    requested_counts: list[int] = [tc.count for tc in req.type_counts]

    devices_by_type: list[list[Device]] = []
    for t in requested_type_ids:
        devices = list(catalog.devices_for_type(t))
        devices.sort(key=lambda d: d.price)
        devices_by_type.append(devices)

    bridge_ecosystems: set[EcosystemId] = set()
    ecosystem_to_hub_types: defaultdict[EcosystemId, set[DeviceTypeId]] = defaultdict(set)

    for devices in devices_by_type:
        for d in devices:
            bridge = d.bridge_ecosystem_id
            if bridge is not None:
                if (not req.include_ecosystem_ids or bridge in req.include_ecosystem_ids) and bridge not in req.exclude_ecosystem_ids:
                    bridge_ecosystems.add(bridge)    
            id = bridge if bridge is not None else req.main_ecosystem_id
            if d.hub_type_id is not None:
                ecosystem_to_hub_types[id].add(d.hub_type_id)

    archive = ParetoArchive()

    for bridge_set in _iter_subsets(sorted(bridge_ecosystems), MAX_BRIDGE_ECOSYSTEMS):
        if perf_counter() - start >= req.time_budget_seconds:
            break

        hub_types = set[DeviceTypeId]()
        if req.main_ecosystem_id in ecosystem_to_hub_types:
            hub_types |= ecosystem_to_hub_types[req.main_ecosystem_id]
        for bridge in bridge_set:
            if bridge in ecosystem_to_hub_types:
                hub_types |= ecosystem_to_hub_types[bridge]

        for hub_set in _iter_subsets(sorted(hub_types), MAX_HUBS):
            if perf_counter() - start >= req.time_budget_seconds:
                break

            candidates_by_type: list[list[Device]] = []
            feasible = True

            for devices_sorted in devices_by_type:
                candidates = _filter_candidates(
                    devices_sorted=devices_sorted,
                    bridge_set=bridge_set,
                    hub_set=hub_set,
                )
                if not candidates:
                    feasible = False
                    break
                candidates_by_type.append(candidates)

            hub_type_ids = sorted(hub_set)
            for hub_type_id in hub_type_ids:
                hubs = catalog.devices_for_type(hub_type_id)
                if not hubs:
                    feasible = False
                    break
                candidates_by_type.append(hubs)

            if not feasible:
                continue

            quantities = requested_counts + [1] * len(hub_type_ids)

            chosen_idx = [len(candidates) - 1 for candidates in candidates_by_type]

            repaired = _repair_to_budget(
                budget=req.budget,
                candidates_by_type=candidates_by_type,
                chosen_idx=chosen_idx,
                quantities=quantities,
            )
            if repaired is None:
                continue

            point = _build_solution(
                chosen_devices=repaired,
                quantities=quantities,
            )

            if point.total_cost <= req.budget + 1e-9:
                archive.add(point)

    return archive
