"""
Готовит CSV-шаблоны для ручной разметки ground truth.

Делает два файла:
  spec_labels_template.csv     — разметка КАЧЕСТВА ХАРАКТЕРИСТИК (для стадии 1);
  overall_labels_template.csv  — разметка ОБЩЕГО КАЧЕСТВА устройства (для стадии 2).

В каждом — выборка устройств по категориям с их спеками и рейтингом, чтобы было
на что опираться (можно сверять с обзорами/«топ-10» из интернета). Колонку `label`
заполняешь вручную значениями: bad / good / excellent. Пустые строки игнорируются
при обучении.

Чтобы выборка не состояла из одних «хороших», устройства в каждой категории берутся
с трёх концов по рейтингу (низ / середина / верх) — это лишь подсказка, финальную
метку ставит человек.

Запуск:  uv run python scripts/make_labeling_template.py --per-category 6
"""
from __future__ import annotations

import argparse
import csv
import json
import os
from collections import defaultdict


def aggregate_rating(dev: dict) -> tuple[float | None, int]:
    listings = dev.get("listings") or []
    total, weighted = 0, 0.0
    for ls in listings:
        c = ls.get("review_count") or 0
        r = ls.get("rating")
        if r is None or c <= 0:
            continue
        total += c
        weighted += r * c
    return (weighted / total if total else None), total


def pick_spread(devs: list[dict], k: int) -> list[dict]:
    """Берёт k устройств, равномерно растянутых по рейтингу (низ..верх)."""
    rated = [(aggregate_rating(d)[0], d) for d in devs]
    rated = [(r, d) for r, d in rated if r is not None]
    rated.sort(key=lambda x: x[0])
    if len(rated) <= k:
        return [d for _, d in rated]
    idx = [round(i * (len(rated) - 1) / (k - 1)) for i in range(k)]
    return [rated[i][1] for i in idx]


def main() -> int:
    here = os.path.dirname(os.path.abspath(__file__))
    parser = argparse.ArgumentParser()
    parser.add_argument("--catalog", default=os.path.join(here, "..", "evaluation", "catalog.json"))
    parser.add_argument("--out-dir", default=os.path.join(here, "..", "evaluation"))
    parser.add_argument("--per-category", type=int, default=6, help="Сколько устройств на категорию.")
    args = parser.parse_args()

    catalog = json.load(open(args.catalog, encoding="utf-8"))
    by_cat: dict[str, list[dict]] = defaultdict(list)
    for d in catalog["devices"]:
        by_cat[d["category"]].append(d)

    rows = []
    for cat in sorted(by_cat):
        for dev in pick_spread(by_cat[cat], args.per_category):
            rating, count = aggregate_rating(dev)
            attrs = dev.get("device_attributes") or {}
            specs = {k: v for k, v in attrs.items() if k not in ("name", "brand", "model", "type")}
            rows.append({
                "device_id": dev.get("id"),
                "category": cat,
                "brand": dev.get("brand"),
                "model": dev.get("model"),
                "rating": None if rating is None else round(rating, 2),
                "review_count": count,
                "specs": json.dumps(specs, ensure_ascii=False),
                "label": "",  # <- заполнить вручную: bad / good / excellent
            })

    os.makedirs(args.out_dir, exist_ok=True)
    cols = ["device_id", "category", "brand", "model", "rating", "review_count", "specs", "label"]
    for fname in ("spec_labels_template.csv", "overall_labels_template.csv"):
        path = os.path.join(args.out_dir, fname)
        with open(path, "w", newline="", encoding="utf-8") as f:
            writer = csv.DictWriter(f, fieldnames=cols)
            writer.writeheader()
            writer.writerows(rows)
        print(f"  написал {path}  ({len(rows)} устройств)")

    print("\nЗаполни колонку label (bad/good/excellent) в обоих файлах и сохрани как")
    print("spec_labels.csv и overall_labels.csv соответственно.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
