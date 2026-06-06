"""
Greedy baseline solvers. Used as a lower bound on heuristic performance.

Algorithm:
1. Enumerate subsets of non-main ecosystems (powerset).
2. For each subset, build a permissive available_hub_types set (the hub type
   with the largest protocol coverage in each ecosystem) - used only for
   feasibility routing in find_connection, not for actual hub purchase.
3. For each requirement, find all devices that have a valid ConnectionPlan
   under this ecosystem subset, then apply `device_selector` to pick one.
   (The whole-count quantity uses the same single model: homogeneous-within-req.)
4. Inspect the resulting connections to determine which ecosystems actually
   have a hub-required leg. Pick a hub only for those ecosystems (skipping
   ecosystems where every device connects via wifi or cloud).
5. For each ecosystem that needs a hub, apply `device_selector` to the hubs
   that cover the required protocol set.
6. Build SolutionItems (wiring hub_solution_item_id references), compute
   objectives, check budget, add to Pareto archive.

solve_greedy_cheapest and solve_greedy_quality are thin wrappers that supply
min-by-price and max-by-quality selectors respectively.
"""
from __future__ import annotations

from itertools import combinations
from time import perf_counter
from typing import Callable, Iterable, Iterator, Optional, Sequence, TypeVar

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
from device_selection.core.objectives import compute_objectives
from device_selection.core.pareto import ParetoArchive
from device_selection.core.pathfinding import (
    find_connection,
    find_hub_connection,
    hub_required,
)
from device_selection.data.catalog import Catalog


DeviceSelector = Callable[[Sequence[Device]], Device]

T = TypeVar("T")


def _powerset(items: Iterable[T]) -> Iterator[frozenset[T]]:
    items = list(items)
    for r in range(len(items) + 1):
        for combo in combinations(items, r):
            yield frozenset(combo)


