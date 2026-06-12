from __future__ import annotations

from typing import Any

LABELS = ["bad", "good", "excellent"]  # индекс = тир 0/1/2


def _round_half_up(x: float) -> int:
    # обычный round() в Python банковский (round(1.5)=2, round(0.5)=0) — нам нужен предсказуемый half-up
    import math
    return int(math.floor(x + 0.5))


def _tier_numeric(value: Any, thresholds: list[float], inverse: bool) -> int | None:
    if not isinstance(value, (int, float)) or isinstance(value, bool):
        return None
    t1, t2 = thresholds
    if value < t1:
        tier = 0
    elif value < t2:
        tier = 1
    else:
        tier = 2
    return (2 - tier) if inverse else tier


def _tier_ordinal(value: Any, mapping: dict[str, int]) -> int | None:
    if not isinstance(value, str):
        return None
    return mapping.get(value)


def _tier_count(value: Any, thresholds: list[float]) -> int | None:
    if not isinstance(value, list):
        return None
    n = len(value)
    t1, t2 = thresholds
    if n < t1:
        return 0
    if n < t2:
        return 1
    return 2


def axis_tier(axis: dict[str, Any], specs: dict[str, Any]) -> int | None:
    value = specs.get(axis["spec"])
    if value is None:
        return None
    kind = axis["kind"]
    if kind == "numeric":
        return _tier_numeric(value, axis["thresholds"], axis.get("inverse", False))
    if kind == "ordinal":
        return _tier_ordinal(value, axis["map"])
    if kind == "count":
        return _tier_count(value, axis["thresholds"])
    return None


def label_device(category: str, specs: dict[str, Any], rubric: dict[str, Any]) -> tuple[str | None, int]:
    """
    Возвращает (label|None, n_axes_with_data).
    None — у устройства нет данных ни по одной оси эталона своей категории.
    """
    cat_rubric = rubric.get("categories", {}).get(category)
    if not cat_rubric:
        return None, 0

    tiers = [t for ax in cat_rubric["axes"] if (t := axis_tier(ax, specs)) is not None]
    if not tiers:
        return None, 0

    combine = cat_rubric.get("combine", "mean")
    if combine == "min":
        tier = min(tiers)
    elif combine == "max":
        tier = max(tiers)
    else:  # mean
        tier = _round_half_up(sum(tiers) / len(tiers))

    tier = max(0, min(2, tier))
    return LABELS[tier], len(tiers)
