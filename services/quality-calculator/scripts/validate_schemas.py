import json
import os
import sys


def validate():
    current_dir = os.path.dirname(os.path.abspath(__file__))
    eval_path = os.path.join(current_dir, "evaluation_traits.json")
    types_path = os.path.abspath(
        os.path.join(current_dir, "..", "..", "..", "shared", "schemas", "devices", "device_types.json"))

    try:
        with open(eval_path, 'r', encoding='utf-8') as f:
            eval_data = json.load(f)
        with open(types_path, 'r', encoding='utf-8') as f:
            types_data = json.load(f)
    except FileNotFoundError as e:
        print(f"❌ Error: File not found - {e}")
        sys.exit(1)

    central_traits = types_data.get("traits", {})
    errors = []

    print("🔍 Starting JSON schemas cross-validation...")

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

        if total_weight > 0 and not (0.99 <= total_weight <= 1.01):
            errors.append(f"Sum of weights for '{trait_name}' is {total_weight}, but must be 1.0")

    if errors:
        print("\n❌ VALIDATION FAILED. Errors found:")
        for err in errors:
            print(f"  - {err}")
        sys.exit(1)
    else:
        print("\n✅ Validation passed successfully! Structure is synchronized, weights are correct.")


if __name__ == "__main__":
    validate()