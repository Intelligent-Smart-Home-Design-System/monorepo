from __future__ import annotations

from collections import Counter, defaultdict
from typing import Any

from quality_calculator.evaluation.ground_truth import LABELS, label_device
from quality_calculator.evaluation.evaluate import build_device_record
from quality_calculator.evaluation.metrics import kendall_tau
from quality_calculator.evaluator import QualityEvaluator


def _assign_by_marginals(ns_values: list[float], gt_counts: list[int]) -> list[int]:
    """
    Раскладывает N(S) на 3 класса так, чтобы размеры классов совпали с эталонными
    (gt_counts = [n_bad, n_good, n_excellent]). Самые низкие N(S) -> bad и т.д.
    Это делает сравнение независимым от абсолютной шкалы N(S): проверяем, верно ли
    стратегия УПОРЯДОЧИВАЕТ устройства по тирам, а не попадает ли в произвольный порог.
    """
    order = sorted(range(len(ns_values)), key=lambda i: ns_values[i])
    pred = [0] * len(ns_values)
    cursor = 0
    for tier, cnt in enumerate(gt_counts):
        for _ in range(cnt):
            pred[order[cursor]] = tier
            cursor += 1
    return pred


def _macro_f1(gt: list[int], pred: list[int]) -> tuple[float, dict[str, dict[str, float]]]:
    per_class: dict[str, dict[str, float]] = {}
    f1s = []
    for tier, name in enumerate(LABELS):
        tp = sum(1 for g, p in zip(gt, pred) if g == tier and p == tier)
        fp = sum(1 for g, p in zip(gt, pred) if g != tier and p == tier)
        fn = sum(1 for g, p in zip(gt, pred) if g == tier and p != tier)
        support = sum(1 for g in gt if g == tier)
        prec = tp / (tp + fp) if (tp + fp) else 0.0
        rec = tp / (tp + fn) if (tp + fn) else 0.0
        f1 = 2 * prec * rec / (prec + rec) if (prec + rec) else 0.0
        per_class[name] = {"precision": round(prec, 3), "recall": round(rec, 3),
                           "f1": round(f1, 3), "support": support}
        if support > 0:  # macro по присутствующим в эталоне классам
            f1s.append(f1)
    macro = sum(f1s) / len(f1s) if f1s else 0.0
    return macro, per_class


def evaluate_spec_strategy(
    catalog: dict[str, Any],
    tech_schema: dict[str, Any],
    spec_strategy: dict[str, Any],
    rubric: dict[str, Any],
    strategy_name: str,
    min_n: int = 10,
) -> dict[str, Any]:
    """
    Тестирует ОДНУ спек-стратегию против эталона по N(S) (не по Q!).
    Возвращает per-category метрики + общий взвешенный macro-F1.
    """
    # Эвалюатор только ради N(S): веса компонентов не важны (берём specs-only).
    evaluator = QualityEvaluator(tech_schema, spec_strategy, weights=(0.0, 1.0, 0.0))

    by_cat_ns: dict[str, list[float]] = defaultdict(list)
    by_cat_gt: dict[str, list[int]] = defaultdict(list)
    labeled_total = 0

    for device in catalog["devices"]:
        record = build_device_record(device)
        cat = record["category"]
        label, _n_axes = label_device(cat, record["specs"], rubric)
        if label is None:
            continue
        ns = evaluator.eval_specs(evaluator.traits_for_type(cat), record["specs"])
        if ns is None:
            continue  # эталон есть, но стратегия не смогла оценить спеки — вне пары
        by_cat_ns[cat].append(ns)
        by_cat_gt[cat].append(LABELS.index(label))
        labeled_total += 1

    per_category = []
    weighted_f1_num = 0.0
    weighted_f1_den = 0
    for cat in sorted(by_cat_ns):
        ns = by_cat_ns[cat]
        gt = by_cat_gt[cat]
        n = len(ns)
        classes_present = len(set(gt))
        entry: dict[str, Any] = {
            "category": cat,
            "n": n,
            "gt_distribution": {LABELS[t]: c for t, c in sorted(Counter(gt).items())},
        }
        # macro-F1 требует >=2 классов в эталоне и достаточного n
        if n >= min_n and classes_present >= 2:
            gt_counts = [sum(1 for g in gt if g == t) for t in range(3)]
            pred = _assign_by_marginals(ns, gt_counts)
            macro, per_class = _macro_f1(gt, pred)
            tau = kendall_tau(ns, [float(g) for g in gt])
            accuracy = sum(1 for g, p in zip(gt, pred) if g == p) / n
            entry.update({
                "macro_f1": round(macro, 3),
                "accuracy": round(accuracy, 3),
                "kendall_tau_vs_gt": None if tau is None else round(tau, 3),
                "per_class": per_class,
                "evaluated": True,
            })
            weighted_f1_num += macro * n
            weighted_f1_den += n
        else:
            reason = "too_few_devices" if n < min_n else "single_gt_class"
            entry.update({"evaluated": False, "reason": reason})
        per_category.append(entry)

    overall = round(weighted_f1_num / weighted_f1_den, 3) if weighted_f1_den else None
    return {
        "strategy": strategy_name,
        "labeled_devices": labeled_total,
        "evaluated_devices": weighted_f1_den,
        "weighted_macro_f1": overall,
        "per_category": per_category,
    }
