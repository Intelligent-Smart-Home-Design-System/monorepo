"""Post-LLM enrichment from silver-layer parser fields."""

from __future__ import annotations

from typing import Any

from extractor.domain.models import ListingSnapshot


def normalize_brand(brand: str) -> str:
    return brand.strip().lower()


def enrich_from_silver_layer(
    listing: ListingSnapshot,
    attributes: dict[str, Any],
) -> tuple[str, str, dict[str, Any]]:
    """
    Fill gaps in LLM output using parser-extracted fields.

    LLM remains the primary source; silver-layer values are used when the model
    left brand/model/name empty or null.
    """
    enriched = dict(attributes)

    if not enriched.get("name") and listing.name:
        enriched["name"] = listing.name

    brand = enriched.get("brand") or (normalize_brand(listing.brand) if listing.brand else None)
    if brand and not enriched.get("brand"):
        enriched["brand"] = brand

    model = enriched.get("model") or listing.model_number or "unknown"
    if listing.model_number and not enriched.get("model"):
        enriched["model"] = listing.model_number

    return brand or listing.brand, model, enriched
