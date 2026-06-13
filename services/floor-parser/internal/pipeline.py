from __future__ import annotations

from pathlib import Path
from tempfile import NamedTemporaryFile

import structlog
from fastapi import UploadFile

from internal.classification.classifier import SemanticClassifier
from internal.entities.warnings import ParseWarning
from internal.export.floor_exporter import FloorExporter
from internal.normalization.geometry_normalizer import GeometryNormalizer
from internal.readers.dxf.extractor import DxfExtractor
from internal.readers.dxf.reader import DxfReader
from internal.topology.topology_builder import TopologyBuilder

log = structlog.get_logger("floor-parser.pipeline")


async def parse_floor(file: UploadFile) -> dict[str, object]:
    filename = file.filename or ""
    if not filename.lower().endswith(".dxf"):
        raise ValueError("Only DXF files are supported on /parse.")
    return await parse_dxf_floor(file)


async def parse_dxf_floor(file: UploadFile) -> dict[str, object]:
    contents = await file.read()
    warnings: list[ParseWarning] = []

    with NamedTemporaryFile(suffix=".dxf", delete=False) as temp_file:
        temp_file.write(contents)
        temp_path = Path(temp_file.name)
    try:
        log.info("processing dxf file", filename=file.filename, size=len(contents))

        dxf_reader = DxfReader()
        extractor = DxfExtractor()
        normalizer = GeometryNormalizer()
        classifier = SemanticClassifier()
        topology_builder = TopologyBuilder()
        exporter = FloorExporter()

        read_result = dxf_reader.read_path(temp_path)
        raw_plan = extractor.extract(read_result)
        normalized_entities = normalizer.normalize(raw_plan)
        classified_entities = classifier.classify(normalized_entities, units=raw_plan.metadata.units)
        floor_plan = topology_builder.build_floor(
            source_file=file.filename or raw_plan.metadata.source_file,
            classified_entities=classified_entities,
            units=raw_plan.metadata.units,
        )
        result = exporter.export(
            floor_plan,
            source=raw_plan.metadata.source_format.value,
            units=raw_plan.metadata.units,
            warnings=warnings,
        )

        rooms = result.get("floor_plan", {}).get("rooms", [])
        log.info("dxf processing completed", filename=file.filename, rooms=len(rooms))
        return result
    except Exception:
        log.exception("dxf processing failed", filename=file.filename)
        raise
    finally:
        temp_path.unlink(missing_ok=True)
