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
        self._assert_floor_summary_matches_expected("square_room.dxf", "square_room.expected.json")

    def test_room_with_door_and_window(self):
        self._assert_floor_summary_matches_expected(
            "room_with_door_and_window.dxf",
            "room_with_door_and_window.expected.json",
        )

    def test_apartment_partition_lines(self):
        self._assert_floor_summary_matches_expected(
            "apartment_partition_lines.dxf",
            "apartment_partition_lines.expected.json",
        )

    def test_us_house_plan(self):
        self._assert_floor_summary_matches_expected("us_house_plan.dxf", "us_house_plan.expected.json")

    def test_two_bedroom_ensuite_apartment(self):
        self._assert_floor_summary_matches_expected(
            "two_bedroom_ensuite_apartment.dxf",
            "two_bedroom_ensuite_apartment.expected.json",
        )

    def test_one_bedroom_apartment(self):
        self._assert_floor_summary_matches_expected(
            "one_bedroom_apartment.dxf",
            "one_bedroom_apartment.expected.json",
        )

    def test_apartment_second_floor_insert_blocks(self):
        self._assert_floor_summary_matches_expected(
            "apartment_second_floor_insert_blocks.dxf",
            "apartment_second_floor_insert_blocks.expected.json",
        )

    def test_insert_backed_entities_are_extracted_from_blocks(self):
        service_dir = Path(__file__).resolve().parents[1]
        raw_plan = self._read_and_extract(service_dir / "data" / "apartment_first_floor_insert_blocks.dxf")

        self.assertGreater(len(raw_plan.entities), raw_plan.metadata.entity_count)
        self.assertTrue(
            any(
                entity.source_insert_id is not None and entity.layer in {"doors", "windows"}
                for entity in raw_plan.entities
            )
        )
        self.assertTrue(
            any(
                getattr(entity, "text", None) == "Bedroom"
                for entity in raw_plan.entities
            )
        )

    def test_insert_backed_openings_are_detected(self):
        parsed_plan = self._parse_named_dxf("apartment_first_floor_insert_blocks.dxf")

        self.assertEqual(parsed_plan["meta"]["units"], "mm")
        self.assertEqual(len(parsed_plan["doors"]), 4)
        self.assertEqual(len(parsed_plan["windows"]), 5)
        self.assertGreaterEqual(len(parsed_plan["rooms"]), 4)

        for door in parsed_plan["doors"]:
            self.assertGreaterEqual(door["width"], 890.0)
            self.assertLessEqual(door["width"], 910.0)
            self.assertTrue(door["rooms"])
            self.assertEqual(door.get("swing"), "single_swing")
            self.assertIn(door.get("hinge_side"), {"start", "end"})
            self.assertIn(door.get("opens_towards_room"), door["rooms"])

        for window in parsed_plan["windows"]:
            self.assertGreaterEqual(window["width"], 800.0)
            self.assertTrue(window.get("room"))

    def test_insert_backed_door_bindings_and_hinge_side_are_populated(self):
        classified = self._classify_dxf("apartment_first_floor_insert_blocks.dxf")

        self.assertEqual(len(classified.doors), 4)

        for door in classified.doors:
            self.assertIsNotNone(door.wall_id)
            self.assertTrue(door.support_wall_ids)
            self.assertEqual(door.swing, "single_swing")
            self.assertIn(door.hinge_side, {"start", "end"})
            self.assertIn(door.opens_towards_wall_side, {"positive_normal", "negative_normal"})

    def test_insert_backed_wall_widths_are_recovered(self):
        parsed_plan = self._parse_named_dxf("apartment_first_floor_insert_blocks.dxf")

        self.assertEqual(len(parsed_plan["walls"]), 27)
        self.assertTrue(all(wall["width"] > 0.0 for wall in parsed_plan["walls"]))
        self.assertTrue(all(wall["width"] > 100.0 for wall in parsed_plan["walls"]))

    def test_insert_backed_room_labels_are_populated(self):
        parsed_plan = self._parse_named_dxf("apartment_first_floor_insert_blocks.dxf")

        labeled_rooms = [
            room for room in parsed_plan["rooms"]
            if not room["name"].startswith("Room ")
        ]
        self.assertGreaterEqual(len(labeled_rooms), 3)

    def test_opening_bindings_are_populated(self):
        classified = self._classify_dxf("us_house_plan.dxf")

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

    def test_us_house_plan_has_multiple_labeled_rooms(self):
        parsed_plan = self._parse_named_dxf("us_house_plan.dxf")
        labeled_rooms = [
            room for room in parsed_plan["rooms"]
            if not room["name"].startswith("Room ")
        ]
        self.assertGreaterEqual(len(labeled_rooms), 6)

    def test_room_topology_is_populated_for_parser_fixtures(self):
        insert_plan = self._parse_named_dxf("apartment_first_floor_insert_blocks.dxf")
        self.assertEqual(len(insert_plan["rooms"]), 4)

        for room in insert_plan["rooms"]:
            self.assertGreater(room["area_m2"], 0.0)
            self.assertTrue(room["walls"])

        for door in insert_plan["doors"]:
            self.assertTrue(door["rooms"])
            if door.get("swing") == "single_swing":
                self.assertIn(door.get("opens_towards_room"), door["rooms"])

        for window in insert_plan["windows"]:
            self.assertTrue(window.get("room"))

        ensuite_plan = self._parse_named_dxf("two_bedroom_ensuite_apartment.dxf")
        self.assertEqual(len(ensuite_plan["rooms"]), 5)

        for room in ensuite_plan["rooms"]:
            self.assertGreater(room["area_m2"], 0.0)
            self.assertTrue(room["walls"])

    def test_room_walls_reference_exported_wall_ids(self):
        for dxf_filename in (
            "apartment_first_floor_insert_blocks.dxf",
            "one_bedroom_apartment.dxf",
            "two_bedroom_ensuite_apartment.dxf",
            "us_house_plan.dxf",
        ):
            parsed_plan = self._parse_named_dxf(dxf_filename)
            wall_ids = {wall["id"] for wall in parsed_plan["walls"]}
            for room in parsed_plan["rooms"]:
                self.assertTrue(set(room["walls"]).issubset(wall_ids))

    def test_opening_room_contracts_are_consistent(self):
        for dxf_filename in (
            "apartment_first_floor_insert_blocks.dxf",
            "one_bedroom_apartment.dxf",
            "two_bedroom_ensuite_apartment.dxf",
            "us_house_plan.dxf",
        ):
            parsed_plan = self._parse_named_dxf(dxf_filename)
            room_ids = {room["id"] for room in parsed_plan["rooms"]}

            for door in parsed_plan["doors"]:
                self.assertTrue(door["rooms"])
                self.assertTrue(set(door["rooms"]).issubset(room_ids))
                if "opens_towards_room" in door:
                    self.assertIn(door["opens_towards_room"], door["rooms"])

            for window in parsed_plan["windows"]:
                room_id = window.get("room")
                if room_id is not None:
                    self.assertIn(room_id, room_ids)

    def test_furniture_contracts_are_consistent(self):
        for dxf_filename in (
            "apartment_first_floor_insert_blocks.dxf",
            "apartment_second_floor_insert_blocks.dxf",
            "one_bedroom_apartment.dxf",
        ):
            parsed_plan = self._parse_named_dxf(dxf_filename)
            room_ids = {room["id"] for room in parsed_plan["rooms"]}
            furniture_ids = {item["id"] for item in parsed_plan["furniture"]}

            self.assertTrue(parsed_plan["furniture"])

            for item in parsed_plan["furniture"]:
                self.assertTrue(item["category"])
                self.assertGreaterEqual(len(item["points"]), 4)
                room_id = item.get("room")
                if room_id is not None:
                    self.assertIn(room_id, room_ids)

            for room in parsed_plan["rooms"]:
                self.assertTrue(set(room["furniture"]).issubset(furniture_ids))

    def _assert_floor_summary_matches_expected(self, dxf_filename: str, json_filename: str) -> None:
        service_dir = Path(__file__).resolve().parents[1]
        data_dir = service_dir / "data"
        dxf_path = data_dir / dxf_filename
        expected_json_path = data_dir / json_filename

        result = self._parse_dxf(dxf_path)

        with expected_json_path.open("r", encoding="utf-8") as expected_file:
            expected = json.load(expected_file)

        self.assertEqual(result["meta"]["units"], expected["meta"]["units"])
        self.assertEqual(len(result["walls"]), len(expected["walls"]))
        self.assertEqual(len(result["doors"]), len(expected["doors"]))
        self.assertEqual(len(result["windows"]), len(expected["windows"]))
        self.assertEqual(len(result["furniture"]), len(expected["furniture"]))
        self.assertEqual(len(result["rooms"]), len(expected["rooms"]))
        self.assertEqual(len(result["warnings"]), len(expected["warnings"]))

    def _parse_dxf(self, dxf_path: Path) -> dict[str, object]:
        raw_plan, classified_entities = self._classify_path(dxf_path)
        topology_builder = TopologyBuilder()
        exporter = FloorExporter()

        floor_plan = topology_builder.build_floor(
            source_file=dxf_path.name,
            classified_entities=classified_entities,
            units=raw_plan.metadata.units,
        )

        return exporter.export(
            floor_plan,
            source=raw_plan.metadata.source_format.value,
            units=raw_plan.metadata.units,
            warnings=[],
        )

    def _parse_named_dxf(self, dxf_filename: str) -> dict[str, object]:
        service_dir = Path(__file__).resolve().parents[1]
        return self._parse_dxf(service_dir / "data" / dxf_filename)

    def _classify_dxf(self, dxf_filename: str):
        service_dir = Path(__file__).resolve().parents[1]
        dxf_path = service_dir / "data" / dxf_filename
        _, classified_entities = self._classify_path(dxf_path)
        return classified_entities

    def _classify_path(self, dxf_path: Path):
        raw_plan = self._read_and_extract(dxf_path)
        normalizer = GeometryNormalizer()
        classifier = SemanticClassifier()
        normalized_entities = normalizer.normalize(raw_plan)
        return raw_plan, classifier.classify(normalized_entities, units=raw_plan.metadata.units)

    def _read_and_extract(self, dxf_path: Path):
        reader = DxfReader()
        extractor = DxfExtractor()

        read_result = reader.read_path(dxf_path)
        raw_plan = extractor.extract(read_result)
        return raw_plan


if __name__ == "__main__":
    unittest.main()
