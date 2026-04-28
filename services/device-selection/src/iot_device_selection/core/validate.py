from __future__ import annotations
from dataclasses import dataclass
from typing import Sequence

from iot_device_selection.core.model import DeviceSelectionRequest, ParetoPoint, ConnectionMethod
from iot_device_selection.data.catalog import Catalog

@dataclass(frozen=True)
class ValidationError:
    code: str
    message: str

def validate_solution(req: DeviceSelectionRequest, sol: ParetoPoint, catalog: Catalog) -> list[ValidationError]:
    errors: list[ValidationError] = []

    # 1) budget / total_cost
    if sol.total_cost > req.budget + 1e-9:
        errors.append(ValidationError("BUDGET", f"total_cost {sol.total_cost} > budget {req.budget}"))

    # 2) device ids exist
    for item in sol.items:
        if catalog.get_device(item.device.device_id) is None:
            errors.append(ValidationError("UNKNOWN_DEVICE", f"device_id {item.device.device_id} not in catalog"))

    # 3) type_counts satisfied (only for requested types)
    required = {tc.type_id: tc.count for tc in req.type_counts}
    got: dict[int, int] = {t: 0 for t in required}
    got_type_ids = set([item.device.type_id for item in sol.items])
    for item in sol.items:
        t = item.device.type_id
        if t in got:
            got[t] += item.quantity

    for t, need in required.items():
        if got.get(t, 0) != need:
            errors.append(ValidationError("TYPE_COUNTS", f"type {t}: got {got.get(t,0)} need {need}"))

    # 4) connection sanity
    for item in sol.items:
        conn = item.connection
        be = conn.bridge_ecosystem_id
        hub_id = conn.hub_device_id
        hub = catalog.get_device(hub_id) if hub_id is not None else None
        d = catalog.get_device(item.device.device_id)

        if conn.method == ConnectionMethod.VIA_ECOSYSTEM:
            if conn.bridge_ecosystem_id is None:
                errors.append(ValidationError("CONNECTION", "VIA_ECOSYSTEM but bridge_ecosystem_id is None"))
        if conn.bridge_ecosystem_id is not None:
            if req.include_ecosystem_ids and be not in req.include_ecosystem_ids:
                errors.append(ValidationError("INCLUDE_ECOSYSTEM", f"bridge ecosystem {be} not in allowed set"))
            if be in req.exclude_ecosystem_ids:
                errors.append(ValidationError("EXCLUDE_ECOSYSTEM", f"bridge ecosystem {be} in excluded set"))
            if be != d.bridge_ecosystem_id:
                errors.append(ValidationError("CONNECTION", f"chosen bridge ecosystem {be} != bridge ecosystem {d.bridge_ecosystem_id}"))
        
        if conn.bridge_ecosystem_id != d.bridge_ecosystem_id:
            errors.append(ValidationError("CONNECTION", f"incorrect bridge ecosystem {be}"))
        if hub_id is None:
            if d.hub_type_id is not None:
                errors.append(ValidationError("CONNECTION", f"hub not specified but hub needed"))
        else:
            if hub is None:
                errors.append(ValidationError("CONNECTION", f"hub with id {hub_id} not found"))
            else:
                if hub.type_id != d.hub_type_id:
                    errors.append(ValidationError("CONNECTION", f"chosen hub type {hub.hub_type_id} != needed hub type {d.hub_type_id}"))
                if d.hub_type_id not in got_type_ids:
                    errors.append(ValidationError("CONNECTION", f"no hubs of chosen hub type {d.hub_type_id}"))

    return errors
