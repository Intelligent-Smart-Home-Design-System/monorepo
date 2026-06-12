"""
Кросс-валидация стратегий оценки против канонической таксономии
(shared/schemas/devices/device_types.json).

Проверяет для config/evaluation_traits.json И каждого варианта в
config/strategies/specs/*.json, что:
  1. каждый трейт стратегии существует в device_types.traits;
  2. каждое поле, на которое ссылается оценка, существует в этом трейте;
  3. сумма весов взвешиваемых полей трейта равна 1.0 (бонусы не считаются).

Запуск:  uv run python scripts/validate_schemas.py
"""
import json
import os
import sys


def _check_strategy(label: str, eval_data: dict, central_traits: dict) -> list[str]:
    errors: list[str] = []
    for trait_name, properties in eval_data.items():
        if trait_name not in central_traits:
            errors.append(f"[{label}] trait '{trait_name}' missing in device_types.json")
            continue
        central_props = central_traits[trait_name].get("properties", {})
        total_weight = 0.0
        for prop_name, metrics in properties.items():
            if prop_name not in central_props:
                errors.append(f"[{label}] property '{prop_name}' (in {trait_name}) missing in device_types.json")
            total_weight += metrics.get("weight", 0.0)
        if total_weight > 0 and not (0.99 <= total_weight <= 1.01):
            errors.append(f"[{label}] sum of weights for '{trait_name}' is {round(total_weight, 4)}, must be 1.0")
    return errors


def validate() -> int:
    current_dir = os.path.dirname(os.path.abspath(__file__))
    config_dir = os.path.join(current_dir, "..", "config")
    eval_path = os.path.abspath(os.path.join(config_dir, "evaluation_traits.json"))
    specs_dir = os.path.abspath(os.path.join(config_dir, "strategies", "specs"))
    types_path = os.path.abspath(
        os.path.join(current_dir, "..", "..", "..", "shared", "schemas", "devices", "device_types.json")
    )

    try:
        with open(types_path, "r", encoding="utf-8") as f:
            types_data = json.load(f)
    except FileNotFoundError as e:
        print(f"\u274c Error: File not found - {e}")
        return 1

    central_traits = types_data.get("traits", {})

    # Список валидируемых стратегий: каноническая + все спек-варианты.
    targets = [("evaluation_traits.json", eval_path)]
    if os.path.isdir(specs_dir):
        for fn in sorted(os.listdir(specs_dir)):
            if fn.endswith(".json"):
                targets.append((f"specs/{fn}", os.path.join(specs_dir, fn)))

    print("\U0001f50d Starting JSON schemas cross-validation...")
    errors: list[str] = []
    for label, path in targets:
        try:
            with open(path, "r", encoding="utf-8") as f:
                data = json.load(f)
        except FileNotFoundError:
            errors.append(f"[{label}] file not found: {path}")
            continue
        errors.extend(_check_strategy(label, data, central_traits))

    if errors:
        print("\n\u274c VALIDATION FAILED. Errors found:")
        for err in errors:
            print(f"  - {err}")
        return 1

    checked = ", ".join(label for label, _ in targets)
    print(f"\n\u2705 Validation passed: {checked} — traits/fields synchronized, weights correct.")
    return 0


if __name__ == "__main__":
    sys.exit(validate())
