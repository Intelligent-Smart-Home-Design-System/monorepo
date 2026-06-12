"""
Обучение весов модели качества методом градиентного спуска (MSE к {0, 0.5, 1}).

Две стадии, как и договаривались:
  Стадия 1 — веса ВНУТРИ N(S): для каждого обучаемого трейта (>1 весового поля)
             подбираем веса полей так, чтобы N(S) приближал ручную спек-разметку.
  Стадия 2 — веса компонентов (w_R, w_S, w_E): фиксируем стратегию из стадии 1,
             подбираем смешивание трёх сигналов под ручную общую разметку.

Модель линейна по весам, данных мало, параметров <= 3, поэтому:
  * градиент считаем конечными разностями (просто и без вывода производных);
  * ограничение «веса >= 0 и в сумме = 1» держим параметризацией через softmax.

Разметка — словарь {device_id: "bad"|"good"|"excellent"}.
"""
from __future__ import annotations

import json
from typing import Any

import numpy as np

from quality_calculator.evaluator import QualityEvaluator
from quality_calculator.evaluation.evaluate import build_device_record

LABEL_TO_TARGET = {"bad": 0.0, "good": 0.5, "excellent": 1.0}
EPS = 1e-5  # шаг конечной разности


def softmax(theta: np.ndarray) -> np.ndarray:
    z = theta - theta.max()
    e = np.exp(z)
    return e / e.sum()


# ---------- вклад одного поля в [0..1] (не зависит от веса) ----------

def field_contribution(value: Any, rule: dict, schema_prop: dict) -> float | None:
    """Нормированный вклад поля: число -> normalize, шкала -> map, массив -> доля от count_max."""
    if value is None:
        return None
    if "scale" in rule:
        return rule["scale"].get(value)
    if "count_max" in rule and isinstance(value, list):
        cm = rule["count_max"]
        return min(1.0, len(value) / cm) if cm else None
    lo, hi = schema_prop.get("minimum"), schema_prop.get("maximum")
    if lo is None or hi is None:
        return None
    return QualityEvaluator.normalize(value, lo, hi, rule.get("inverse", False))


# ---------- сбор обучающей матрицы для одного трейта ----------

def collect_trait_matrix(
    devices: list[dict], labels: dict[int, str], tech_schema: dict, strategy: dict, trait: str
) -> tuple[np.ndarray, np.ndarray, list[str]]:
    """
    Возвращает (C, y, fields):
      C — матрица вкладов полей [n устройств x k полей], np.nan где поля нет;
      y — цели в {0,0.5,1};
      fields — порядок весовых полей трейта.
    Берём устройства, у которых трейт присутствует (по таксономии), есть метка
    и хотя бы одно поле трейта заполнено.
    """
    trait_rules = strategy[trait]
    fields = [f for f, r in trait_rules.items() if "weight" in r]  # обучаем только весовые поля
    schema_props = tech_schema["traits"][trait]["properties"]
    types = tech_schema["types"]

    rows, ys = [], []
    for dev in devices:
        dev_id = dev.get("id")
        if dev_id not in labels:
            continue
        cat = dev.get("category")
        if trait not in types.get(cat, {}).get("traits", []):
            continue
        specs = build_device_record(dev)["specs"]
        contribs = [field_contribution(specs.get(f), trait_rules[f], schema_props.get(f, {})) for f in fields]
        if all(c is None for c in contribs):
            continue
        rows.append([np.nan if c is None else c for c in contribs])
        ys.append(LABEL_TO_TARGET[labels[dev_id]])

    return np.array(rows, dtype=float), np.array(ys, dtype=float), fields


# ---------- прямой проход и потеря ----------

def predict_ns(C: np.ndarray, mask: np.ndarray, w: np.ndarray) -> np.ndarray:
    """N(S) трейта как взвешенное среднее по присутствующим полям (как в эвалюаторе)."""
    num = np.where(mask, C, 0.0) @ w
    den = mask @ w
    return num / den


def mse(pred: np.ndarray, y: np.ndarray) -> float:
    return float(np.mean((pred - y) ** 2))


# ---------- градиентный спуск ----------

def fit_weights(C: np.ndarray, y: np.ndarray, n_iter: int = 500, lr: float = 0.5) -> tuple[np.ndarray, list[float]]:
    """Подбор весов (softmax) под цели y по MSE. Возвращает (веса, история потерь)."""
    mask = ~np.isnan(C)
    k = C.shape[1]
    theta = np.zeros(k)

    def loss(th: np.ndarray) -> float:
        return mse(predict_ns(C, mask, softmax(th)), y)

    history = [loss(theta)]
    for _ in range(n_iter):
        grad = np.zeros(k)
        for j in range(k):  # конечная разность по каждому параметру
            tp, tm = theta.copy(), theta.copy()
            tp[j] += EPS
            tm[j] -= EPS
            grad[j] = (loss(tp) - loss(tm)) / (2 * EPS)
        theta -= lr * grad
        history.append(loss(theta))
    return softmax(theta), history


