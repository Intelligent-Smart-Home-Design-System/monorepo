import json
import sys
from pathlib import Path
from typing import Any, Dict, List
import argparse

def load_taxonomy(file_path: Path) -> Dict[str, Any]:
    try:
        with open(file_path, "r", encoding="utf-8-sig") as file:
            return json.load(file)
    except json.JSONDecodeError as e:
        print(f"ERROR: Invalid JSON in {file_path}: {e}")
        sys.exit(1)
    except FileNotFoundError:
        print(f"ERROR: File not found: {file_path}")
        sys.exit(1)

def generate_device_schema(
        device_name: str,
        device_def: Dict[str, Any],
        traits: Dict[str, Any]
) -> Dict[str, Any]:
    properties = {}
    required: List[str] = []

    for trait_name in device_def.get("traits", []):
        trait = traits.get(trait_name)
        if trait is None:
            print(f"ERROR: Trait {trait_name} not found in device {device_name}")
            sys.exit(1)

        props = trait.get("properties", {})
        for prop_name, prop_schema in props.items():
            if prop_name in properties:
                print(f"WARNING: property {prop_name} already exists in device {device_name}, overwriting")
            properties[prop_name] = prop_schema

        for req in trait.get("required", []):
            if req not in required:
                required.append(req)

    extra = device_def.get("extra_schema", {})
    for prop_name, prop_schema in extra.get("properties", {}).items():
        if prop_name in properties:
            print(f"WARNING: property {prop_name} already exists in device {device_name}, overwriting")
        properties[prop_name] = prop_schema

    for req in extra.get("required", []):
        if req not in required:
            required.append(req)

    schema = {
        "$schema": "http://json-schema.org/draft-07/schema#",
        "type": "object",
        "properties": properties,
        "required": required,
        "additionalProperties": False
    }
    return schema

def save_schema(schema: Dict[str, Any], output_path: Path) -> None:
    output_path.parent.mkdir(parents=True, exist_ok=True)
    with open(output_path, "w", encoding="utf-8") as f:
        json.dump(schema, f, indent=2, ensure_ascii=False)

def main():
    parser = argparse.ArgumentParser(description="Generate device JSON schemas from taxonomy")
    parser.add_argument("--input", "-i", default="device_types.json",
                        help="Path to the taxonomy JSON file (default: device_types.json)")
    parser.add_argument("--output", "-o", default="schemas",
                        help="Output directory for generated schemas (default: schemas)")
    args = parser.parse_args()

    input_path = Path(args.input)
    output_dir = Path(args.output)

    taxonomy = load_taxonomy(input_path)
    traits = taxonomy.get("traits", {})
    types = taxonomy.get("types", {})

    if not traits:
        print("ERROR: No 'traits' object found in taxonomy.")
        sys.exit(1)
    if not types:
        print("ERROR: No 'types' object found in taxonomy.")
        sys.exit(1)

    for device_name, device_def in types.items():
        schema = generate_device_schema(device_name, device_def, traits)
        output_file = output_dir / f"{device_name}.schema.json"
        save_schema(schema, output_file)

if __name__ == "__main__":
    main()