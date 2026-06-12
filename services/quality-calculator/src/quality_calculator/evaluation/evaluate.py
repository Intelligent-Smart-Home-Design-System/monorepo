from __future__ import annotations

from collections import defaultdict
from typing import Any

from quality_calculator.evaluator import QualityEvaluator
from quality_calculator.evaluation.metrics import (
    StrategyMetrics,
    compute_category_metrics,
)


def aggregate_reviews(device: dict[str, Any]) -> dict[str, Any]:
    """
    Сворачивает листинги одного устройства в единую репутацию:
      * count  = сумма числа отзывов по всем листингам;
      * rating = средневзвешенный по числу отзывов рейтинг.
    """
    listings = device.get("listings") or []
    total = 0
    weighted = 0.0
    for ls in listings:
        c = ls.get("review_count") or 0
        r = ls.get("rating")
        if r is None or c <= 0:
            continue
        total += c
        weighted += r * c
    rating = weighted / total if total else None
    return {"rating": rating, "count": total}


def build_device_record(device: dict[str, Any]) -> dict[str, Any]:
    attrs = device.get("device_attributes") or {}
    return {
        "id": device.get("id"),
        "name": attrs.get("name") or device.get("model"),
        "category": device.get("category"),
        "specs": attrs,
        "protocol": attrs.get("protocol") or [],
        "reviews": aggregate_reviews(device),
        "price": device.get("median_price"),
    }


def ground_truth_rating(reviews: dict[str, Any], bayes_m: int, bayes_c: float) -> float | None:
    """
    Ground truth = байесовски сглаженный средневзвешенный рейтинг пользователей.
    Это лучший доступный прокси 'воспринимаемого качества': устройство с рейтингом
    4.9 и 5 отзывами не должно обгонять 4.7 c 3000 отзывов.
    """
    rating = reviews.get("rating")
    count = reviews.get("count", 0)
    if rating is None or count <= 0:
        return None
    return (count * rating + bayes_m * bayes_c) / (count + bayes_m)


def run_strategy(
    catalog: dict[str, Any],
    evaluator: QualityEvaluator,
    strategy_name: str,
    weights: dict[str, float],
    reputation_mode: str,
) -> tuple[StrategyMetrics, list[dict[str, Any]]]:
    devices = catalog["devices"]

    by_cat_q: dict[str, list[float | None]] = defaultdict(list)
    by_cat_gt: dict[str, list[float]] = defaultdict(list)
    by_cat_specs: dict[str, list[bool]] = defaultdict(list)
    per_device: list[dict[str, Any]] = []

    for device in devices:
        record = build_device_record(device)
        gt = ground_truth_rating(record["reviews"], evaluator.bayes_m, evaluator.bayes_c)
        if gt is None:
            continue  # без отзывов нет ground truth — устройство вне оценки

        result = evaluator.evaluate_device(record)
        cat = record["category"]
        by_cat_q[cat].append(result["Q_total"])
        by_cat_gt[cat].append(gt)
        by_cat_specs[cat].append(result["N_S"] is not None)
        per_device.append({**result, "ground_truth": round(gt, 4)})

    metrics = StrategyMetrics(strategy=strategy_name, weights=weights, reputation_mode=reputation_mode)
    for cat in sorted(by_cat_q):
        metrics.per_category.append(
            compute_category_metrics(cat, by_cat_q[cat], by_cat_gt[cat], by_cat_specs[cat])
        )
    return metrics, per_device
