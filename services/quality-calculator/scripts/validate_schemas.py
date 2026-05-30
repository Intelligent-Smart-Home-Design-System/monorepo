"""
Кросс-валидация стратегии оценки (config/evaluation_traits.json) против
канонической таксономии (shared/schemas/devices/device_types.json).

Проверяет, что:
  1. каждый трейт из evaluation_traits существует в device_types.traits;
  2. каждое поле, на которое ссылается оценка, существует в этом трейте;
  3. сумма весов взвешиваемых полей трейта равна 1.0 (бонусы не считаются).

Запуск:  uv run python scripts/validate_schemas.py
"""
import json
import os
import sys


def validate() -> int:
    current_dir = os.path.dirname(os.path.abspath(__file__))
    eval_path = os.path.abspath(os.path.join(current_dir, "..", "config", "evaluation_traits.json"))
    types_path = os.path.abspath(
        os.path.join(current_dir, "..", "..", "..", "shared", "schemas", "devices", "device_types.json")
    )

    try:
        with open(eval_path, "r", encoding="utf-8") as f:
            eval_data = json.load(f)
        with open(types_path, "r", encoding="utf-8") as f:
            types_data = json.load(f)
    except FileNotFoundError as e:
        print(f"\u274c Error: File not found - {e}")
        return 1

    central_traits = types_data.get("traits", {})
    errors: list[str] = []

    print("\U0001f50d Starting JSON schemas cross-validation...")

    for trait_name, properties in eval_data.items():
        if trait_name not in central_traits:
            errors.append(f"Trait '{trait_name}' exists in evaluation_traits.json but is missing in device_types.json")
            continue

        central_props = central_traits[trait_name].get("properties", {})
        total_weight = 0.0

        for prop_name, metrics in properties.items():
            if prop_name not in central_props:
                errors.append(f"Property '{prop_name}' (in {trait_name}) is missing in device_types.json")
            total_weight += metrics.get("weight", 0.0)

        # Трейт может состоять только из бонусов (total_weight == 0) — это допустимо.
        if total_weight > 0 and not (0.99 <= total_weight <= 1.01):
            errors.append(f"Sum of weights for '{trait_name}' is {round(total_weight, 4)}, but must be 1.0")

    if errors:
        print("\n\u274c VALIDATION FAILED. Errors found:")
        for err in errors:
            print(f"  - {err}")
        return 1

    print("\n\u2705 Validation passed: traits and fields are synchronized, weights are correct.")
    return 0


if __name__ == "__main__":
    sys.exit(validate())
