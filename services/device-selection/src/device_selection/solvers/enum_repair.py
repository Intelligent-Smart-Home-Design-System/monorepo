from __future__ import annotations

from dataclasses import dataclass
from itertools import combinations, product
from time import perf_counter
from typing import Iterable, Optional, TypeVar

from device_selection.core.model import (
    ConnectionInfo,
    ConnectionPlan,
    Device,
    DeviceRequirement,
    DeviceSelectionRequest,
    EcosystemId,
    Filter,
    FilterOp,
    HubType,
    ParetoPoint,
    ProtocolId,
    SolutionItem,
)
from device_selection.core.pareto import ParetoArchive
from device_selection.core.pathfinding import find_connection, find_hub_connection, hub_required
from device_selection.data.catalog import Catalog


@dataclass(frozen=True)
class SolverConfig:
    max_bridge_ecosystems: int = 5
    max_hub_types: int = 4
    max_candidates_per_type: Optional[int] = None


T = TypeVar("T")


def _iter_subsets(items: list[T], max_size: int) -> Iterable[frozenset[T]]:
    n = len(items)
    for r in range(0, min(max_size, n) + 1):
        for comb in combinations(items, r):
            yield frozenset(comb)


def _iter_hub_sets(
    hub_types_by_ecosystem: dict[EcosystemId, list[HubType]],
    max_hub_types: int,
) -> Iterable[frozenset[HubType]]:
    """
    Iterate over all combinations of hub types where we pick at most one
    hub type per ecosystem, and at most max_hub_types total.
    Yields frozensets of HubType.
    """
    ecosystems = sorted(hub_types_by_ecosystem.keys())
    choices_per_ecosystem: list[list[Optional[HubType]]] = []
    for eco in ecosystems:
        options: list[Optional[HubType]] = [None] + hub_types_by_ecosystem[eco]
        choices_per_ecosystem.append(options)

    for combo in product(*choices_per_ecosystem):
        chosen = frozenset(ht for ht in combo if ht is not None)
        if len(chosen) <= max_hub_types:
            yield chosen


def _hub_type_to_requirement(hub_type: HubType, req_id: int) -> DeviceRequirement:
    protocol_filters = tuple(
        Filter(field="protocol", op=FilterOp.CONTAINS, value=p)
        for p in sorted(hub_type.protocols)
    )
    ecosystem_filter = Filter(field="ecosystem", op=FilterOp.CONTAINS, value=hub_type.ecosystem)
    return DeviceRequirement(
        requirement_id=req_id,
        device_type="smart_hub",
        count=1,
        connect_to_main_ecosystem=False,
        filters=(ecosystem_filter,) + protocol_filters,
    )


@dataclass(frozen=True)
class _Slot:
    """Unified representation of a device slot - both user requirements and hub slots."""
    requirement: DeviceRequirement
    candidates: list[Device]
    hub_type: Optional[HubType] = None


def _build_slots(
    req: DeviceSelectionRequest,
    devices_by_req: list[list[Device]],
    hub_set: frozenset[HubType],
    avail_ecosystems: frozenset[EcosystemId],
    catalog: Catalog,
    max_candidates: Optional[int],
) -> Optional[list[_Slot]]:
    """
    Build slots for all requirements + hub requirements.
    Returns None if any slot has no candidates (infeasible).
    """
    slots: list[_Slot] = []

    for i, r in enumerate(req.requirements):
        candidates = _filter_candidates(
            devices=devices_by_req[i],
            requirement=r,
            main_ecosystem=req.main_ecosystem,
            available_ecosystems=avail_ecosystems,
            available_hub_types=hub_set,
            max_candidates=max_candidates,
        )
        if not candidates:
            return None
        slots.append(_Slot(requirement=r, candidates=candidates))

    for hub_type in sorted(hub_set, key=lambda h: h.ecosystem):
        hub_req = _hub_type_to_requirement(hub_type, req_id=-1)
        hub_devices = list(catalog.devices_for_requirement(hub_req))
        if not hub_devices:
            return None
        hub_devices.sort(key=lambda d: d.price)
        filtered = _filter_candidates(
            devices=hub_devices,
            requirement=hub_req,
            main_ecosystem=req.main_ecosystem,
            available_ecosystems=avail_ecosystems,
            available_hub_types=hub_set,
            max_candidates=max_candidates,
            hub_target_ecosystem=hub_type.ecosystem
        )
        if not filtered:
            return None
        slots.append(_Slot(requirement=hub_req, candidates=hub_devices, hub_type=hub_type))

    return slots


