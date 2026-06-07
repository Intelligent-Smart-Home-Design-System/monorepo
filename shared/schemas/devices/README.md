Из корня проекта

rm services/extractor/taxonomy_schema.json 
python3 shared/schemas/devices/schema_generator.py --input shared/schemas/devices/device_types.json --output services/extractor/taxonomy_schema.json --combined

python3 shared/schemas/devices/schema_generator.py --input shared/schemas/devices/device_types.json --output shared/schemas/devices/schemas