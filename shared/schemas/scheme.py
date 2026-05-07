import json
from jsonschema import validate
from jsonschema.exceptions import ValidationError

schema = {
    "type": "object",
    "properties": {
        "name": {"type": "string"},
        "price": {"type": "number"},
        "ecosystem": {
            "type": "array",
            "items": {
                "type": "string",
            },
            "minItems": 1,
        },
        "protocol": {
            "type": "array",
            "items": {
                "type": "string",
            },
            "minItems": 1,
        },
        "power": {"type": "number"},
        "voltage": {
            "type": "object",
            "properties": {
                "min": {"type": "number", "minimum": 0},
                "max": {"type": "number", "minimum": 0}
            },
            "required": ["min", "max"]
        },
        "articul_YANDEX": {"type": "string"},
        "lifetime": {"type": "number"},
        "color_temperature": {"type": "number"},
        "color_rendering_index": {"type": "number"}
    },
    "required": ["name", "price", "ecosystem", "protocol", "power", "voltage", "articul_YANDEX"]
}

def make_json():
    filename = input("Название файла в формате name.json (просьба не использовать пробелы):\n").strip()
    data = {
        "name": input("Название товара:\n").strip(),
        "price": float(input("Цена(рубли):\n")),
        "ecosystem": [x.strip() for x in input("Экосистемы (через запятую):\n").split(",")],
        "protocol": [x.strip() for x in input("Протоколы (через запятую):\n").split(",")],
        "power": float(input("Мощность (Вт):\n")),
        "voltage": {
            "min": float(input("Минимальное напряжение (В):\n")),
            "max": float(input("Максимальное напряжение (В):\n"))
        },
        "articul_YANDEX": input("Артикул с ЯМ:\n").strip(),
        #необязательные поля, чтобы ничего не вводить написать -
        "lifetime": float(val) if (val := input("Срок службы (в годах):\n").strip()) != "-" else None,
        "color_temperature": float(val) if (val := input("Цветовая температура (К):\n").strip()) != "-" else None,
        "color_rendering_index": float(val) if (val := input("Индекс цветопередачи:\n").strip()) != "-" else None
    }
    data = {k: v for k, v in data.items() if v is not None}
    try:
        validate(instance=data, schema=schema)
        print("все хорошо")
    except ValidationError as e:
        print("ошибка валидации", e.message)
        return
    with open(filename, "w", encoding="utf-8") as f:
        json.dump(data, f, indent=4, ensure_ascii=False)

schema_sensor = {
"type": "object",
    "properties": {
        "type": {"type": "string"},
        "name": {"type": "string"},
        "price": {"type": "number"},
        "ecosystem": {
            "type": "array",
            "items": {
                "type": "string",
            },
            "minItems": 1,
        },
        "protocol": {
            "type": "array",
            "items": {
                "type": "string",
            },
            "minItems": 1,
        },
        "articul_YANDEX": {"type": "string"}
    }
}

schema_lamp = {
    "type": "object",
    "properties": {
        "type": {"type": "string"},
        "name": {"type": "string"},
        "price": {"type": "number"},
        "ecosystem": {
            "type": "array",
            "items": {
                "type": "string",
            },
            "minItems": 1,
        },
        "protocol": {
            "type": "array",
            "items": {
                "type": "string",
            },
            "minItems": 1,
        },
        "power": {"type": "number"},
        "voltage": {
            "type": "object",
            "properties": {
                "min": {"type": "number", "minimum": 0},
                "max": {"type": "number", "minimum": 0}
            },
            "required": ["min", "max"]
        },
        "articul_YANDEX": {"type": "string"},
        "lifetime": {"type": "number"},
        "color_temperature": {"type": "number"},
        "color_rendering_index": {"type": "number"}
    },
    "required": ["type","name", "price", "ecosystem", "protocol", "power", "voltage", "articul_YANDEX"]
}

schema_camera = {
    "type": "object",
    "properties": {
        "type": {"type": "string"},
        "name": {"type": "string"},
        "price": {"type": "number"},
        "ecosystem": {
            "type": "array",
            "items": {"type": "string"},
            "minItems": 1
        },
        "protocol": {
            "type": "array",
            "items": {"type": "string"},
            "minItems": 1
        },
        "resolution": {"type": "string"},
        "supports_microSD": {"type": "boolean"},
        "has_hub": {"type": "boolean"},
        "articul_YANDEX": {"type": "string"}
    },
    "required": ["type", "name", "price", "ecosystem", "protocol", "resolution", "supports_microSD", "has_hub", "articul_YANDEX"
    ]
}

schema_switcher = {
    "type": "object",
    "properties": {
        "type": {"type": "string"},
        "name": {"type": "string"},
        "price": {"type": "number"},
        "ecosystem": {
            "type": "array",
            "items": {"type": "string"},
            "minItems": 1
        },
        "protocol": {
            "type": "array",
            "items": {"type": "string"},
            "minItems": 1
        },
        "articul_YANDEX": {"type": "string"}
    }
}