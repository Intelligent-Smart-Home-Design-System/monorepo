"""
Solution validator.
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import Optional

from device_selection.core.model import (
    ConnectionInfo,
    ConnectionPlan,
    Device,
    DeviceRequirement,
    DeviceSelectionRequest,
    EcosystemId,
    Filter,
    ParetoPoint,
    SolutionItem,
)
from device_selection.core.objectives import compute_objectives
from device_selection.core.pathfinding import hub_required
from device_selection.data.catalog import Catalog, _matches_filters


# ------------------------------------------------------------------ #
#  Public surface                                                     #
# ------------------------------------------------------------------ #

@dataclass(frozen=True)
class ValidationError:
    code: str
    message: str

    def __str__(self) -> str:
        return f"[{self.code}] {self.message}"


def validate_solution(
    req: DeviceSelectionRequest,
    sol: ParetoPoint,
) -> list[ValidationError]:
    """
    Validate a single ParetoPoint against the request that produced it.

    Checks (in order):
      1. Requirement coverage  — correct device type, filters, quantity
      2. Budget                — total_cost <= request.budget
      3. Recorded objectives   — total_cost / avg_quality / num_ecosystems /
                                 num_hubs match recomputed values
      4. Hub items             — hubs connect directly over a hubless protocol
      5. Device connection plans
           a. direct-only: direct_compat exists; if hub required, hub item
              exists and supports the protocol in that ecosystem
           b. bridge: bridge_compat record exists; source-side hub supports
              the bridging protocol; target-side hub present if protocol
              is not cloud/hubless
      6. No dead hubs          — every hub item is referenced by ≥1 device leg
      7. Ecosystem filters     — include_ecosystems / exclude_ecosystems
    """
    errors: list[ValidationError] = []

    # index items by their id for connection-plan cross-referencing
    items_by_id: dict[int, SolutionItem] = {item.id: item for item in sol.items}

    hub_items = [it for it in sol.items if it.device.device_type == "smart_hub"]
    device_items = [it for it in sol.items if it.device.device_type != "smart_hub"]

    # ---- 1. Requirement coverage ----
    errors.extend(_check_requirements(req, device_items))

    # ---- 2. Budget ----
    errors.extend(_check_budget(req, sol))

    # ---- 3. Recorded objectives ----
    errors.extend(_check_objectives(req, sol))

    # ---- 4. Hub self-connectivity ----
    for hub_item in hub_items:
        errors.extend(_check_hub_self_connectivity(hub_item))

    # ---- 5. Device connection plans ----
    for item in device_items:
        errors.extend(_check_device_connection(item, items_by_id, req))

    # ---- 6. No dead hubs ----
    errors.extend(_check_no_dead_hubs(hub_items, device_items))

    # ---- 7. Ecosystem filters ----
    errors.extend(_check_ecosystem_filters(req, sol))

    return errors


def _check_requirements(
    req: DeviceSelectionRequest,
    device_items: list[SolutionItem],
) -> list[ValidationError]:
    errors: list[ValidationError] = []

    # group solution items by requirement_id
    by_req: dict[int, list[SolutionItem]] = {}
    for item in device_items:
        if item.requirement_id is not None:
            by_req.setdefault(item.requirement_id, []).append(item)

    for r in req.requirements:
        items = by_req.get(r.requirement_id, [])

        # total quantity (items carry quantity for homogeneous packs)
        total_qty = sum(it.quantity for it in items)
        if total_qty != r.count:
            errors.append(ValidationError(
                code="REQ_COUNT_MISMATCH",
                message=(
                    f"Requirement {r.requirement_id} ({r.device_type}): "
                    f"expected {r.count} devices, got {total_qty}"
                ),
            ))

        for item in items:
            d = item.device

            # device type
            if d.device_type != r.device_type:
                errors.append(ValidationError(
                    code="REQ_TYPE_MISMATCH",
                    message=(
                        f"Requirement {r.requirement_id}: expected type "
                        f"{r.device_type!r}, item {item.id} has {d.device_type!r}"
                    ),
                ))

            # filters
            if r.filters:
                try:
                    if not _matches_filters(d, r.filters):
                        errors.append(ValidationError(
                            code="REQ_FILTER_MISMATCH",
                            message=(
                                f"Requirement {r.requirement_id}: device "
                                f"{d.device_id} ({d.brand} {d.model}) "
                                f"does not satisfy filters"
                            ),
                        ))
                except TypeError as exc:
                    errors.append(ValidationError(
                        code="REQ_FILTER_TYPE_ERROR",
                        message=(
                            f"Requirement {r.requirement_id}: filter type "
                            f"error for device {d.device_id}: {exc}"
                        ),
                    ))

    # items with no matching requirement
    known_req_ids = {r.requirement_id for r in req.requirements}
    for item in device_items:
        if item.requirement_id not in known_req_ids:
            errors.append(ValidationError(
                code="UNKNOWN_REQUIREMENT_ID",
                message=(
                    f"Item {item.id} (device {item.device.device_id}) "
                    f"references unknown requirement_id {item.requirement_id}"
                ),
            ))

    return errors


def _check_budget(
    req: DeviceSelectionRequest,
    sol: ParetoPoint,
) -> list[ValidationError]:
    if sol.total_cost > req.budget + 1e-6:
        return [ValidationError(
            code="BUDGET_EXCEEDED",
            message=(
                f"total_cost {sol.total_cost:.2f} exceeds budget "
                f"{req.budget:.2f}"
            ),
        )]
    return []


def _check_objectives(
    req: DeviceSelectionRequest,
    sol: ParetoPoint,
) -> list[ValidationError]:
    errors: list[ValidationError] = []
    recomputed = compute_objectives(sol.items)

    checks = [
        ("total_cost",     sol.total_cost,     recomputed.total_cost,     0.01),
        ("avg_quality",    sol.avg_quality,     recomputed.avg_quality,    1e-6),
        ("num_ecosystems", sol.num_ecosystems,  recomputed.num_ecosystems, 0),
        ("num_hubs",       sol.num_hubs,        recomputed.num_hubs,       0),
    ]
    for name, recorded, expected, tol in checks:
        if abs(recorded - expected) > tol:
            errors.append(ValidationError(
                code=f"{name.upper()}_MISMATCH",
                message=(
                    f"recorded {name} {recorded} != recomputed {expected}"
                ),
            ))
    return errors


def _check_hub_self_connectivity(hub_item: SolutionItem) -> list[ValidationError]:
    """
    A hub must connect to its own ecosystem over a hubless protocol.
    Its connection plan must be direct-only (no bridge) and the protocol
    must not require a hub.
    """
    errors: list[ValidationError] = []
    hub   = hub_item.device
    plan  = hub_item.connection

    if plan.connection_final is not None:
        errors.append(ValidationError(
            code="HUB_BRIDGED_CONNECTION",
            message=(
                f"Hub item {hub_item.id} (device {hub.device_id}) "
                f"has a bridge connection — hubs must connect directly"
            ),
        ))
        return errors   # rest of checks require direct-only

    direct = plan.connection_direct
    if hub_required(direct.protocol):
        errors.append(ValidationError(
            code="HUB_REQUIRES_HUB",
            message=(
                f"Hub item {hub_item.id} (device {hub.device_id}) "
                f"connects via hub-required protocol {direct.protocol!r} — "
                f"hubs must use a hubless protocol (e.g. wifi)"
            ),
        ))

    # verify the hub actually has that direct_compat entry
    has_entry = any(
        dc.ecosystem == direct.ecosystem and dc.protocol == direct.protocol
        for dc in hub.direct_compat
    )
    if not has_entry:
        errors.append(ValidationError(
            code="HUB_NO_DIRECT_COMPAT",
            message=(
                f"Hub item {hub_item.id} (device {hub.device_id}) "
                f"has no direct_compat for "
                f"({direct.ecosystem}, {direct.protocol})"
            ),
        ))

    return errors


def _check_device_connection(
    item: SolutionItem,
    items_by_id: dict[int, SolutionItem],
    req: DeviceSelectionRequest,
) -> list[ValidationError]:
    plan   = item.connection
    device = item.device

    if plan.connection_final is None:
        return _check_direct_connection(item, items_by_id, req)
    else:
        return _check_bridge_connection(item, items_by_id, req)


def _check_direct_connection(
    item: SolutionItem,
    items_by_id: dict[int, SolutionItem],
    req: DeviceSelectionRequest,
) -> list[ValidationError]:
    errors: list[ValidationError] = []
    device = item.device
    direct = item.connection.connection_direct

    # must terminate at main ecosystem for connect_to_main_ecosystem reqs
    r = _find_requirement(item, req)
    if r is not None and r.connect_to_main_ecosystem:
        if direct.ecosystem != req.main_ecosystem:
            errors.append(ValidationError(
                code="WRONG_TERMINATION_ECOSYSTEM",
                message=(
                    f"Item {item.id}: requirement {r.requirement_id} requires "
                    f"termination at {req.main_ecosystem!r}, but direct "
                    f"connection ends at {direct.ecosystem!r}"
                ),
            ))

    # device must have the direct_compat entry
    has_compat = any(
        dc.ecosystem == direct.ecosystem and dc.protocol == direct.protocol
        for dc in device.direct_compat
    )
    if not has_compat:
        errors.append(ValidationError(
            code="NO_DIRECT_COMPAT",
            message=(
                f"Item {item.id} (device {device.device_id}): no direct_compat "
                f"for ({direct.ecosystem}, {direct.protocol})"
            ),
        ))

    # if the protocol requires a hub, the referenced hub item must exist
    # and must support that protocol in that ecosystem
    if hub_required(direct.protocol):
        errors.extend(_check_hub_reference(
            item_id      = item.id,
            leg_label    = "direct",
            hub_item_id  = direct.hub_solution_item_id,
            ecosystem    = direct.ecosystem,
            protocol     = direct.protocol,
            items_by_id  = items_by_id,
        ))

    return errors


def _check_bridge_connection(
    item: SolutionItem,
    items_by_id: dict[int, SolutionItem],
    req: DeviceSelectionRequest,
) -> list[ValidationError]:
    errors: list[ValidationError] = []
    device = item.device
    direct = item.connection.connection_direct
    final  = item.connection.connection_final   # not None here

    # termination check — final leg must reach main ecosystem
    r = _find_requirement(item, req)
    if r is not None and r.connect_to_main_ecosystem:
        if final.ecosystem != req.main_ecosystem:
            errors.append(ValidationError(
                code="WRONG_TERMINATION_ECOSYSTEM",
                message=(
                    f"Item {item.id}: requirement {r.requirement_id} requires "
                    f"termination at {req.main_ecosystem!r}, but bridge final "
                    f"leg ends at {final.ecosystem!r}"
                ),
            ))

    # device must have a bridge_compat record matching
    # (direct.ecosystem -> final.ecosystem, final.protocol)
    has_bridge_compat = any(
        bc.source_ecosystem == direct.ecosystem
        and bc.target_ecosystem == final.ecosystem
        and bc.protocol == final.protocol
        for bc in device.bridge_compat
    )
    if not has_bridge_compat:
        errors.append(ValidationError(
            code="NO_BRIDGE_COMPAT",
            message=(
                f"Item {item.id} (device {device.device_id}): no bridge_compat "
                f"for ({direct.ecosystem} → {final.ecosystem}, {final.protocol})"
            ),
        ))

    # device must also have the direct_compat entry for the source leg
    has_direct_compat = any(
        dc.ecosystem == direct.ecosystem and dc.protocol == direct.protocol
        for dc in device.direct_compat
    )
    if not has_direct_compat:
        errors.append(ValidationError(
            code="NO_DIRECT_COMPAT",
            message=(
                f"Item {item.id} (device {device.device_id}): no direct_compat "
                f"for source leg ({direct.ecosystem}, {direct.protocol})"
            ),
        ))

    # source-side hub: required if protocol needs a hub
    if hub_required(direct.protocol):
        errors.extend(_check_hub_reference(
            item_id     = item.id,
            leg_label   = "bridge-source",
            hub_item_id = direct.hub_solution_item_id,
            ecosystem   = direct.ecosystem,
            protocol    = direct.protocol,
            items_by_id = items_by_id,
        ))

    # target-side hub: required unless the bridging protocol is cloud/hubless
    if hub_required(final.protocol) and final.protocol != "cloud":
        errors.extend(_check_hub_reference(
            item_id     = item.id,
            leg_label   = "bridge-target",
            hub_item_id = final.hub_solution_item_id,
            ecosystem   = final.ecosystem,
            protocol    = final.protocol,
            items_by_id = items_by_id,
        ))

    return errors


def _check_hub_reference(
    item_id: int,
    leg_label: str,
    hub_item_id: Optional[int],
    ecosystem: EcosystemId,
    protocol: str,
    items_by_id: dict[int, SolutionItem],
) -> list[ValidationError]:
    """
    Verify that a hub_solution_item_id points to a real hub item that
    supports `protocol` in `ecosystem`.
    """
    errors: list[ValidationError] = []

    if hub_item_id is None:
        errors.append(ValidationError(
            code="MISSING_HUB_REFERENCE",
            message=(
                f"Item {item_id} ({leg_label} leg): protocol {protocol!r} "
                f"requires a hub but hub_solution_item_id is None"
            ),
        ))
        return errors

    hub_item = items_by_id.get(hub_item_id)
    if hub_item is None:
        errors.append(ValidationError(
            code="HUB_ITEM_NOT_FOUND",
            message=(
                f"Item {item_id} ({leg_label} leg): "
                f"hub_solution_item_id {hub_item_id} not found in solution"
            ),
        ))
        return errors

    if hub_item.device.device_type != "smart_hub":
        errors.append(ValidationError(
            code="HUB_ITEM_WRONG_TYPE",
            message=(
                f"Item {item_id} ({leg_label} leg): "
                f"referenced item {hub_item_id} is not a smart_hub "
                f"(type: {hub_item.device.device_type!r})"
            ),
        ))
        return errors

    # hub must have direct_compat for (ecosystem, protocol)
    hub = hub_item.device
    has_compat = any(
        dc.ecosystem == ecosystem and dc.protocol == protocol
        for dc in hub.direct_compat
    )
    if not has_compat:
        errors.append(ValidationError(
            code="HUB_MISSING_PROTOCOL",
            message=(
                f"Item {item_id} ({leg_label} leg): hub item {hub_item_id} "
                f"(device {hub.device_id}) has no direct_compat for "
                f"({ecosystem}, {protocol})"
            ),
        ))

    return errors


def _check_no_dead_hubs(
    hub_items: list[SolutionItem],
    device_items: list[SolutionItem],
) -> list[ValidationError]:
    """
    Every hub item must be referenced by at least one device's connection
    plan (either direct leg or bridge source leg) via hub_solution_item_id.
    A hub that nobody references is dead weight — a constraint violation.
    """
    referenced: set[int] = set()
    for item in device_items:
        plan = item.connection
        if plan.connection_direct.hub_solution_item_id is not None:
            referenced.add(plan.connection_direct.hub_solution_item_id)
        if (plan.connection_final is not None
                and plan.connection_final.hub_solution_item_id is not None):
            referenced.add(plan.connection_final.hub_solution_item_id)

    errors: list[ValidationError] = []
    for hub_item in hub_items:
        if hub_item.id not in referenced:
            errors.append(ValidationError(
                code="DEAD_HUB",
                message=(
                    f"Hub item {hub_item.id} (device {hub_item.device.device_id}, "
                    f"{hub_item.device.brand} {hub_item.device.model}) "
                    f"is not referenced by any device connection plan"
                ),
            ))
    return errors


def _check_ecosystem_filters(
    req: DeviceSelectionRequest,
    sol: ParetoPoint,
) -> list[ValidationError]:
    errors: list[ValidationError] = []

    allowed = req.include_ecosystems | {req.main_ecosystem}

    for item in sol.items:
        plan = item.connection
        for eco in _plan_ecosystems(plan):
            if req.include_ecosystems and eco not in allowed:
                errors.append(ValidationError(
                    code="ECO_NOT_INCLUDED",
                    message=(
                        f"Item {item.id}: ecosystem {eco!r} not in "
                        f"include_ecosystems {set(req.include_ecosystems)}"
                    ),
                ))
            if eco in req.exclude_ecosystems:
                errors.append(ValidationError(
                    code="ECO_EXCLUDED",
                    message=(
                        f"Item {item.id}: ecosystem {eco!r} is in "
                        f"exclude_ecosystems"
                    ),
                ))

    return errors


def _plan_ecosystems(plan: ConnectionPlan) -> list[EcosystemId]:
    ecos = [plan.connection_direct.ecosystem]
    if plan.connection_final is not None:
        ecos.append(plan.connection_final.ecosystem)
    return ecos


def _find_requirement(
    item: SolutionItem,
    req: DeviceSelectionRequest,
) -> Optional[DeviceRequirement]:
    if item.requirement_id is None:
        return None
    for r in req.requirements:
        if r.requirement_id == item.requirement_id:
            return r
    return None
