"""
Exact enumeration solver. Used as a baseline to evaluate solve_enum_repair.

Differences from solve_enum_repair:
- Allows heterogeneous picks per requirement (multisets via
  combinations_with_replacement) instead of homogeneous (same model x count).
- Keeps all connectable candidates per requirement (no per-candidate
  price/quality Pareto pruning).
- Enumerates specific hub devices, not just hub types.

Intractable on large catalogs. Intended for small instances to validate the
heuristic against optimal Pareto fronts.
"""
from __future__ import annotations

from itertools import combinations_with_replacement, product
from time import perf_counter
from typing import Optional

from device_selection.core.model import (
    ConnectionInfo,
    ConnectionPlan,
    Device,
    DeviceRequirement,
    DeviceSelectionRequest,
    EcosystemId,
    HubType,
    ParetoPoint,
    ProtocolId,
    SolutionItem,
)
from device_selection.core.pareto import ParetoArchive
from device_selection.core.pathfinding import (
    find_connection,
    find_hub_connection,
    hub_required,
)
from device_selection.data.catalog import Catalog
from device_selection.solvers.enum_repair import (
    SolverConfig,
    _hub_type_to_requirement,
    _iter_hub_sets,
    _iter_subsets,
)


def _connectable_candidates(
    devices: list[Device],
    requirement: DeviceRequirement,
    main_ecosystem: EcosystemId,
    available_ecosystems: frozenset[EcosystemId],
    available_hub_types: frozenset[HubType],
    max_candidates: Optional[int],
    hub_target_ecosystem: Optional[EcosystemId] = None,
) -> list[Device]:
    """
    Return all connectable devices, sorted by price ascending.
    No Pareto-by-price/quality pruning - that would be a heuristic.
    """
    res: list[Device] = []
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
                target_ecosystem=hub_target_ecosystem,
            )
        if conn is None:
            continue
        res.append(d)
        if max_candidates is not None and len(res) >= max_candidates:
            break
    return res


def _build_solution_brute(
    requirements: list[DeviceRequirement],
    chosen_per_req: list[tuple[Device, ...]],
    hub_types_ordered: list[HubType],
    chosen_hub_devices: list[Device],
    main_ecosystem: EcosystemId,
    available_ecosystems: frozenset[EcosystemId],
    available_hub_types: frozenset[HubType],
) -> Optional[ParetoPoint]:
    """
    Build a ParetoPoint from a brute-force pick.

    chosen_per_req[i] is a multiset of devices satisfying requirements[i].
    chosen_hub_devices[i] is a specific hub device for hub_types_ordered[i].

    Identical devices within a requirement's multiset are grouped into one
    SolutionItem with the appropriate quantity.
    """
    items_partial: list[
        tuple[int, Device, Optional[int], int, ConnectionPlan]
    ] = []
    next_item_id = 0

    for req, devices in zip(requirements, chosen_per_req):
        # Group identical devices, preserving first-seen order for stable output
        order: list[int] = []
        counts: dict[int, int] = {}
        device_by_id: dict[int, Device] = {}
        for d in devices:
            if d.device_id not in counts:
                counts[d.device_id] = 0
                order.append(d.device_id)
                device_by_id[d.device_id] = d
            counts[d.device_id] += 1

        for device_id in order:
            d = device_by_id[device_id]
            qty = counts[device_id]
            conn = find_connection(
                device=d,
                require_main_ecosystem=req.connect_to_main_ecosystem,
                main_ecosystem=main_ecosystem,
                available_ecosystems=available_ecosystems,
                available_hub_types=available_hub_types,
            )
            if conn is None:
                return None
            items_partial.append((next_item_id, d, req.requirement_id, qty, conn))
            next_item_id += 1

    for hub_type, hub_dev in zip(hub_types_ordered, chosen_hub_devices):
        conn = find_hub_connection(hub=hub_dev, target_ecosystem=hub_type.ecosystem)
        if conn is None:
            return None
        items_partial.append((next_item_id, hub_dev, None, 1, conn))
        next_item_id += 1

    # Lookup: (ecosystem, protocol) -> item_id for hubs, used to fill
    # hub_solution_item_id on connections that need a hub.
    hub_key_to_item_id: dict[tuple[EcosystemId, ProtocolId], int] = {}
    for item_id, device, _, _, _ in items_partial:
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

    for item_id, device, req_id, qty, conn in items_partial:
        direct = _resolve(conn.connection_direct)
        final = _resolve(conn.connection_final) if conn.connection_final else None

        if direct.hub_solution_item_id is not None:
            used_hub_item_ids.add(direct.hub_solution_item_id)
        if final is not None and final.hub_solution_item_id is not None:
            used_hub_item_ids.add(final.hub_solution_item_id)

        used_ecosystems.add(direct.ecosystem)
        if final is not None:
            used_ecosystems.add(final.ecosystem)

        total_cost += qty * device.price

        items.append(
            SolutionItem(
                id=item_id,
                device=device,
                requirement_id=req_id,
                quantity=qty,
                connection=ConnectionPlan(
                    connection_direct=direct,
                    connection_final=final,
                ),
            )
        )

    if len(used_hub_item_ids) != len(available_hub_types):
        # some hubs are useless - reject solution (matches enum_repair)
        return None

    # Per advisor's spec:
    # 1. For each requirement, average quality across its multiset.
    # 2. Average across requirement-averages and individual hub qualities.
    # This avoids weighting big-count requirements (e.g. 5 lamps) over small ones (1 leak sensor).
    req_qualities: list[float] = []
    for req, devices in zip(requirements, chosen_per_req):
        req_qualities.append(sum(d.quality for d in devices) / len(devices))
    hub_qualities = [d.quality for d in chosen_hub_devices]
    all_q = req_qualities + hub_qualities
    avg_quality = sum(all_q) / len(all_q) if all_q else 0.0

    return ParetoPoint(
        items=tuple(items),
        total_cost=total_cost,
        avg_quality=avg_quality,
        num_ecosystems=len(used_ecosystems),
        num_hubs=len(used_hub_item_ids),
    )


