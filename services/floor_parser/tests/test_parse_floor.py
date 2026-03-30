from __future__ import annotations

import json
import unittest
from pathlib import Path

from services.floor_parser.internal.classification.classifier import SemanticClassifier
from services.floor_parser.internal.export.floor_exporter import FloorExporter
from services.floor_parser.internal.normalization.geometry_normalizer import GeometryNormalizer
from services.floor_parser.internal.readers.dxf.extractor import DxfExtractor
from services.floor_parser.internal.readers.dxf.reader import DxfReader
from services.floor_parser.internal.topology.topology_builder import TopologyBuilder


class ParseFloorIntegrationTest(unittest.TestCase):
    def test_square_room(self):
        self._assert_floor_json_matches_expected("square_room.dxf", "square_room.json")

    def test_apartment_partition_lines(self):
        self._assert_floor_json_matches_expected("apartment_partition_lines.dxf", "apartment_partition_lines.json")

    def test_apartment_outline_polyline(self):
        self._assert_floor_json_matches_expected("apartment_outline_polyline.dxf", "apartment_outline_polyline.json")

    def test_floorplan(self):
        self._assert_floor_json_matches_expected("floorplan.dxf", "floorplan.json")

    def _assert_floor_json_matches_expected(self, dxf_filename: str, json_filename: str) -> None:
        tests_dir = Path(__file__).resolve().parent
        dxf_path = tests_dir / dxf_filename
        expected_json_path = tests_dir / json_filename

        result = self._parse_dxf(dxf_path)

        with expected_json_path.open("r", encoding="utf-8") as expected_file:
            expected = json.load(expected_file)

        self.assertEqual(result, expected)

    def _parse_dxf(self, dxf_path: Path) -> dict[str, object]:
        reader = DxfReader()
        extractor = DxfExtractor()
        normalizer = GeometryNormalizer()
        classifier = SemanticClassifier()
        topology_builder = TopologyBuilder()
        exporter = FloorExporter()

        read_result = reader.read_path(dxf_path)
        raw_plan = extractor.extract(read_result)
        normalized_entities = normalizer.normalize(raw_plan)
        classified_entities = classifier.classify(normalized_entities)
        floor_plan = topology_builder.build_floor(
            source_file=dxf_path.name,
            classified_entities=classified_entities,
            parsed_entity_count=len(raw_plan.entities),
        )

        return exporter.export(
            floor_plan,
            source=raw_plan.metadata.source_format.value,
            units=raw_plan.metadata.units,
        )


if __name__ == "__main__":
    unittest.main()
