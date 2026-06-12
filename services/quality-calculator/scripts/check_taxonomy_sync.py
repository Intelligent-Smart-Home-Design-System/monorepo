"""
Проверяет согласованность канонической таксономии (shared/schemas/devices/device_types.json)
с разрешённой схемой извлечения, которую использует extractor (services/extractor/taxonomy_schema.json).

device_types.json хранит таксономию в виде traits + types (тип = композиция трейтов).
taxonomy_schema.json хранит уже "разрешённую" плоскую схему: type -> {description, schema: {properties}}.

Скрипт:
  1. разрешает каждый тип из device_types.json в полный набор полей (base + его trait'ы + extra_schema);
  2. сравнивает множества типов и множества полей по каждому общему типу;
  3. печатает отчёт и возвращает ненулевой код, если найдены расхождения.

Использование:
    uv run python scripts/check_taxonomy_sync.py
    uv run python scripts/check_taxonomy_sync.py --device-types <path> --taxonomy <path>
"""
import argparse
import json
import os
import sys
from typing import Any


def _default_paths() -> tuple[str, str]:
    here = os.path.dirname(os.path.abspath(__file__))
    device_types = os.path.abspath(
        os.path.join(here, "..", "..", "..", "shared", "schemas", "devices", "device_types.json")
    )
    taxonomy = os.path.abspath(
        os.path.join(here, "..", "..", "extractor", "taxonomy_schema.json")
    )
    return device_types, taxonomy


def resolve_type_properties(device_types: dict[str, Any], type_name: str) -> dict[str, Any]:
    """Собирает полный набор полей типа: свойства всех его трейтов + extra_schema."""
    traits = device_types["traits"]
    type_def = device_types["types"][type_name]
    props: dict[str, Any] = {}
    for trait_name in type_def.get("traits", []):
        props.update(traits.get(trait_name, {}).get("properties", {}))
    props.update(type_def.get("extra_schema", {}).get("properties", {}))
    return props


def load(path: str) -> dict[str, Any]:
    with open(path, "r", encoding="utf-8") as f:
        return json.load(f)


def main() -> int:
    dt_default, tax_default = _default_paths()
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--device-types", default=dt_default)
    parser.add_argument("--taxonomy", default=tax_default)
    # Поля, которые есть только в canonical-схеме и намеренно не извлекаются LLM
    # (служебные / каталожные), поэтому их отсутствие в taxonomy_schema — не ошибка.
    parser.add_argument(
        "--ignore-fields",
        nargs="*",
        default=["type", "price", "articul_YANDEX"],
        help="Поля canonical-схемы, отсутствие которых в taxonomy_schema не считается расхождением.",
    )
    parser.add_argument(
        "--ignore-types",
        nargs="*",
        default=["unknown"],
        help="Типы taxonomy_schema, отсутствие которых в device_types не считается расхождением.",
    )
    args = parser.parse_args()

    try:
        device_types = load(args.device_types)
        taxonomy = load(args.taxonomy)
    except FileNotFoundError as e:
        print(f"\u274c Файл не найден: {e}")
        return 1

    ignore_fields = set(args.ignore_fields)
    ignore_types = set(args.ignore_types)

    dt_types = set(device_types.get("types", {}).keys()) - ignore_types
    tax_types = set(taxonomy.keys()) - ignore_types

    errors: list[str] = []
    warnings: list[str] = []

    print("\U0001f50d Проверка синхронизации device_types.json <-> taxonomy_schema.json\n")

    # 1. Сравнение множеств типов
    only_dt = sorted(dt_types - tax_types)
    only_tax = sorted(tax_types - dt_types)
    if only_dt:
        errors.append(f"Типы есть в device_types.json, но НЕТ в taxonomy_schema.json: {only_dt}")
    if only_tax:
        errors.append(f"Типы есть в taxonomy_schema.json, но НЕТ в device_types.json: {only_tax}")

    # 2. Сравнение полей по общим типам
    common = sorted(dt_types & tax_types)
    for t in common:
        dt_props = set(resolve_type_properties(device_types, t).keys())
        tax_props = set(taxonomy[t].get("schema", {}).get("properties", {}).keys())

        missing_in_tax = sorted((dt_props - tax_props) - ignore_fields)
        missing_in_dt = sorted(tax_props - dt_props)

        if missing_in_tax:
            errors.append(f"[{t}] поля canonical-схемы отсутствуют в taxonomy_schema: {missing_in_tax}")
        if missing_in_dt:
            errors.append(f"[{t}] поля taxonomy_schema отсутствуют в device_types: {missing_in_dt}")

    # 3. Отчёт
    print(f"  Типов в device_types.json : {len(dt_types)} -> {sorted(dt_types)}")
    print(f"  Типов в taxonomy_schema   : {len(tax_types)} (без {sorted(ignore_types)})")
    print(f"  Общих типов               : {len(common)} -> {common}\n")

    if errors:
        print("\u274c РАССИНХРОН. Найдены расхождения:")
        for err in errors:
            print(f"  - {err}")
        if warnings:
            print("\n\u26a0\ufe0f  Предупреждения:")
            for w in warnings:
                print(f"  - {w}")
        print(
            "\nЭто значит, что quality-calculator не сможет прочитать характеристики из золотого слоя "
            "по тем именам, что заданы в evaluation_traits.json/device_types.json."
        )
        return 1

    print("\u2705 Схемы синхронизированы: наборы типов и полей согласованы.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
