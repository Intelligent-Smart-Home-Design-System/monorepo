from __future__ import annotations

import math
from dataclasses import dataclass, field
from typing import Any


def _ranks(values: list[float]) -> list[float]:
    """Средние ранги с обработкой связок (ties)."""
    order = sorted(range(len(values)), key=lambda i: values[i])
    ranks = [0.0] * len(values)
    i = 0
    while i < len(order):
        j = i
        while j + 1 < len(order) and values[order[j + 1]] == values[order[i]]:
            j += 1
        avg = (i + j) / 2.0 + 1.0  # ранги с 1
        for k in range(i, j + 1):
            ranks[order[k]] = avg
        i = j + 1
    return ranks


def _pearson(x: list[float], y: list[float]) -> float | None:
    n = len(x)
    if n < 2:
        return None
    mx, my = sum(x) / n, sum(y) / n
    cov = sum((a - mx) * (b - my) for a, b in zip(x, y))
    vx = math.sqrt(sum((a - mx) ** 2 for a in x))
    vy = math.sqrt(sum((b - my) ** 2 for b in y))
    if vx == 0 or vy == 0:
        return None
    return cov / (vx * vy)


def spearman(x: list[float], y: list[float]) -> float | None:
    """Корреляция рангов Спирмена."""
    if len(x) < 2:
        return None
    return _pearson(_ranks(x), _ranks(y))


def kendall_tau(x: list[float], y: list[float]) -> float | None:
    """Kendall tau-b (с поправкой на связки)."""
    n = len(x)
    if n < 2:
        return None
    concordant = discordant = tx = ty = 0
    for i in range(n):
        for j in range(i + 1, n):
            dx = x[i] - x[j]
            dy = y[i] - y[j]
            if dx == 0 and dy == 0:
                continue
            if dx == 0:
                tx += 1
            elif dy == 0:
                ty += 1
            elif (dx > 0) == (dy > 0):
                concordant += 1
            else:
                discordant += 1
    n0 = concordant + discordant + tx
    n1 = concordant + discordant + ty
    denom = math.sqrt(n0 * n1)
    if denom == 0:
        return None
    return (concordant - discordant) / denom


def precision_at_k(pred: list[float], truth: list[float], k: int) -> float | None:
    """Доля пересечения top-k по предсказанию и top-k по ground truth."""
    n = len(pred)
    if n < k or k <= 0:
        return None
    top_pred = set(sorted(range(n), key=lambda i: pred[i], reverse=True)[:k])
    top_truth = set(sorted(range(n), key=lambda i: truth[i], reverse=True)[:k])
    return len(top_pred & top_truth) / k


def stdev(values: list[float]) -> float:
    n = len(values)
    if n < 2:
        return 0.0
    m = sum(values) / n
    return math.sqrt(sum((v - m) ** 2 for v in values) / (n - 1))


@dataclass
class CategoryMetrics:
    category: str
    n: int
    n_scored: int                 # сколько устройств получили Q
    specs_coverage: float         # доля с непустым N(S)
    spearman: float | None
    kendall: float | None
    precision_at_10: float | None
    discrimination: float         # разброс Q (stdev)


@dataclass
class StrategyMetrics:
    strategy: str
    weights: dict[str, float]
    reputation_mode: str
    per_category: list[CategoryMetrics] = field(default_factory=list)

    def summary(self, min_n: int = 10) -> dict[str, Any]:
        """Агрегаты по категориям с n >= min_n, взвешенные по числу устройств."""
        eligible = [c for c in self.per_category if c.n >= min_n]

        def wmean(attr: str) -> float | None:
            pairs = [(getattr(c, attr), c.n) for c in eligible if getattr(c, attr) is not None]
            wsum = sum(n for _, n in pairs)
            return sum(v * n for v, n in pairs) / wsum if wsum else None

        total = sum(c.n for c in self.per_category)
        scored = sum(c.n_scored for c in self.per_category)
        return {
            "strategy": self.strategy,
            "weights": self.weights,
            "reputation_mode": self.reputation_mode,
            "total_devices": total,
            "scored_devices": scored,
            "weighted_spearman": wmean("spearman"),
            "weighted_kendall": wmean("kendall"),
            "weighted_precision_at_10": wmean("precision_at_10"),
            "weighted_specs_coverage": wmean("specs_coverage"),
            "weighted_discrimination": wmean("discrimination"),
            "categories_considered": [c.category for c in eligible],
        }


def compute_category_metrics(
    category: str,
    q_values: list[float | None],
    gt_values: list[float],
    specs_present: list[bool],
) -> CategoryMetrics:
    n = len(q_values)
    paired = [(q, gt) for q, gt in zip(q_values, gt_values) if q is not None]
    qs = [q for q, _ in paired]
    gts = [gt for _, gt in paired]
    return CategoryMetrics(
        category=category,
        n=n,
        n_scored=len(paired),
        specs_coverage=sum(specs_present) / n if n else 0.0,
        spearman=spearman(qs, gts),
        kendall=kendall_tau(qs, gts),
        precision_at_10=precision_at_k(qs, gts, 10),
        discrimination=stdev(qs),
    )
