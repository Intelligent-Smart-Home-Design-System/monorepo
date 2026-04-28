from __future__ import annotations

import json
from pathlib import Path
from typing import Sequence

from iot_device_selection.core.model import ParetoPoint
from iot_device_selection.data.loader import get_type_registry, get_eco_registry


def _device_lookup(enriched_devices: list[dict]) -> dict[int, dict]:
    return {d["id"]: d for d in enriched_devices}


def export_result(
    points: Sequence[ParetoPoint],
    enriched_devices: list[dict],
    output_path: str | Path,
) -> None:
    lookup = _device_lookup(enriched_devices)
    type_reg = {v: k for k, v in get_type_registry().items()}   # id -> name
    eco_reg  = {v: k for k, v in get_eco_registry().items()}

    out_points = []
    for p in points:
        items_out = []
        for item in p.items:
            dev_id  = item.device.device_id
            raw_dev = lookup.get(dev_id, {})

            conn = {
                "method": item.connection.method.value,
                "bridge_ecosystem": eco_reg.get(item.connection.bridge_ecosystem_id)
                    if item.connection.bridge_ecosystem_id else None,
            }

            # pick best listing for display (most reviews)
            listings = raw_dev.get("listings", [])
            best_listing = max(listings, key=lambda l: l.get("review_count") or 0, default=None)

            items_out.append({
                "device_id":    dev_id,
                "brand":        raw_dev.get("brand"),
                "model":        raw_dev.get("model"),
                "category":     type_reg.get(item.device.type_id, str(item.device.type_id)),
                "quantity":     item.quantity,
                "price_each":   item.device.price,
                "quality":      item.device.quality,
                "image_url":    raw_dev.get("image_url"),
                "connection":   conn,
                "listings":     listings,
                "best_listing": best_listing,
                "device_attributes": raw_dev.get("device_attributes", {}),
                "direct_compatibility":  raw_dev.get("direct_compatibility", []),
                "bridge_compatibility":  raw_dev.get("bridge_compatibility", []),
            })

        out_points.append({
            "total_cost":     p.total_cost,
            "avg_quality":    p.avg_quality,
            "num_ecosystems": p.num_ecosystems,
            "num_hubs":       p.num_hubs,
            "items":          items_out,
        })

    Path(output_path).write_text(
        json.dumps({"pareto_points": out_points}, ensure_ascii=False, indent=2),
        encoding="utf-8",
    )
    print(f"Wrote {len(out_points)} Pareto points → {output_path}")
