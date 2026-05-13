"""Export an InMemoryCatalog to the JSON format used by evaluation test cases."""
from __future__ import annotations

import json
from pathlib import Path
from typing import Any

from device_selection.core.model import Device
from device_selection.data.catalog import InMemoryCatalog


def _device_to_dict(d: Device) -> dict[str, Any]:
    return {
        "device_id": d.device_id,
        "device_type": d.device_type,
        "brand": d.brand,
        "model": d.model,
        "attributes": d.attributes,
        "price": d.price,
        "quality": d.quality,
        "source_listing_id": d.source_listing_id,
        "direct_compat": [
            {"ecosystem": dc.ecosystem, "protocol": dc.protocol}
            for dc in d.direct_compat
        ],
        "bridge_compat": [
            {
                "source_ecosystem": bc.source_ecosystem,
                "target_ecosystem": bc.target_ecosystem,
                "protocol": bc.protocol,
            }
            for bc in d.bridge_compat
        ],
    }


def export_catalog(catalog: InMemoryCatalog, output_path: Path) -> int:
    """Serialize catalog to JSON. Returns number of devices written."""
    devices = catalog._devices_by_id.values()
    payload = {"devices": [_device_to_dict(d) for d in devices]}
    output_path.write_text(json.dumps(payload, ensure_ascii=False, indent=2))
    return len(devices)
