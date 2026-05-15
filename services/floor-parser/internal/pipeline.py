from __future__ import annotations

from pathlib import Path
from tempfile import NamedTemporaryFile

from fastapi import UploadFile

from internal.classification.classifier import SemanticClassifier
from internal.entities.warnings import ParseWarning
from internal.export.floor_exporter import FloorExporter
from internal.normalization.geometry_normalizer import GeometryNormalizer
from internal.readers.dxf.extractor import DxfExtractor
from internal.readers.dxf.reader import DxfReader
from internal.topology.topology_builder import TopologyBuilder


def parse_floor_path(path: Path, *, source_name: str | None = None) -> dict[str, object]:
    warnings: list[ParseWarning] = []

    reader = DxfReader()
    extractor = DxfExtractor()
    normalizer = GeometryNormalizer()
    classifier = SemanticClassifier()
    topology_builder = TopologyBuilder()
    exporter = FloorExporter()

    read_result = reader.read_path(path)
    raw_plan = extractor.extract(read_result)
    normalized_entities = normalizer.normalize(raw_plan)
    classified_entities = classifier.classify(normalized_entities)
    floor_plan = topology_builder.build_floor(
        source_file=source_name or raw_plan.metadata.source_file,
        classified_entities=classified_entities,
        parsed_entity_count=len(raw_plan.entities),
    )
    return exporter.export(
        floor_plan,
        source=raw_plan.metadata.source_format.value,
        units=raw_plan.metadata.units,
        warnings=warnings,
    )


async def parse_floor(file: UploadFile) -> dict[str, object]:
    contents = await file.read()

    with NamedTemporaryFile(suffix=".dxf", delete=False) as temp_file:
        temp_file.write(contents)
        temp_path = Path(temp_file.name)
    try:
        return parse_floor_path(temp_path, source_name=file.filename or temp_path.name)
    finally:
        temp_path.unlink(missing_ok=True)
