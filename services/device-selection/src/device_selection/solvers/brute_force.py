"""
Exact enumeration solver. Used as ground truth to evaluate solve_enum_repair
on small instances.

Iterates over all possible sets of devices matching requirements + any hub for each ecosystem.
Algorithm:
1. Per requirement, enumerate all multisets of size `count` over matching devices.
2. Cross-product across requirements.
3. For each device combo, compute candidate ecosystems (union of compatibility
   record ecosystems, filtered by include/exclude; main_ecosystem always included).
4. Iterate subsets of non-main ecosystems -> available_ecosystems = subset + main.
5. Per available ecosystem, iterate hub choices: [None, ...all hubs that are
   directly compatible with that ecosystem].
6. Run find_connection per device, find_hub_connection per chosen hub.
7. Check budget and that every chosen hub is actually used by some connection.


"""
from __future__ import annotations

from itertools import combinations, combinations_with_replacement, product
from time import perf_counter
from typing import Iterable, Iterator, Optional, TypeVar

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
from device_selection.core.pathfinding import (
    find_connection,
    find_hub_connection,
    hub_required,
)
from device_selection.data.catalog import Catalog


T = TypeVar("T")


def _powerset(items: Iterable[T]) -> Iterator[frozenset[T]]:
    items = list(items)
    for r in range(len(items) + 1):
        for combo in combinations(items, r):
            yield frozenset(combo)


def _hub_type_for(hub: Device, ecosystem: EcosystemId) -> HubType:
    """HubType this hub represents when used to serve the given ecosystem."""
    protocols = frozenset(
        dc.protocol for dc in hub.direct_compat if dc.ecosystem == ecosystem
    )
    return HubType(ecosystem=ecosystem, protocols=protocols)


def _hubs_in_ecosystem(catalog: Catalog, ecosystem: EcosystemId) -> list[Device]:
    """Hubs that have at least one direct_compat record in the given ecosystem.
    
    The catalog query may return hubs whose only relation to `ecosystem` is via
    e.g. a bridge_compat record; we want hubs that can directly serve devices
    in this ecosystem.
    """
    candidates = catalog.devices_for_requirement(DeviceRequirement(
        requirement_id=-1,
        device_type="smart_hub",
        count=1,
        connect_to_main_ecosystem=False,
        filters=(Filter(field="ecosystem", op=FilterOp.CONTAINS, value=ecosystem),),
    ))
    return [
        h for h in candidates
        if any(dc.ecosystem == ecosystem for dc in h.direct_compat)
    ]


def solve_brute_force(req: DeviceSelectionRequest, catalog: Catalog) -> ParetoArchive:
    start = perf_counter()
    archive = ParetoArchive()

    def time_up() -> bool:
        return perf_counter() - start >= req.time_budget_seconds

    # 1. candidates per requirement
    candidates_per_req: list[list[Device]] = []
    for r in req.requirements:
        cands = list(catalog.devices_for_requirement(r))
        if not cands:
            return archive
        candidates_per_req.append(cands)

    # 2. multisets per requirement
    multisets_per_req = [
        list(combinations_with_replacement(cands, r.count))
        for r, cands in zip(req.requirements, candidates_per_req)
    ]

    included = frozenset(req.include_ecosystems) if req.include_ecosystems else None
    excluded = frozenset(req.exclude_ecosystems)
    hub_cache: dict[EcosystemId, list[Device]] = {}

    def get_hubs(eco: EcosystemId) -> list[Device]:
        if eco not in hub_cache:
            hub_cache[eco] = _hubs_in_ecosystem(catalog, eco)
        return hub_cache[eco]

    for combo in product(*multisets_per_req):
        if time_up():
            break

        device_cost = sum(d.price for ms in combo for d in ms)
        if device_cost > req.budget + 1e-9:
            continue

        # 3. ecosystems present in any chosen device (filtered by include/exclude)
        ecos_present: set[EcosystemId] = set()
        for ms in combo:
            for d in ms:
                ecos_present.update(dc.ecosystem for dc in d.direct_compat)
                for bc in d.bridge_compat:
                    ecos_present.add(bc.source_ecosystem)
                    ecos_present.add(bc.target_ecosystem)
        if included is not None:
            ecos_present &= included
        ecos_present -= excluded
        ecos_present.add(req.main_ecosystem)  # main is always available regardless
        non_main_ecos = ecos_present - {req.main_ecosystem}

        # 4. iterate subsets of non-main ecosystems
        for eco_subset in _powerset(non_main_ecos):
            if time_up():
                break
            avail_ecos = frozenset({req.main_ecosystem}) | eco_subset

            # 5. hub picks per available ecosystem; None = no hub bought
            ecos_ordered = sorted(avail_ecos)
            hub_options: list[list[Optional[Device]]] = [
                [None, *get_hubs(eco)] for eco in ecos_ordered
            ]

            for hub_pick in product(*hub_options):
                if time_up():
                    break

                # Track (ecosystem, hub) pairs so HubType is bound to the chosen ecosystem
                chosen_hub_pairs: list[tuple[EcosystemId, Device]] = [
                    (eco, h) for eco, h in zip(ecos_ordered, hub_pick) if h is not None
                ]
                total_cost = device_cost + sum(h.price for _, h in chosen_hub_pairs)
                if total_cost > req.budget + 1e-9:
                    continue

                avail_hub_types = frozenset(
                    _hub_type_for(h, eco) for eco, h in chosen_hub_pairs
                )

                # 6. find connection plans for each device
                dev_plans: list[tuple[Device, DeviceRequirement, ConnectionPlan]] = []
                feasible = True
                for r, ms in zip(req.requirements, combo):
                    for d in ms:
                        plan = find_connection(
                            device=d,
                            require_main_ecosystem=r.connect_to_main_ecosystem,
                            main_ecosystem=req.main_ecosystem,
                            available_ecosystems=avail_ecos,
                            available_hub_types=avail_hub_types,
                        )
                        if plan is None:
                            feasible = False
                            break
                        dev_plans.append((d, r, plan))
                    if not feasible:
                        break
                if not feasible:
                    continue

                # find_hub_connection per chosen hub, target = its chosen ecosystem
                hub_plans: list[tuple[Device, EcosystemId, ConnectionPlan]] = []
                for eco, h in chosen_hub_pairs:
                    hp = find_hub_connection(hub=h, target_ecosystem=eco)
                    if hp is None:
                        feasible = False
                        break
                    hub_plans.append((h, eco, hp))
                if not feasible:
                    continue

                # 7. every chosen (eco, hub) must be used by some device's connection
                required_keys: set[tuple[EcosystemId, ProtocolId]] = {
                    (info.ecosystem, info.protocol)
                    for _, _, plan in dev_plans
                    for info in (plan.connection_direct, plan.connection_final)
                    if info is not None and hub_required(info.protocol)
                }
                if any(
                    not (
                        {
                            (dc.ecosystem, dc.protocol)
                            for dc in h.direct_compat
                            if dc.ecosystem == eco and hub_required(dc.protocol)
                        }
                        & required_keys
                    )
                    for eco, h in chosen_hub_pairs
                ):
                    continue

                point = _build_point(dev_plans, hub_plans, total_cost)
                if point is not None:
                    archive.add(point)

    return archive


