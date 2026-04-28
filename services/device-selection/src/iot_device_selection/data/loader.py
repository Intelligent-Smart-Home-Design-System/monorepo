from __future__ import annotations

import json
from pathlib import Path
from typing import Any, Optional

from iot_device_selection.core.model import Device, DeviceTypeId, EcosystemId
from iot_device_selection.data.catalog import InMemoryCatalog

_type_registry: dict[str, int] = {}
_eco_registry:  dict[str, int] = {}

def type_id(name: str) -> int:
    if name not in _type_registry:
        _type_registry[name] = len(_type_registry) + 1
    return _type_registry[name]

def eco_id(name: str) -> int:
    if name not in _eco_registry:
        _eco_registry[name] = len(_eco_registry) + 1
    return _eco_registry[name]

def get_type_registry() -> dict[str, int]:
    return dict(_type_registry)

def get_eco_registry() -> dict[str, int]:
    return dict(_eco_registry)


_hub_type_registry: dict[tuple[int, str], int] = {}

HUB_PROTOCOLS = {"zigbee"}   # extend if you add z-wave etc.

def _hub_type_id_for(ecosystem: str, protocol: str) -> int:
    key = (eco_id(ecosystem), protocol)
    if key not in _hub_type_registry:
        _hub_type_registry[key] = 10_000 + len(_hub_type_registry)
    return _hub_type_registry[key]

def get_hub_type_registry() -> dict[tuple[int, str], int]:
    return dict(_hub_type_registry)


# ── Bayesian quality score ────────────────────────────────────────────────────

BAYES_M = 10      # equivalent number of "prior" votes
BAYES_C = 3.5     # prior mean rating (out of 5)

def _bayes_quality(total_reviews: int, weighted_rating: float) -> float:
    """
    weighted_rating = sum(rating_i * review_count_i) / total_reviews  (raw mean)
    We pass in the already-aggregated totals.
    """
    if total_reviews == 0:
        return BAYES_C / 5.0
    raw_mean = weighted_rating / total_reviews
    bayes = (total_reviews * raw_mean + BAYES_M * BAYES_C) / (total_reviews + BAYES_M)
    return round(bayes / 5.0, 4)   # normalise to [0, 1]


# ── main loader ───────────────────────────────────────────────────────────────

def load_catalog(path: str | Path) -> tuple[InMemoryCatalog, list[dict]]:
    """
    Returns (InMemoryCatalog, raw_devices_list).
    raw_devices_list is the original JSON list, enriched with computed fields
    (quality, hub_type_ids, type_id, ecosystem ids) so we can use it later
    for the output JSON / HTML.
    """
    raw = json.loads(Path(path).read_text(encoding="utf-8"))
    raw_devices: list[dict] = raw["devices"]

    devices_by_type: dict[int, list[Device]] = {}
    enriched: list[dict] = []

    for d in raw_devices:
        category: str = d.get("category", "unknown")
        tid = type_id(category)

        # ── price ────────────────────────────────────────────────────────────
        price = d.get("median_price") or 0.0

        # ── quality ──────────────────────────────────────────────────────────
        total_reviews = 0
        weighted_sum  = 0.0
        for lst in d.get("listings", []):
            rc = lst.get("review_count") or 0
            rt = lst.get("rating")
            if rc and rt is not None:
                total_reviews += rc
                weighted_sum  += rt * rc
        quality = _bayes_quality(total_reviews, weighted_sum)

        attrs       = d.get("device_attributes", {})
        ecosystems  = attrs.get("ecosystem") or []
        protocols   = attrs.get("protocol")  or []

        bridge_eco_id: Optional[int] = None
        bridges = d.get("bridge_compatibility", [])
        if bridges:
            bridge_eco_id = eco_id(bridges[0]["ecosystem_source"])

        computed_hub_type_ids: list[int] = []
        if category == "smart_hub":
            for eco in ecosystems:
                for proto in protocols:
                    if proto in HUB_PROTOCOLS:
                        computed_hub_type_ids.append(
                            _hub_type_id_for(eco, proto)
                        )

        hub_tid: Optional[int] = computed_hub_type_ids[0] if computed_hub_type_ids else None

        req_hub_type_id: Optional[int] = None

        if category != "smart_hub":
            direct_compat = d.get("direct_compatibility", [])
            # find ecosystems this device reaches via zigbee directly
            zigbee_ecosystems = [
                c["ecosystem"]
                for c in direct_compat
                if c.get("protocol") == "zigbee"
            ]

            if zigbee_ecosystems:
                # use the first one (caller can filter by main_ecosystem later)
                req_hub_type_id = _hub_type_id_for(zigbee_ecosystems[0], "zigbee")
                #print(req_hub_type_id)

        if category == "smart_hub":
            tid = hub_tid
        if req_hub_type_id == None and category != "smart_hub":
            continue
        device = Device(
            device_id         = d["id"],
            type_id           = tid,
            price             = float(price),
            quality           = quality,
            bridge_ecosystem_id = bridge_eco_id,
            hub_type_id       = req_hub_type_id,
        )

        devices_by_type.setdefault(tid, []).append(device)

        enriched.append({
            **d,
            "_type_id":          tid,
            "_quality":          quality,
            "_total_reviews":    total_reviews,
            "_bridge_eco_id":    bridge_eco_id,
            "_hub_type_ids":     computed_hub_type_ids,
            "_ecosystem_ids":    [eco_id(e) for e in ecosystems],
        })

    catalog = InMemoryCatalog(devices_by_type)
    return catalog, enriched