def _filter_candidates(
    devices: list[Device],
    requirement: DeviceRequirement,
    main_ecosystem: EcosystemId,
    available_ecosystems: frozenset[EcosystemId],
    available_hub_types: frozenset[HubType],
    max_candidates: Optional[int],
    hub_target_ecosystem: Optional[EcosystemId] = None,
) -> list[Device]:
    """
    Filter to connectable devices keeping only the price/quality pareto front.
    Input devices must be sorted by price ascending.
    """
    res: list[Device] = []
    last_quality = -1.0
    for d in devices:
        if hub_target_ecosystem is None:
            conn = find_connection(
                device=d,
                require_main_ecosystem=requirement.connect_to_main_ecosystem,
                main_ecosystem=main_ecosystem,
                available_ecosystems=available_ecosystems,
                available_hub_types=available_hub_types,
            )
        else:
            conn = find_hub_connection(
                hub=d,
                target_ecosystem=hub_target_ecosystem
            )
        if conn is None:
            continue
        if d.quality <= last_quality:
            continue
        last_quality = d.quality
        res.append(d)
        if max_candidates is not None and len(res) >= max_candidates:
            break
    return res


def _repair_to_budget(
    budget: float,
    slots: list[_Slot],
    chosen_idx: list[int],
) -> Optional[list[Device]]:
    n = len(slots)
    while True:
        chosen = [slots[i].candidates[chosen_idx[i]] for i in range(n)]
        total = sum(slots[i].requirement.count * chosen[i].price for i in range(n))
        if total <= budget:
            return chosen

        best_i = -1
        best_cost = -1.0
        for i in range(n):
            if chosen_idx[i] <= 0:
                continue
            cost = slots[i].candidates[chosen_idx[i]].price * slots[i].requirement.count
            if cost > best_cost:
                best_cost = cost
                best_i = i

        if best_i == -1:
            return None
        chosen_idx[best_i] -= 1


def _build_solution(
    slots: list[_Slot],
    chosen_devices: list[Device],
    main_ecosystem: EcosystemId,
    available_ecosystems: frozenset[EcosystemId],
    available_hub_types: frozenset[HubType],
) -> Optional[ParetoPoint]:
    # first pass: find connection plans, assign item ids
    items_partial: list[tuple[int, Device, DeviceRequirement, ConnectionPlan]] = []
    for i, (slot, device) in enumerate(zip(slots, chosen_devices)):
        if slot.hub_type is None:
            conn = find_connection(
                device=device,
                require_main_ecosystem=slot.requirement.connect_to_main_ecosystem,
                main_ecosystem=main_ecosystem,
                available_ecosystems=available_ecosystems,
                available_hub_types=available_hub_types,
            )
        else:
            conn = find_hub_connection(
                hub=device,
                target_ecosystem=slot.hub_type.ecosystem,
            )
        if conn is None:
            return None
        items_partial.append((i, device, slot.requirement, conn))

    # build lookup: (ecosystem, protocol) -> item_id for hub devices
    hub_key_to_item_id: dict[tuple[EcosystemId, ProtocolId], int] = {}
    for item_id, device, req, _ in items_partial:
        if device.device_type == "smart_hub":
            for dc in device.direct_compat:
                hub_key_to_item_id[(dc.ecosystem, dc.protocol)] = item_id

    def _resolve(info: ConnectionInfo) -> ConnectionInfo:
        if info.hub_solution_item_id is not None:
            return info
        if not hub_required(info.protocol):
            return info
        item_id = hub_key_to_item_id.get((info.ecosystem, info.protocol))
        if item_id is None:
            return info
        return ConnectionInfo(
            ecosystem=info.ecosystem,
            protocol=info.protocol,
            hub_solution_item_id=item_id,
        )

    items: list[SolutionItem] = []
    used_ecosystems: set[EcosystemId] = set()
    used_hub_item_ids: set[int] = set()
    total_cost = 0.0

    for item_id, device, req, conn in items_partial:
        direct = _resolve(conn.connection_direct)
        final = _resolve(conn.connection_final) if conn.connection_final else None

        if direct.hub_solution_item_id is not None:
            used_hub_item_ids.add(direct.hub_solution_item_id)
        if final is not None and final.hub_solution_item_id is not None:
            used_hub_item_ids.add(final.hub_solution_item_id)

        used_ecosystems.add(direct.ecosystem)
        if final is not None:
            used_ecosystems.add(final.ecosystem)

        qty = req.count
        total_cost += qty * device.price

        # requirement_id = -1 means auto-added hub, expose as None to caller
        requirement_id = req.requirement_id if req.requirement_id != -1 else None

        items.append(SolutionItem(
            id=item_id,
            device=device,
            requirement_id=requirement_id,
            quantity=qty,
            connection=ConnectionPlan(
                connection_direct=direct,
                connection_final=final,
            ),
        ))

    avg_quality = sum(item.device.quality for item in items) / len(items)

    if len(used_hub_item_ids) != len(available_hub_types):
        # some hubs are useless - reject solution
        return None

    return ParetoPoint(
        items=tuple(items),
        total_cost=total_cost,
        avg_quality=avg_quality,
        num_ecosystems=len(used_ecosystems),
        num_hubs=len(used_hub_item_ids),
    )