def _build_point(
    dev_plans: list[tuple[Device, DeviceRequirement, ConnectionPlan]],
    hub_plans: list[tuple[Device, EcosystemId, ConnectionPlan]],
    total_cost: float,
) -> Optional[ParetoPoint]:
    """Group multiset duplicates into items, resolve hub_solution_item_id, compute objectives."""

    # group device picks by (requirement_id, device_id) -> (device, req, plan, qty)
    grouped: dict[
        tuple[int, int],
        tuple[Device, DeviceRequirement, ConnectionPlan, int],
    ] = {}
    order: list[tuple[int, int]] = []
    for d, r, plan in dev_plans:
        key = (r.requirement_id, d.device_id)
        if key in grouped:
            old = grouped[key]
            grouped[key] = (old[0], old[1], old[2], old[3] + 1)
        else:
            grouped[key] = (d, r, plan, 1)
            order.append(key)

    # Each items_partial entry carries: (item_id, device, req_id, qty, plan, hub_eco_or_None)
    # hub_eco_or_None is set for hub items so we know which ecosystem to register
    # in hub_key_to_item; None for non-hub items.
    items_partial: list[
        tuple[int, Device, Optional[int], int, ConnectionPlan, Optional[EcosystemId]]
    ] = []
    next_id = 0
    for key in order:
        d, r, plan, qty = grouped[key]
        items_partial.append((next_id, d, r.requirement_id, qty, plan, None))
        next_id += 1
    for h, eco, plan in hub_plans:
        items_partial.append((next_id, h, None, 1, plan, eco))
        next_id += 1

    # (ecosystem, protocol) -> hub item_id lookup, restricted to the ecosystem
    # each hub was chosen for.
    hub_key_to_item: dict[tuple[EcosystemId, ProtocolId], int] = {}
    for item_id, device, _, _, _, hub_eco in items_partial:
        if hub_eco is None:
            continue
        for dc in device.direct_compat:
            if dc.ecosystem != hub_eco:
                continue
            if hub_required(dc.protocol):
                hub_key_to_item[(dc.ecosystem, dc.protocol)] = item_id

    def resolve(info: ConnectionInfo) -> ConnectionInfo:
        if info.hub_solution_item_id is not None or not hub_required(info.protocol):
            return info
        item_id = hub_key_to_item.get((info.ecosystem, info.protocol))
        if item_id is None:
            return info
        return ConnectionInfo(
            ecosystem=info.ecosystem,
            protocol=info.protocol,
            hub_solution_item_id=item_id,
        )

    items: list[SolutionItem] = []
    used_ecos: set[EcosystemId] = set()
    for item_id, device, req_id, qty, plan, _ in items_partial:
        direct = resolve(plan.connection_direct)
        final = resolve(plan.connection_final) if plan.connection_final else None
        used_ecos.add(direct.ecosystem)
        if final is not None:
            used_ecos.add(final.ecosystem)
        items.append(SolutionItem(
            id=item_id,
            device=device,
            requirement_id=req_id,
            quantity=qty,
            connection=ConnectionPlan(connection_direct=direct, connection_final=final),
        ))

    # avg quality per advisor's spec: per-req avg, then avg across reqs + hub items
    req_qs: dict[int, list[float]] = {}
    for d, r, _ in dev_plans:
        req_qs.setdefault(r.requirement_id, []).append(d.quality)
    req_avgs = [sum(qs) / len(qs) for qs in req_qs.values()]
    hub_qs = [h.quality for h, _, _ in hub_plans]
    all_q = req_avgs + hub_qs
    avg_quality = sum(all_q) / len(all_q) if all_q else 0.0

    return ParetoPoint(
        items=tuple(items),
        total_cost=total_cost,
        avg_quality=avg_quality,
        num_ecosystems=len(used_ecos),
        num_hubs=len(hub_plans),
    )
