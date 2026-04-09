from __future__ import annotations

from pathlib import Path
from tempfile import NamedTemporaryFile

from fastapi import UploadFile

from services.floor_parser.internal.classification.classifier import SemanticClassifier
from services.floor_parser.internal.entities.warnings import ParseWarning
from services.floor_parser.internal.export.floor_exporter import FloorExporter
from services.floor_parser.internal.normalization.geometry_normalizer import GeometryNormalizer
from services.floor_parser.internal.readers.dxf.extractor import DxfExtractor
from services.floor_parser.internal.readers.dxf.reader import DxfReader
from services.floor_parser.internal.topology.topology_builder import TopologyBuilder


async def parse_floor(file: UploadFile) -> dict[str, object]:
    contents = await file.read()
    warnings: list[ParseWarning] = []

    with NamedTemporaryFile(suffix=".dxf", delete=False) as temp_file:
        temp_file.write(contents)
        temp_path = Path(temp_file.name)
    try:
        reader = DxfReader()
        extractor = DxfExtractor()
        normalizer = GeometryNormalizer()
        classifier = SemanticClassifier()
        topology_builder = TopologyBuilder()
        exporter = FloorExporter()

        read_result = reader.read_path(temp_path)
        raw_plan = extractor.extract(read_result)
        normalized_entities = normalizer.normalize(raw_plan)
        classified_entities = classifier.classify(normalized_entities)
        floor_plan = topology_builder.build_floor(
            source_file=file.filename or raw_plan.metadata.source_file,
            classified_entities=classified_entities,
            parsed_entity_count=len(raw_plan.entities),
        )
        return exporter.export(
            floor_plan,
            source=raw_plan.metadata.source_format.value,
            units=raw_plan.metadata.units,
            warnings=warnings,
        )
    finally:
        temp_path.unlink(missing_ok=True)