# ---------- стадия 2: веса компонентов ----------

def collect_component_matrix(
    devices: list[dict], labels: dict[int, str], evaluator: QualityEvaluator
) -> tuple[np.ndarray, np.ndarray]:
    """Матрица [N(R), N(S), E] для размеченных устройств с тремя доступными компонентами."""
    rows, ys = [], []
    for dev in devices:
        dev_id = dev.get("id")
        if dev_id not in labels:
            continue
        res = evaluator.evaluate_device(build_device_record(dev))
        comps = [res["N_R"], res["N_S"], res["E"]]
        if any(c is None for c in comps):
            continue  # для обучения смешивания нужны все три сигнала
        rows.append(comps)
        ys.append(LABEL_TO_TARGET[labels[dev_id]])
    return np.array(rows, dtype=float), np.array(ys, dtype=float)


def fit_component_weights(F: np.ndarray, y: np.ndarray, n_iter: int = 500, lr: float = 0.5):
    """Подбор (w_R, w_S, w_E) по MSE. F — матрица [n x 3] компонентов."""
    theta = np.zeros(3)

    def loss(th: np.ndarray) -> float:
        return mse(F @ softmax(th), y)

    history = [loss(theta)]
    for _ in range(n_iter):
        grad = np.zeros(3)
        for j in range(3):
            tp, tm = theta.copy(), theta.copy()
            tp[j] += EPS
            tm[j] -= EPS
            grad[j] = (loss(tp) - loss(tm)) / (2 * EPS)
        theta -= lr * grad
        history.append(loss(theta))
    return softmax(theta), history


# ---------- сборка: обучить и вернуть готовую стратегию + веса компонентов ----------

def train(
    devices: list[dict],
    spec_labels: dict[int, str],
    overall_labels: dict[int, str],
    tech_schema: dict,
    base_strategy: dict,
    n_iter: int = 500,
    lr: float = 0.5,
) -> dict[str, Any]:
    """
    Полный двухстадийный прогон. Возвращает:
      trained_strategy — обновлённый evaluation_traits (веса трейтов обучены),
      component_weights — {"reputation","specs","ecosystem"},
      report — что и на скольких объектах обучалось + динамика потерь.
    """
    import copy
    strategy = copy.deepcopy(base_strategy)
    report: dict[str, Any] = {"stage1": {}, "stage2": {}}

    # --- стадия 1: веса внутри N(S) по трейтам ---
    for trait, props in strategy.items():
        weighted = [f for f, r in props.items() if "weight" in r]
        if len(weighted) < 2:
            continue  # один весовой признак -> вес 1.0, обучать нечего
        C, y, fields = collect_trait_matrix(devices, spec_labels, tech_schema, strategy, trait)
        if len(y) < 2 * len(fields):  # совсем мало данных под число параметров -> не трогаем
            report["stage1"][trait] = {"trained": False, "reason": "too_few_labeled", "n": int(len(y))}
            continue
        w, hist = fit_weights(C, y, n_iter=n_iter, lr=lr)
        for f, wj in zip(fields, w):
            strategy[trait][f]["weight"] = round(float(wj), 3)
        report["stage1"][trait] = {
            "trained": True, "n": int(len(y)), "fields": fields,
            "weights": [round(float(x), 3) for x in w],
            "mse_before": round(hist[0], 4), "mse_after": round(hist[-1], 4),
        }

    # --- стадия 2: веса компонентов на обученной стратегии ---
    evaluator = QualityEvaluator(tech_schema, strategy, weights=(1.0, 1.0, 1.0))
    F, y = collect_component_matrix(devices, overall_labels, evaluator)
    if len(y) >= 6:
        cw, hist = fit_component_weights(F, y, n_iter=n_iter, lr=lr)
        component_weights = {"reputation": round(float(cw[0]), 3),
                             "specs": round(float(cw[1]), 3),
                             "ecosystem": round(float(cw[2]), 3)}
        report["stage2"] = {"trained": True, "n": int(len(y)),
                            "weights": component_weights,
                            "mse_before": round(hist[0], 4), "mse_after": round(hist[-1], 4)}
    else:
        component_weights = {"reputation": 0.3, "specs": 0.4, "ecosystem": 0.3}
        report["stage2"] = {"trained": False, "reason": "too_few_labeled", "n": int(len(y)),
                            "weights": component_weights}

    return {"trained_strategy": strategy, "component_weights": component_weights, "report": report}
