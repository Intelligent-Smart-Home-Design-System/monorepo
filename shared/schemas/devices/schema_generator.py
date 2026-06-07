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
            print(f"ERROR: Trait '{trait_name}' not found in device '{device_name}'")
            sys.exit(1)

        for prop_name, prop_schema in trait.get("properties", {}).items():
            if prop_name in properties:
                print(f"WARNING: property '{prop_name}' already exists in device '{device_name}', overwriting")
            properties[prop_name] = prop_schema

        for req in trait.get("required", []):
            if req not in required:
                required.append(req)

    extra = device_def.get("extra_schema", {})
    for prop_name, prop_schema in extra.get("properties", {}).items():
        if prop_name in properties:
            print(f"WARNING: property '{prop_name}' already exists in device '{device_name}', overwriting")
        properties[prop_name] = prop_schema

    for req in extra.get("required", []):
        if req not in required:
            required.append(req)

    # применяем переопределения описаний полей из property_descriptions
    prop_descriptions = device_def.get("property_descriptions", {})
    for prop_name, description in prop_descriptions.items():
        if prop_name in properties:
            # копируем чтобы не мутировать оригинальный трейт
            properties[prop_name] = {**properties[prop_name], "description": description}

    schema: Dict[str, Any] = {
        "$schema": "http://json-schema.org/draft-07/schema#",
    }

    if "title" in device_def:
        schema["title"] = device_def["title"]

    if "schema_description" in device_def:
        schema["description"] = device_def["schema_description"]

    schema["type"] = "object"
    schema["properties"] = properties
    schema["required"] = required
    schema["additionalProperties"] = False

    return schema


def save_schema(schema: Dict[str, Any], output_path: Path) -> None:
    output_path.parent.mkdir(parents=True, exist_ok=True)
    with open(output_path, "w", encoding="utf-8") as f:
        json.dump(schema, f, indent=2, ensure_ascii=False)


def main():
    parser = argparse.ArgumentParser(description="Generate device JSON schemas from taxonomy")
    parser.add_argument(
        "--input", "-i",
        default="device_types.json",
        help="Path to the taxonomy JSON file (default: device_types.json)"
    )
    parser.add_argument(
        "--output", "-o",
        default="schemas",
        help="Output directory for generated schemas, or output file path if --combined is set (default: schemas)"
    )
    parser.add_argument(
        "--combined", "-c",
        action="store_true",
        help="Output all schemas into a single JSON file instead of individual files per device type"
    )
    args = parser.parse_args()

    input_path = Path(args.input)
    output_path = Path(args.output)

    taxonomy = load_taxonomy(input_path)
    traits = taxonomy.get("traits", {})
    types = taxonomy.get("types", {})

    if not traits:
        print("ERROR: No 'traits' object found in taxonomy.")
        sys.exit(1)
    if not types:
        print("ERROR: No 'types' object found in taxonomy.")
        sys.exit(1)

    if args.combined:
        combined: Dict[str, Any] = {}
        for device_name, device_def in types.items():
            schema = generate_device_schema(device_name, device_def, traits)
            combined[device_name] = {
                "description": device_def.get("description", ""),
                "schema": schema
            }

        if output_path.suffix == "":
            output_file = output_path / "schemas.json"
        else:
            output_file = output_path

        output_file.parent.mkdir(parents=True, exist_ok=True)
        with open(output_file, "w", encoding="utf-8") as f:
            json.dump(combined, f, indent=2, ensure_ascii=False)

        print(f"Combined schema written to: {output_file}")
    else:
        for device_name, device_def in types.items():
            schema = generate_device_schema(device_name, device_def, traits)
            output_file = output_path / f"{device_name}.schema.json"
            save_schema(schema, output_file)

        print(f"Individual schemas written to: {output_path}/")


if __name__ == "__main__":
    main()
