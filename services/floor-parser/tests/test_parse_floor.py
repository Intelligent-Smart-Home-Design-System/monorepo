from __future__ import annotations

import json
import unittest
from pathlib import Path

from internal.classification.classifier import SemanticClassifier
from internal.export.floor_exporter import FloorExporter
from internal.normalization.geometry_normalizer import GeometryNormalizer
from internal.readers.dxf.extractor import DxfExtractor
from internal.readers.dxf.reader import DxfReader
from internal.topology.topology_builder import TopologyBuilder


class ParseFloorIntegrationTest(unittest.TestCase):
    def test_square_room(self):
        self._assert_floor_json_matches_expected("square_room.dxf", "square_room.json")

    def test_door_and_window(self):
        self._assert_floor_json_matches_expected("door_and_window.dxf", "door_and_window.json")

    def test_apartment_partition_lines(self):
        self._assert_floor_json_matches_expected("apartment_partition_lines.dxf", "apartment_partition_lines.json")

    def test_apartment_outline_polyline(self):
        self._assert_floor_json_matches_expected("apartment_outline_polyline.dxf", "apartment_outline_polyline.json")

    def test_floorplan(self):
        self._assert_floor_json_matches_expected("floorplan.dxf", "floorplan.json")

    def test_opening_bindings_are_populated(self):
        classified = self._classify_dxf("floorplan.dxf")

        self.assertTrue(classified.doors)
        self.assertTrue(classified.windows)

        for door in classified.doors:
            self.assertIsNotNone(door.wall_id)
            self.assertTrue(door.support_wall_ids)
            if door.swing == "single_swing":
                self.assertIn(door.opens_towards_wall_side, {"positive_normal", "negative_normal"})

        for window in classified.windows:
            self.assertIsNotNone(window.wall_id)
            self.assertTrue(window.support_wall_ids)

    def _assert_floor_json_matches_expected(self, dxf_filename: str, json_filename: str) -> None:
        service_dir = Path(__file__).resolve().parents[1]
        data_dir = service_dir / "data"
        dxf_path = data_dir / dxf_filename
        expected_json_path = data_dir / json_filename

        result = self._parse_dxf(dxf_path)

        with expected_json_path.open("r", encoding="utf-8") as expected_file:
            expected = json.load(expected_file)

        self.assertEqual(result, expected)

    def _parse_dxf(self, dxf_path: Path) -> dict[str, object]:
        raw_plan, classified_entities = self._classify_path(dxf_path)
        topology_builder = TopologyBuilder()
        exporter = FloorExporter()

        floor_plan = topology_builder.build_floor(
            source_file=dxf_path.name,
            classified_entities=classified_entities,
            parsed_entity_count=len(raw_plan.entities),
        )

        return exporter.export(
            floor_plan,
            source=raw_plan.metadata.source_format.value,
            units=raw_plan.metadata.units,
            warnings=[],
        )

    def _classify_dxf(self, dxf_filename: str):
        service_dir = Path(__file__).resolve().parents[1]
        dxf_path = service_dir / "data" / dxf_filename
        _, classified_entities = self._classify_path(dxf_path)
        return classified_entities

    def _classify_path(self, dxf_path: Path):
        raw_plan, _ = self._read_and_extract(dxf_path)
        normalizer = GeometryNormalizer()
        classifier = SemanticClassifier()
        normalized_entities = normalizer.normalize(raw_plan)
        return raw_plan, classifier.classify(normalized_entities, units=raw_plan.metadata.units)

    def _read_and_extract(self, dxf_path: Path):
        reader = DxfReader()
        extractor = DxfExtractor()

        read_result = reader.read_path(dxf_path)
        raw_plan = extractor.extract(read_result)
        return raw_plan, len(raw_plan.entities)


if __name__ == "__main__":
    unittest.main()
