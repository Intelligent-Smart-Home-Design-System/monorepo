from __future__ import annotations

from dataclasses import dataclass
import re

from services.floor_parser.internal.entities.geometry import ArcEntity, InsertEntity, LineEntity, NormalizedEntity, Point, PolylineEntity, TextEntity


OPENING_LAYER_MARKERS = ("opening", "open", "door", "window", "wind", "glaz")
HEADER_LAYER_MARKERS = ("header", "frame", "jamb")
GARAGE_DOOR_LAYER_MARKERS = ("garage", "overhead")
WINDOW_OPERATION_MARKERS = ("XO", "OX", "SH", "HS", "DH", "FX")
SLIDING_DOOR_MARKERS = ("SGD", "SLIDING", "PATIO")
DOOR_BLOCK_MARKERS = ("door", "dr", "doorleaf", "door-panel")

BASE_TEXT_TO_SEGMENT_DISTANCE = 80.0
BASE_TEXT_TO_ARC_DISTANCE = 40.0
BASE_MIN_OPENING_LENGTH = 20.0


@dataclass(frozen=True)
class OpeningSegment:
    start: Point
    end: Point
    source_entity_id: str
    layer: str


class OpeningDetector:
    def detect_doors(self, entities: list[NormalizedEntity]) -> list[LineEntity]:
        opening_segments = self._collect_segments(entities, use_header_layers=False)
        header_segments = self._collect_segments(entities, use_header_layers=True)
        opening_arcs = [
            entity for entity in entities
            if isinstance(entity, ArcEntity) and self._is_opening_layer(entity.layer)
        ]

        doors: list[LineEntity] = []

        for entity in entities:
            if isinstance(entity, TextEntity):
                normalized_text = " ".join(entity.text.strip().upper().split())
                tokens = re.findall(r"[A-Z0-9]+", normalized_text)

                if {"O", "H", "DOOR"} <= set(tokens) or {"OH", "DOOR"} <= set(tokens):
                    continue

                expected_width = self._get_label_width(normalized_text)
                matched_segment: OpeningSegment | None = None

                if self._is_sliding_door_label(tokens):
                    matched_segment = self._match_opening_segment(
                        entity.insert,
                        opening_segments,
                        expected_length=expected_width,
                    )
                elif self._is_swing_door_label(normalized_text):
                    matched_segment = self._match_opening_segment(
                        entity.insert,
                        header_segments,
                        expected_length=expected_width,
                    )

                    has_nearby_arc = any(
                        self._get_length(entity.insert, arc.center) <= self._get_arc_search_distance(expected_width)
                        for arc in opening_arcs
                    )

                    if matched_segment is None or (
                        not has_nearby_arc and not self._is_precise_length_match(matched_segment, expected_width)
                    ):
                        matched_segment = self._match_opening_segment(
                            entity.insert,
                            opening_segments,
                            expected_length=expected_width,
                        )

                if matched_segment is None:
                    continue

                doors.append(
                    LineEntity(
                        id=entity.id,
                        layer=matched_segment.layer,
                        start=matched_segment.start,
                        end=matched_segment.end,
                    )
                )

            if isinstance(entity, InsertEntity):
                normalized_name = " ".join(entity.block_name.strip().upper().split()).replace("_", "-").lower()
                if any(marker in normalized_name for marker in GARAGE_DOOR_LAYER_MARKERS):
                    continue
                if not any(marker in normalized_name for marker in DOOR_BLOCK_MARKERS):
                    continue

                matched_segment = self._match_opening_segment(entity.insert, header_segments)
                if matched_segment is None:
                    matched_segment = self._match_opening_segment(entity.insert, opening_segments)

                if matched_segment is None:
                    continue

                doors.append(
                    LineEntity(
                        id=entity.id,
                        layer=matched_segment.layer,
                        start=matched_segment.start,
                        end=matched_segment.end,
                    )
                )

        return self._dedupe_linear_entities(doors)

    def detect_windows(self, entities: list[NormalizedEntity]) -> list[LineEntity]:
        opening_segments = self._collect_segments(entities, use_header_layers=False)
        header_segments = self._collect_segments(entities, use_header_layers=True)
        windows: list[LineEntity] = []

        for entity in entities:
            if not isinstance(entity, TextEntity):
                continue

            normalized_text = " ".join(entity.text.strip().upper().split())
            tokens = re.findall(r"[A-Z0-9]+", normalized_text)
            if not any(marker in tokens for marker in WINDOW_OPERATION_MARKERS):
                continue

            expected_width = self._get_label_width(normalized_text)
            matched_segment = self._match_opening_segment(
                entity.insert,
                header_segments,
                expected_length=expected_width,
            )
            if matched_segment is None:
                matched_segment = self._match_opening_segment(
                    entity.insert,
                    opening_segments,
                    expected_length=expected_width,
                )
            if matched_segment is None:
                continue

            windows.append(
                LineEntity(
                    id=entity.id,
                    layer=matched_segment.layer,
                    start=matched_segment.start,
                    end=matched_segment.end,
                )
            )

        return self._dedupe_linear_entities(windows)

    def _collect_segments(
        self,
        entities: list[NormalizedEntity],
        use_header_layers: bool,
    ) -> list[OpeningSegment]:
        segments: list[OpeningSegment] = []

        for entity in entities:
            layer_matches = self._is_header_layer(entity.layer) if use_header_layers else self._is_opening_layer(entity.layer)
            if not layer_matches:
                continue

            if isinstance(entity, LineEntity):
                segments.append(
                    OpeningSegment(
                        start=entity.start,
                        end=entity.end,
                        source_entity_id=entity.id,
                        layer=entity.layer,
                    )
                )
                continue

            if isinstance(entity, PolylineEntity):
                points = entity.points
                if len(points) < 2:
                    continue

                for index in range(len(points) - 1):
                    segments.append(
                        OpeningSegment(
                            start=points[index],
                            end=points[index + 1],
                            source_entity_id=f"{entity.id}:{index + 1}",
                            layer=entity.layer,
                        )
                    )

                if entity.closed:
                    segments.append(
                        OpeningSegment(
                            start=points[-1],
                            end=points[0],
                            source_entity_id=f"{entity.id}:closing",
                            layer=entity.layer,
                        )
                    )

        return segments

    def _is_sliding_door_label(self, tokens: list[str]) -> bool:
        collapsed = "".join(tokens)
        return any(marker in tokens or marker in collapsed for marker in SLIDING_DOOR_MARKERS)

    def _is_swing_door_label(self, text: str) -> bool:
        return len(text) == 4 and text.isdigit()

    def _match_opening_segment(
        self,
        anchor: Point,
        opening_segments: list[OpeningSegment],
        expected_length: float | None = None,
    ) -> OpeningSegment | None:
        candidates = []

        for segment in opening_segments:
            orientation = self._get_segment_orientation(segment.start, segment.end)
            if orientation is None:
                continue

            length = self._get_length(segment.start, segment.end)
            if length < self._get_min_opening_length(expected_length):
                continue

            midpoint = self._get_midpoint(segment.start, segment.end)
            distance = self._get_length(anchor, midpoint)
            if distance > self._get_segment_search_distance(expected_length):
                continue

            if orientation == "horizontal":
                min_x = min(segment.start.x, segment.end.x) - 24.0
                max_x = max(segment.start.x, segment.end.x) + 24.0
                if not (min_x <= anchor.x <= max_x and abs(anchor.y - midpoint.y) <= 40.0):
                    continue
            else:
                min_y = min(segment.start.y, segment.end.y) - 24.0
                max_y = max(segment.start.y, segment.end.y) + 24.0
                if not (min_y <= anchor.y <= max_y and abs(anchor.x - midpoint.x) <= 40.0):
                    continue

            length_penalty = 0.0
            if expected_length is not None:
                length_penalty = round(abs(length - expected_length), 3)

            candidates.append((length_penalty, distance, -length, orientation, segment))

        if not candidates:
            return None

        _, _, _, orientation, primary_segment = min(candidates)

        grouped_segments = []
        for _, _, _, candidate_orientation, candidate_segment in candidates:
            if candidate_orientation != orientation:
                continue

            left_length = self._get_length(primary_segment.start, primary_segment.end)
            right_length = self._get_length(candidate_segment.start, candidate_segment.end)
            length_ratio = min(left_length, right_length) / max(left_length, right_length)
            if length_ratio < 0.7:
                continue

            left_midpoint = self._get_midpoint(primary_segment.start, primary_segment.end)
            right_midpoint = self._get_midpoint(candidate_segment.start, candidate_segment.end)

            if orientation == "horizontal" and abs(left_midpoint.y - right_midpoint.y) > 8.0:
                continue
            if orientation == "vertical" and abs(left_midpoint.x - right_midpoint.x) > 8.0:
                continue

            grouped_segments.append(candidate_segment)

        if orientation == "horizontal":
            x_start = min(min(segment.start.x, segment.end.x) for segment in grouped_segments)
            x_end = max(max(segment.start.x, segment.end.x) for segment in grouped_segments)
            y = sum(self._get_midpoint(segment.start, segment.end).y for segment in grouped_segments) / len(grouped_segments)
            return OpeningSegment(
                start=Point(x=x_start, y=round(y, 6)),
                end=Point(x=x_end, y=round(y, 6)),
                source_entity_id=grouped_segments[0].source_entity_id,
                layer=grouped_segments[0].layer,
            )

        y_start = min(min(segment.start.y, segment.end.y) for segment in grouped_segments)
        y_end = max(max(segment.start.y, segment.end.y) for segment in grouped_segments)
        x = sum(self._get_midpoint(segment.start, segment.end).x for segment in grouped_segments) / len(grouped_segments)
        return OpeningSegment(
            start=Point(x=round(x, 6), y=y_start),
            end=Point(x=round(x, 6), y=y_end),
            source_entity_id=grouped_segments[0].source_entity_id,
            layer=grouped_segments[0].layer,
        )

    def _is_opening_layer(self, layer: str) -> bool:
        normalized_layer = layer.strip().lower()
        return (
            any(marker in normalized_layer for marker in OPENING_LAYER_MARKERS)
            and not any(marker in normalized_layer for marker in GARAGE_DOOR_LAYER_MARKERS)
        )

    def _is_header_layer(self, layer: str) -> bool:
        normalized_layer = layer.strip().lower()
        return (
            any(marker in normalized_layer for marker in HEADER_LAYER_MARKERS)
            and not any(marker in normalized_layer for marker in GARAGE_DOOR_LAYER_MARKERS)
        )

    def _get_label_width(self, text: str) -> float | None:
        digits = "".join(character for character in text if character.isdigit())
        if len(digits) < 2:
            return None

        feet = int(digits[0])
        inches = int(digits[1])
        return float(feet * 12 + inches)

    def _get_segment_search_distance(self, expected_length: float | None) -> float:
        if expected_length is None:
            return BASE_TEXT_TO_SEGMENT_DISTANCE
        return max(BASE_TEXT_TO_SEGMENT_DISTANCE, expected_length * 2.0)

    def _get_arc_search_distance(self, expected_length: float | None) -> float:
        if expected_length is None:
            return BASE_TEXT_TO_ARC_DISTANCE
        return max(BASE_TEXT_TO_ARC_DISTANCE, expected_length * 1.25)

    def _get_min_opening_length(self, expected_length: float | None) -> float:
        if expected_length is None:
            return BASE_MIN_OPENING_LENGTH
        return max(BASE_MIN_OPENING_LENGTH, expected_length * 0.5)

    def _is_precise_length_match(self, segment: OpeningSegment, expected_length: float | None) -> bool:
        if expected_length is None:
            return False
        return abs(self._get_length(segment.start, segment.end) - expected_length) <= 2.5

    def _dedupe_linear_entities(self, entities: list[LineEntity]) -> list[LineEntity]:
        deduped = []
        seen = set()

        for entity in entities:
            key = (
                round(entity.start.x, 6),
                round(entity.start.y, 6),
                round(entity.end.x, 6),
                round(entity.end.y, 6),
            )
            reverse_key = (
                round(entity.end.x, 6),
                round(entity.end.y, 6),
                round(entity.start.x, 6),
                round(entity.start.y, 6),
            )
            if key in seen or reverse_key in seen:
                continue

            seen.add(key)
            deduped.append(entity)

        return deduped

    def _get_segment_orientation(self, start: Point, end: Point) -> str | None:
        if abs(start.y - end.y) <= 1.0:
            return "horizontal"
        if abs(start.x - end.x) <= 1.0:
            return "vertical"
        return None

    def _get_midpoint(self, start: Point, end: Point) -> Point:
        return Point(
            x=round((start.x + end.x) / 2.0, 6),
            y=round((start.y + end.y) / 2.0, 6),
        )

    def _get_length(self, start: Point, end: Point) -> float:
        return ((end.x - start.x) ** 2 + (end.y - start.y) ** 2) ** 0.5
