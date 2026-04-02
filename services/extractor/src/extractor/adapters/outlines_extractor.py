import json
import copy
from typing import Any
import outlines
from outlines.models.base import AsyncModel
from outlines.types import JsonSchema

from extractor.domain.models import DetectedDeviceType, ListingSnapshot


def _preprocess_schema(schema: dict[str, Any], hints: dict[str, str] = {}) -> dict[str, Any]:
    schema = copy.deepcopy(schema)
    properties = schema.get("properties", {})
    
    # make all fields required
    schema["required"] = list(properties.keys())
    
    # make all fields nullable
    # except:
    # const -> stays const
    # array -> can be empty even if minItems = 1
    for field_name, field_schema in properties.items():
        if "const" in field_schema:
            continue

        # Add 'return null' suffixes to descriptions to help avoid hallucinations
        field_type = field_schema.get("type")
        is_array = field_type == "array" or (isinstance(field_type, list) and "array" in field_type)
        
        suffix = "Return empty array if not found." if is_array else "Return null if not found."
        if field_name in hints:
            suffix += " " + hints[field_name]
        
        if "description" in field_schema:
            field_schema["description"] = field_schema["description"].rstrip(".") + ". " + suffix
        else:
            field_schema["description"] = suffix

        if "type" in field_schema:
            current_type = field_schema["type"]
            if isinstance(current_type, list):
                if "null" not in current_type:
                    field_schema["type"] = current_type + ["null"]
            else:
                field_schema["type"] = [current_type, "null"]
        if "enum" in field_schema:
            if None not in field_schema["enum"]:
                field_schema["enum"] = field_schema["enum"] + [None]
        if "type" in field_schema and "array" in field_schema["type"]:
            # arrays can be empty instead of null
            field_schema.pop("minItems", None)
            field_schema["type"] = "array"

    return schema

class FieldInfo:
    def __init__(self, is_unique_items_array: bool):
        self.is_unique_items_array = is_unique_items_array


class DeviceTypeInfo:
    def __init__(self, name: str, description: str, field_descriptions: str, fields: dict[str, FieldInfo], preprocessed_schema: JsonSchema):
        self.name = name
        self.description = description
        self.field_descriptions = field_descriptions
        self.fields = fields
        self.preprocessed_schema = preprocessed_schema


class OutlinesExtractor:
    def __init__(self, taxonomy: dict[str, Any], model: AsyncModel, hints: dict[str, str] = {}, temperature: float = 0):
        """
            taxonomy: dict of device type name to { description, schema }
            model: model instance to use for outlines
            hints: dict of field name to extraction hint. The extraction hint is appended to the description of the respective field.
        """
        self._model = model
        self._temperature = temperature
        self._device_types: dict[str, DeviceTypeInfo] = {}

        for type_name, type_data in taxonomy.items():
            canonical_schema = type_data["schema"]
            preprocessed = _preprocess_schema(canonical_schema, hints)
            field_descriptions = "\n".join(
                f"- {name}: {property["description"]}" if "description" in property.keys() else ""
                for name, property in preprocessed["properties"].items()
            )
            fields: dict[str, FieldInfo] = {}
            for name, property in preprocessed["properties"].items():
                if property["type"] == "array" and "uniqueItems" in property:
                    # outlines doesn't support uniqueItems
                    del property['uniqueItems']
                    fields[name] = FieldInfo(is_unique_items_array=True)
            
            self._device_types[type_name] = DeviceTypeInfo(
                name=type_name,
                description=type_data["description"],
                field_descriptions=field_descriptions,
                fields=fields,
                preprocessed_schema=JsonSchema(preprocessed),
            )
        
        self._detection_schema = JsonSchema({
            "type": "object",
            "properties": {
                "device_type": {
                    "type": "string",
                    "enum": list(self._device_types.keys())
                },
                "confidence": {
                    "type": "number",
                    "minimum": 0.0,
                    "maximum": 1.0
                }
            },
            "required": ["device_type", "confidence"]
        })

        self._type_descriptions = "\n".join(
            f"- {name}: {info.description}"
            for name, info in self._device_types.items()
        )
        

    def _build_detection_prompt(self, listing: ListingSnapshot) -> str:
        return f"""You are classifying shop listings into these smart home device types.

Device types and their descriptions:
{self._type_descriptions}

Listing name: {listing.name}
Listing brand: {listing.brand}
Listing text: {listing.text}

Classify this listing into one of the device types and provide a confidence score from 0.0 to 1.0."""


    def _build_extraction_prompt(self, listing: ListingSnapshot, type_info: DeviceTypeInfo) -> str:
        return f"""You are extracting smart home device characteristics from its marketplace listing.
Device type and description: {type_info.name} - {type_info.description}

Field names and descriptions:
{type_info.field_descriptions}

Listing name: {listing.name}
Listing brand: {listing.brand}
Listing text: {listing.text}

Extract all device attributes from the listing information. Use null for any field not mentioned in the listing.

Step-by-step reasoning. Quote the raw text that proves the respective field. State if they are missing. Do this BEFORE filling out the rest of the schema."

CRITICAL RULES:

NO OUTSIDE KNOWLEDGE: Do not assume a device works with Xiaomi just because it is an Aqara device. Do not assume a device is WiFi. ONLY extract what is literally written in the text.
IF MISSING: If a protocol, ecosystem, or specification is not explicitly mentioned in the text, you must omit it or select 'null'. Do not guess.
"""
    

    async def detect_device_type(self, listing: ListingSnapshot) -> DetectedDeviceType:
        generator = outlines.Generator(self._model, self._detection_schema)
        prompt = self._build_detection_prompt(listing)
        result = json.loads(await generator(prompt))
        return DetectedDeviceType(type=result["device_type"], confidence=result["confidence"])


    async def extract(self, listing: ListingSnapshot, as_type: str) -> dict[str, Any]:
        if as_type not in self._device_types:
            raise ValueError(f"Unknown device type: {as_type}")
        
        type_info = self._device_types[as_type]
        generator = outlines.Generator(self._model, type_info.preprocessed_schema)
        prompt = self._build_extraction_prompt(listing, type_info)
        result = json.loads(await generator(prompt))
        for field in result:
            if field in type_info.fields and type_info.fields[field].is_unique_items_array:
                result[field] = list(set(result[field]))
        return result