def solve_enum_repair(
    req: DeviceSelectionRequest,
    catalog: Catalog,
    cfg: SolverConfig = SolverConfig(),
) -> ParetoArchive:
    start = perf_counter()

    devices_by_req: list[list[Device]] = []
    for r in req.requirements:
        devices = list(catalog.devices_for_requirement(r))
        devices.sort(key=lambda d: d.price)
        devices_by_req.append(devices)

    all_ecosystems = catalog.available_ecosystems()
    candidate_bridge_ecosystems: set[EcosystemId] = set()
    for devices in devices_by_req:
        for d in devices:
            for bc in d.bridge_compat:
                eco = bc.source_ecosystem
                if eco == req.main_ecosystem:
                    continue
                if req.include_ecosystems and eco not in req.include_ecosystems:
                    continue
                if eco in req.exclude_ecosystems:
                    continue
                if eco in all_ecosystems:
                    candidate_bridge_ecosystems.add(eco)

    archive = ParetoArchive()

    for bridge_set in _iter_subsets(
        sorted(candidate_bridge_ecosystems), cfg.max_bridge_ecosystems
    ):
        if perf_counter() - start >= req.time_budget_seconds:
            break

        avail_ecosystems = frozenset({req.main_ecosystem}) | bridge_set

        # collect distinct hub types per ecosystem
        hub_types_by_ecosystem: dict[EcosystemId, list[HubType]] = {}
        for ecosystem in avail_ecosystems:
            hub_types = catalog.available_hub_types_for_ecosystem(ecosystem)
            if hub_types:
                hub_types_by_ecosystem[ecosystem] = hub_types

        for hub_set in _iter_hub_sets(hub_types_by_ecosystem, cfg.max_hub_types):
            if perf_counter() - start >= req.time_budget_seconds:
                break

            slots = _build_slots(
                req=req,
                devices_by_req=devices_by_req,
                hub_set=hub_set,
                avail_ecosystems=avail_ecosystems,
                catalog=catalog,
                max_candidates=cfg.max_candidates_per_type,
            )
            if slots is None:
                continue

            chosen_idx = [len(slot.candidates) - 1 for slot in slots]
            chosen = _repair_to_budget(
                budget=req.budget,
                slots=slots,
                chosen_idx=chosen_idx,
            )
            if chosen is None:
                continue

            point = _build_solution(
                slots=slots,
                chosen_devices=chosen,
                main_ecosystem=req.main_ecosystem,
                available_ecosystems=avail_ecosystems,
                available_hub_types=hub_set,
            )
            if point is None:
                continue
            if point.total_cost <= req.budget + 1e-9:
                archive.add(point)

    return archive
