"""
Objective computation for a candidate solution.
"""

from __future__ import annotations

from dataclasses import dataclass
from device_selection.core.model import EcosystemId, SolutionItem


@dataclass(frozen=True, slots=True)
class Objectives:
    total_cost:      float
    avg_quality:     float
    num_ecosystems:  int
    num_hubs:        int


def compute_objectives(
    items: tuple[SolutionItem, ...]
) -> Objectives:
    """
    Derive total cost and all three objective values from a candidate item set.
    """
    return Objectives(
        total_cost     = _total_cost(items),
        avg_quality    = _avg_quality(items),
        num_ecosystems = _num_ecosystems(items),
        num_hubs       = _num_hubs(items),
    )


def _total_cost(items: tuple[SolutionItem, ...]) -> float:
    return sum(it.device.price * it.quantity for it in items)


def _avg_quality(items: tuple[SolutionItem, ...]) -> float:
    """
    Q(s) = mean( {per_req_avg_quality_i} + {hub_quality_j} )
    """
    hub_items    = [it for it in items if it.device.device_type == "smart_hub"]
    device_items = [it for it in items if it.device.device_type != "smart_hub"]

    # group non-hub items by requirement
    by_req: dict[int, list[SolutionItem]] = {}
    for it in device_items:
        if it.requirement_id is not None:
            by_req.setdefault(it.requirement_id, []).append(it)

    # --- avg_quality ---
    # per-requirement averages
    by_req: dict[int, list[SolutionItem]] = {}
    for item in device_items:
        if item.requirement_id is not None:
            by_req.setdefault(item.requirement_id, []).append(item)

    req_avgs: list[float] = []
    for items in by_req.values():
        if items:
            # expand quantities
            qualities = [
                item.device.quality
                for item in items
                for _ in range(item.quantity)
            ]
            req_avgs.append(sum(qualities) / len(qualities))

    hub_qualities = [it.device.quality for it in hub_items]
    all_terms     = req_avgs + hub_qualities

    return sum(all_terms) / len(all_terms) if all_terms else 0.0


def _num_ecosystems(
    items: tuple[SolutionItem, ...],
) -> int:
    ecosystems: set[EcosystemId] = set()
    for it in items:
        plan = it.connection
        ecosystems.add(plan.connection_direct.ecosystem)
        if plan.connection_final is not None:
            ecosystems.add(plan.connection_final.ecosystem)
    return len(ecosystems)


def _num_hubs(items: tuple[SolutionItem, ...]) -> int:
    return sum(1 for it in items if it.device.device_type == "smart_hub")