def _hubs_in_ecosystem(catalog: Catalog, ecosystem: EcosystemId) -> list[Device]:
    """Hubs with at least one direct_compat record in the given ecosystem.
    
    Same approach as brute_force._hubs_in_ecosystem.
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


def _permissive_hub_type(
    catalog: Catalog, ecosystem: EcosystemId
) -> Optional[HubType]:
    """The hub type with the largest protocol set in this ecosystem.
    
    Used purely to make find_connection more permissive during routing - the
    actual hub purchase decision is made later, after we know which protocols
    are actually used.
    """
    hub_types = catalog.available_hub_types_for_ecosystem(ecosystem)
    if not hub_types:
        return None
    return max(hub_types, key=lambda ht: len(ht.protocols))


def solve_greedy(
    request: DeviceSelectionRequest,
    catalog: Catalog,
    device_selector: DeviceSelector,
) -> ParetoArchive:
    """
    Greedy baseline. Enumerates 2^E_b ecosystem subsets, picks one device
    per requirement using `device_selector` within each, then picks the
    cheapest/best hub for each ecosystem that actually needs one.
    """
    start = perf_counter()
    archive = ParetoArchive()

    def time_up() -> bool:
        return perf_counter() - start >= request.time_budget_seconds

    # Determine non-main ecosystems to enumerate, applying include/exclude.
    all_ecos = set(catalog.available_ecosystems())
    if request.include_ecosystems:
        all_ecos &= request.include_ecosystems
    all_ecos -= request.exclude_ecosystems
    all_ecos.add(request.main_ecosystem)
    non_main_ecos = all_ecos - {request.main_ecosystem}

    # Pre-fetch candidates per requirement (filtered by device type + req filters).
    candidates_per_req: list[list[Device]] = []
    for r in request.requirements:
        cands = list(catalog.devices_for_requirement(r))
        if not cands:
            return archive  # impossible to satisfy this requirement at all
        candidates_per_req.append(cands)

    # Cache permissive hub types and per-ecosystem hub lists.
    permissive_hub_types: dict[EcosystemId, HubType] = {}
    hub_cache: dict[EcosystemId, list[Device]] = {}
    for eco in all_ecos:
        ht = _permissive_hub_type(catalog, eco)
        if ht is not None:
            permissive_hub_types[eco] = ht

    def get_hubs(hub_type: HubType) -> list[Device]:
        protocol_filters = tuple(
            Filter(field="protocol", op=FilterOp.CONTAINS, value=p)
            for p in sorted(hub_type.protocols)
        )
        ecosystem_filter = Filter(field="ecosystem", op=FilterOp.CONTAINS, value=hub_type.ecosystem)
        req = DeviceRequirement(
            requirement_id=-1,
            device_type="smart_hub",
            count=1,
            connect_to_main_ecosystem=False,
            filters=(ecosystem_filter,) + protocol_filters,
        )

        return catalog.devices_for_requirement(req)

    for eco_subset in _powerset(non_main_ecos):
        if time_up():
            break

        avail_ecos = frozenset({request.main_ecosystem}) | eco_subset
        avail_hub_types = frozenset(
            permissive_hub_types[e] for e in avail_ecos
            if e in permissive_hub_types
        )

        # Phase 1: pick a device for each requirement.
        picks: list[tuple[DeviceRequirement, Device, ConnectionPlan]] = []
        feasible = True
        for r, cands in zip(request.requirements, candidates_per_req):
            valid: list[tuple[Device, ConnectionPlan]] = []
            for d in cands:
                plan = find_connection(
                    device=d,
                    require_main_ecosystem=r.connect_to_main_ecosystem,
                    main_ecosystem=request.main_ecosystem,
                    available_ecosystems=avail_ecos,
                    available_hub_types=avail_hub_types,
                )
                if plan is not None:
                    valid.append((d, plan))

            if not valid:
                feasible = False
                break

            devices_only = [d for d, _ in valid]
            chosen = device_selector(devices_only)
            chosen_plan = next(
                p for d, p in valid if d.device_id == chosen.device_id
            )
            picks.append((r, chosen, chosen_plan))

        if not feasible:
            continue

        # Phase 2: figure out which ecosystems actually have a hub-required leg.
        hub_protocols_by_eco: dict[EcosystemId, set[ProtocolId]] = {}
        for _, _, plan in picks:
            leg = plan.connection_direct
            if hub_required(leg.protocol):
                hub_protocols_by_eco.setdefault(leg.ecosystem, set()).add(leg.protocol)
            leg = plan.connection_final
            if leg is None:
                continue
            if hub_required(leg.protocol):
                hub_protocols_by_eco.setdefault(plan.connection_direct.ecosystem, set())
                hub_protocols_by_eco.setdefault(leg.ecosystem, set()).add(leg.protocol)

        # Phase 3: pick a hub for each ecosystem that needs one.
        hub_picks: dict[EcosystemId, Device] = {}
        for eco, required_protocols in hub_protocols_by_eco.items():
            hub_type = HubType(ecosystem=eco, protocols=required_protocols)
            hubs = get_hubs(hub_type)
            if not hubs:
                feasible = False
                break
            hub_picks[eco] = device_selector(hubs)

        if not feasible:
            continue

        # Phase 4: build SolutionItems, check budget, add to archive.
        point = _build_point(picks, hub_picks, request.main_ecosystem)
        if point is None:
            continue
        if point.total_cost > request.budget + 1e-9:
            continue
        archive.add(point)

    return archive


def solve_greedy_cheapest(
    request: DeviceSelectionRequest, catalog: Catalog
) -> ParetoArchive:
    """Greedy baseline picking the lowest-price valid device per requirement."""
    return solve_greedy(
        request, catalog,
        device_selector=lambda devices: min(devices, key=lambda d: d.price),
    )


def solve_greedy_quality(
    request: DeviceSelectionRequest, catalog: Catalog
) -> ParetoArchive:
    """Greedy baseline picking the highest-quality valid device per requirement."""
    return solve_greedy(
        request, catalog,
        device_selector=lambda devices: max(devices, key=lambda d: d.quality),
    )


def _build_point(
    picks: list[tuple[DeviceRequirement, Device, ConnectionPlan]],
    hub_picks: dict[EcosystemId, Device],
    main_ecosystem: EcosystemId,
) -> Optional[ParetoPoint]:
    """
    Materialize the chosen devices and hubs into SolutionItems with proper
    hub_solution_item_id back-references, then compute objectives.
    """
    items: list[SolutionItem] = []
    next_id = 0

    # Hub items first, so device items can reference their IDs.
    hub_key_to_item: dict[tuple[EcosystemId, ProtocolId], int] = {}
    for eco, hub_device in hub_picks.items():
        hub_conn = find_hub_connection(hub=hub_device, target_ecosystem=eco)
        if hub_conn is None:
            return None  # hub itself can't reach main; shouldn't normally happen
        items.append(SolutionItem(
            id=next_id,
            device=hub_device,
            requirement_id=None,
            quantity=1,
            connection=hub_conn,
        ))
        # Register this hub for every (eco, hub-required protocol) it serves.
        for dc in hub_device.direct_compat:
            if dc.ecosystem == eco and hub_required(dc.protocol):
                hub_key_to_item[(dc.ecosystem, dc.protocol)] = next_id
        next_id += 1

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

    for req, device, plan in picks:
        direct = resolve(plan.connection_direct)
        final = resolve(plan.connection_final) if plan.connection_final else None
        items.append(SolutionItem(
            id=next_id,
            device=device,
            requirement_id=req.requirement_id,
            quantity=req.count,
            connection=ConnectionPlan(connection_direct=direct, connection_final=final),
        ))
        next_id += 1

    obj = compute_objectives(items)
    return ParetoPoint(
        items=tuple(items),
        total_cost=obj.total_cost,
        avg_quality=obj.avg_quality,
        num_ecosystems=obj.num_ecosystems,
        num_hubs=obj.num_hubs,
    )