def solve_brute_force(
    req: DeviceSelectionRequest,
    catalog: Catalog,
    cfg: SolverConfig = SolverConfig(),
) -> ParetoArchive:
    start = perf_counter()
    archive = ParetoArchive()

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

    def time_up() -> bool:
        return perf_counter() - start >= req.time_budget_seconds

    for bridge_set in _iter_subsets(
        sorted(candidate_bridge_ecosystems), cfg.max_bridge_ecosystems
    ):
        if time_up():
            break

        avail_ecosystems = frozenset({req.main_ecosystem}) | bridge_set

        hub_types_by_ecosystem: dict[EcosystemId, list[HubType]] = {}
        for ecosystem in avail_ecosystems:
            hub_types = catalog.available_hub_types_for_ecosystem(ecosystem)
            if hub_types:
                hub_types_by_ecosystem[ecosystem] = hub_types

        for hub_set in _iter_hub_sets(hub_types_by_ecosystem, cfg.max_hub_types):
            if time_up():
                break

            # candidates per requirement (no Pareto filter - exact solver)
            candidates_per_req: list[list[Device]] = []
            infeasible = False
            for i, r in enumerate(req.requirements):
                cands = _connectable_candidates(
                    devices=devices_by_req[i],
                    requirement=r,
                    main_ecosystem=req.main_ecosystem,
                    available_ecosystems=avail_ecosystems,
                    available_hub_types=hub_set,
                    max_candidates=cfg.max_candidates_per_type,
                )
                if not cands:
                    infeasible = True
                    break
                candidates_per_req.append(cands)
            if infeasible:
                continue

            # specific hub devices per hub type
            hub_types_ordered = sorted(hub_set, key=lambda h: h.ecosystem)
            hub_devices_per_type: list[list[Device]] = []
            for hub_type in hub_types_ordered:
                hub_req = _hub_type_to_requirement(hub_type, req_id=-1)
                hub_devices = list(catalog.devices_for_requirement(hub_req))
                hub_devices.sort(key=lambda d: d.price)
                filtered = _connectable_candidates(
                    devices=hub_devices,
                    requirement=hub_req,
                    main_ecosystem=req.main_ecosystem,
                    available_ecosystems=avail_ecosystems,
                    available_hub_types=hub_set,
                    max_candidates=cfg.max_candidates_per_type,
                    hub_target_ecosystem=hub_type.ecosystem,
                )
                if not filtered:
                    infeasible = True
                    break
                hub_devices_per_type.append(filtered)
            if infeasible:
                continue

            # multisets per requirement (size = req.count, with replacement)
            multisets_per_req = [
                list(combinations_with_replacement(cands, r.count))
                for r, cands in zip(req.requirements, candidates_per_req)
            ]

            # cross product over multisets, then over hub device choices
            for combo in product(*multisets_per_req):
                if time_up():
                    break

                # cheap prune: minimum cost just from device picks
                base_cost = sum(sum(d.price for d in ms) for ms in combo)
                if base_cost > req.budget + 1e-9:
                    continue

                # product([]) yields one empty tuple, so the no-hub case still runs once
                for hub_combo in product(*hub_devices_per_type):
                    if time_up():
                        break

                    total = base_cost + sum(d.price for d in hub_combo)
                    if total > req.budget + 1e-9:
                        continue

                    point = _build_solution_brute(
                        requirements=list(req.requirements),
                        chosen_per_req=list(combo),
                        hub_types_ordered=hub_types_ordered,
                        chosen_hub_devices=list(hub_combo),
                        main_ecosystem=req.main_ecosystem,
                        available_ecosystems=avail_ecosystems,
                        available_hub_types=hub_set,
                    )
                    if point is None:
                        continue
                    archive.add(point)

    return archive