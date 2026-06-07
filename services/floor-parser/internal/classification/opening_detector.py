from __future__ import annotations

from statistics import median
from math import cos, radians, sin
import re
from dataclasses import dataclass
from typing import Literal

from internal.classification.config import (
    OPENING_DETECTOR_CONFIG,
    OpeningThresholds,
    build_opening_thresholds,
)
from internal.entities.floor import Door, Wall, Window
from internal.entities.geometry import (
    ArcEntity,
    InsertEntity,
    LineEntity,
    NormalizedEntity,
    Point,
    PolylineEntity,
    TextEntity,
)

Orientation = Literal["horizontal", "vertical"]


@dataclass(frozen=True)
class Segment:
    start: Point
    end: Point
    source_entity_id: str
    layer: str


@dataclass(frozen=True)
class Gap:
    id: str
    support_wall_ids: tuple[str, str]
    start: Point
    end: Point
    center: Point
    direction: Point
    normal: Point
    length: float


@dataclass(frozen=True)
class GapHint:
    kind: str
    source_id: str
    anchor: Point
    expected_length: float | None = None


@dataclass(frozen=True)
class DoorBlockGeometry:
    closed_line: LineEntity
    hinge_point: Point
    open_point: Point
    hinge_side: Literal["start", "end"]


@dataclass(frozen=True)
class OpeningContext:
    walls: list[Wall]
    opening_segments: list[Segment]
    header_segments: list[Segment]
    direct_door_segments: list[Segment]
    direct_window_segments: list[Segment]
    arcs: list[ArcEntity]
    texts: list[TextEntity]
    inserts: list[InsertEntity]
    insert_content_ids: frozenset[str]
    door_block_geometries: dict[str, DoorBlockGeometry]
    window_block_lines: dict[str, LineEntity]
    insert_gap_ids: dict[str, str]
    gaps: list[Gap]
    wall_host_ids: dict[str, str]
    thresholds: OpeningThresholds


class OpeningDetector:
    def detect(
        self,
        entities: list[NormalizedEntity],
        walls: list[Wall],
        units: str | None = None,
    ) -> tuple[list[Door], list[Window]]:
        context = _build_opening_context(entities, walls, build_opening_thresholds(units))
        detected_doors = _detect_door_lines(entities, context)
        detected_windows = _detect_window_lines(entities, context)
        resolved_context = _with_wall_host_ids(context, [*detected_doors, *detected_windows])
        doors = [_line_to_door(entity, resolved_context) for entity in detected_doors]
        windows = [_line_to_window(entity, resolved_context) for entity in detected_windows]
        return (
            [door for door in doors if _is_bound_opening(door)],
            [window for window in windows if _is_bound_opening(window)],
        )

    def detect_doors(self, entities: list[NormalizedEntity], walls: list[Wall], units: str | None = None) -> list[Door]:
        doors, _ = self.detect(entities, walls, units=units)
        return doors

    def detect_windows(self, entities: list[NormalizedEntity], walls: list[Wall], units: str | None = None) -> list[Window]:
        _, windows = self.detect(entities, walls, units=units)
        return windows


def detect_by_layer(entities: list[NormalizedEntity], kind: Literal["door", "window"]) -> list[LineEntity]:
    pred = _is_door_layer if kind == "door" else _is_window_layer
    return [
        line
        for e in entities
        if pred(e.layer) and e.source_insert_id is None
        for line in _entity_to_lines(e)
    ]


def _is_bound_opening(opening: Door | Window) -> bool:
    return opening.wall_id is not None or bool(opening.support_wall_ids)


def _detect_door_lines(entities: list[NormalizedEntity], context: OpeningContext) -> list[LineEntity]:
    insert_doors = detect_by_insert_hint(context, "door")
    existing_ids = {entity.id for entity in insert_doors}
    gap_doors = [entity for entity in detect_doors_from_gaps(context) if entity.id not in existing_ids]
    return _dedupe([
        *detect_by_layer(entities, "door"),
        *insert_doors,
        *gap_doors,
        *detect_doors_from_text_hints(context, existing_ids | {entity.id for entity in gap_doors}),
    ])


def _detect_window_lines(entities: list[NormalizedEntity], context: OpeningContext) -> list[LineEntity]:
    insert_windows = detect_by_insert_hint(context, "window")
    existing_ids = {entity.id for entity in insert_windows}
    gap_windows = [entity for entity in detect_windows_from_gaps(context) if entity.id not in existing_ids]
    return _dedupe([
        *detect_by_layer(entities, "window"),
        *insert_windows,
        *gap_windows,
        *detect_windows_from_text_hints(context, existing_ids | {entity.id for entity in gap_windows}),
    ])


def detect_by_insert_hint(context: OpeningContext, kind: Literal["door", "window"]) -> list[LineEntity]:
    inserts = _top_level_insert_hints(context, kind)
    if not inserts:
        return []

    kind_segments = context.direct_door_segments if kind == "door" else context.direct_window_segments
    insert_by_id = {entity.id: entity for entity in inserts}
    gap_pairs = _pair_insert_hints_to_gaps(context, kind, insert_by_id)
    resolved_lines: dict[str, LineEntity] = {}
    inferred_lengths: list[float] = []

    if kind == "door":
        for insert in inserts:
            geometry = context.door_block_geometries.get(insert.id)
            if geometry is None:
                continue
            resolved_lines[insert.id] = geometry.closed_line
            inferred_lengths.append(_dist(geometry.closed_line.start, geometry.closed_line.end))
    for insert, gap in gap_pairs:
        if insert.id in resolved_lines:
            continue
        segment = _match_segment(
            insert.insert,
            context.header_segments,
            preferred_orientation=_orientation(gap.start, gap.end, context.thresholds),
            thresholds=context.thresholds,
        ) or _match_segment(
            insert.insert,
            context.opening_segments + kind_segments,
            preferred_orientation=_orientation(gap.start, gap.end, context.thresholds),
            thresholds=context.thresholds,
        )

        if segment is not None:
            line = _segment_to_line(segment, insert.id)
            if not _looks_like_symbolic_span(line, gap):
                resolved_lines[insert.id] = line
                inferred_lengths.append(_dist(line.start, line.end))
                continue

        if gap.length <= _max_fallback_gap_span(kind, context.thresholds):
            inferred_lengths.append(gap.length)

    fallback_length = _fallback_insert_span(kind, inferred_lengths, context.thresholds)

    for insert, gap in gap_pairs:
        if insert.id in resolved_lines:
            continue
        resolved_lines[insert.id] = _gap_to_line(
            gap,
            insert,
            kind=kind,
            fallback_length=fallback_length,
            thresholds=context.thresholds,
        )

    for insert in inserts:
        if insert.id in resolved_lines:
            continue
        segment = _match_segment(insert.insert, context.header_segments, thresholds=context.thresholds) or _match_segment(
            insert.insert,
            context.opening_segments + kind_segments,
            thresholds=context.thresholds,
        )
        if segment is not None:
            line = _segment_to_line(segment, insert.id)
            if kind == "window":
                geometry_line = context.window_block_lines.get(insert.id)
                line_bound = _is_bindable_line(line, context)
                geometry_bound = geometry_line is not None and _is_bindable_line(geometry_line, context)
                if not line_bound and geometry_bound:
                    resolved_lines[insert.id] = geometry_line
                    continue
            resolved_lines[insert.id] = line
            continue

        if kind == "window":
            geometry_line = context.window_block_lines.get(insert.id)
            if geometry_line is not None and _is_bindable_line(geometry_line, context):
                resolved_lines[insert.id] = geometry_line

    return list(resolved_lines.values())


def detect_doors_from_gaps(context: OpeningContext) -> list[LineEntity]:
    best_matches: dict[str, tuple[float, LineEntity]] = {}

    for gap in context.gaps:
        arc = _nearest_arc(gap, context.arcs, context.thresholds)
        hint = _classify_gap_hint(gap, context)
        if arc is None and (hint is None or hint.kind not in {"door", "sliding_door"}):
            continue

        expected_length = None
        source_id = arc.id if arc is not None else gap.id
        anchor = gap.center
        if hint is not None:
            expected_length = hint.expected_length
            source_id = hint.source_id
            anchor = hint.anchor
        elif arc is not None:
            expected_length = arc.radius
            anchor = arc.center

        preferred_orientation = _orientation(gap.start, gap.end, context.thresholds)
        segment = (
            _match_segment(anchor, context.header_segments, expected_length, preferred_orientation, context.thresholds)
            or _match_segment(anchor, context.opening_segments + context.direct_door_segments, expected_length, preferred_orientation, context.thresholds)
        )
        if segment is None:
            if hint is None or hint.kind not in {"door", "sliding_door"}:
                continue

            if gap.length > _max_fallback_gap_span("door", context.thresholds):
                continue

            line = _gap_hint_to_line(gap, hint, kind="door", thresholds=context.thresholds)
            score = _dist(anchor, gap.center)
            current = best_matches.get(source_id)
            if current is None or score < current[0]:
                best_matches[source_id] = (score, line)
            continue

        if arc is None and expected_length is not None and not _is_precise_match(segment, expected_length, context.thresholds):
            continue

        line = _segment_to_line(segment, source_id)
        score = _dist(anchor, _midpoint(segment.start, segment.end))
        current = best_matches.get(source_id)
        if current is None or score < current[0]:
            best_matches[source_id] = (score, line)

    return [line for _, line in best_matches.values()]


def detect_windows_from_gaps(context: OpeningContext) -> list[LineEntity]:
    best_matches: dict[str, tuple[float, LineEntity]] = {}

    for gap in context.gaps:
        if _nearest_arc(gap, context.arcs, context.thresholds) is not None:
            continue

        hint = _classify_gap_hint(gap, context)
        if hint is None or hint.kind != "window":
            continue

        expected_length = hint.expected_length
        anchor = hint.anchor
        preferred_orientation = _orientation(gap.start, gap.end, context.thresholds)
        segment = (
            _match_segment(anchor, context.header_segments, expected_length, preferred_orientation, context.thresholds)
            or _match_segment(anchor, context.opening_segments + context.direct_window_segments, expected_length, preferred_orientation, context.thresholds)
        )
        if segment is None:
            continue

        if hint.kind == "window":
            line = _segment_to_line(segment, hint.source_id)
            score = _dist(anchor, _midpoint(segment.start, segment.end))
            current = best_matches.get(hint.source_id)
            if current is None or score < current[0]:
                best_matches[hint.source_id] = (score, line)

    return [line for _, line in best_matches.values()]


def detect_doors_from_text_hints(context: OpeningContext, existing_ids: set[str]) -> list[LineEntity]:
    results: list[LineEntity] = []

    for entity in context.texts:
        normalized_text = " ".join(entity.text.strip().upper().split())
        tokens = re.findall(r"[A-Z0-9]+", normalized_text)
        if entity.id in existing_ids or _is_garage_door_label(tokens):
            continue
        if not (_is_swing_door_label(normalized_text) or _is_sliding_door_label(tokens)):
            continue

        expected_length = _parse_label_width(normalized_text)
        nearest_arc = _nearest_arc_to_point(entity.insert, context.arcs, context.thresholds, expected_length)
        preferred_orientation = _infer_segment_orientation(entity.insert, nearest_arc, context.thresholds)
        label_axis_overhang = _label_axis_overhang(expected_length, context.thresholds)
        label_perp_limit = _label_perpendicular_tolerance(expected_length, context.thresholds)

        segment = (
            _match_segment(
                entity.insert,
                context.header_segments,
                expected_length,
                preferred_orientation,
                context.thresholds,
                axis_overhang_limit=label_axis_overhang,
                perp_dist_limit=label_perp_limit,
            )
            or _match_segment(
                entity.insert,
                context.opening_segments + context.direct_door_segments,
                expected_length,
                preferred_orientation,
                context.thresholds,
                axis_overhang_limit=label_axis_overhang,
                perp_dist_limit=label_perp_limit,
            )
        )
        if segment is not None:
            if nearest_arc is None and expected_length is not None and not _is_precise_match(segment, expected_length, context.thresholds):
                segment = None

        if segment is not None:
            results.append(_segment_to_line(segment, entity.id))
            continue

        gap = _nearest_gap_to_text_hint(entity, context.gaps, context.thresholds, expected_length)
        if gap is None:
            continue

        hint = GapHint(
            kind="sliding_door" if _is_sliding_door_label(tokens) else "door",
            source_id=entity.id,
            anchor=entity.insert,
            expected_length=expected_length,
        )
        results.append(_gap_hint_to_line(gap, hint, kind="door", thresholds=context.thresholds))

    return results


def detect_windows_from_text_hints(context: OpeningContext, existing_ids: set[str]) -> list[LineEntity]:
    results: list[LineEntity] = []

    for entity in context.texts:
        normalized_text = " ".join(entity.text.strip().upper().split())
        tokens = re.findall(r"[A-Z0-9]+", normalized_text)
        if entity.id in existing_ids or not any(token in tokens for token in OPENING_DETECTOR_CONFIG.window_operation_tokens):
            continue

        expected_length = _parse_label_width(normalized_text)
        label_axis_overhang = _label_axis_overhang(expected_length, context.thresholds)
        label_perp_limit = _label_perpendicular_tolerance(expected_length, context.thresholds)
        segment = (
            _match_segment(
                entity.insert,
                context.header_segments,
                expected_length,
                thresholds=context.thresholds,
                axis_overhang_limit=label_axis_overhang,
                perp_dist_limit=label_perp_limit,
            )
            or _match_segment(
                entity.insert,
                context.opening_segments + context.direct_window_segments,
                expected_length,
                thresholds=context.thresholds,
                axis_overhang_limit=label_axis_overhang,
                perp_dist_limit=label_perp_limit,
            )
        )
        if segment is None:
            continue

        results.append(_segment_to_line(segment, entity.id))

    return results


def _detect_gaps(walls: list[Wall], thresholds: OpeningThresholds) -> list[Gap]:
    direction_groups: dict[tuple[float, float], list[tuple[float, Wall, Point, Point]]] = {}

    for wall in walls:
        direction = _unit_vector(wall.start, wall.end)
        if direction is None:
            continue

        normal = Point(x=-direction.y, y=direction.x)
        line_offset = _project(wall.start, Point(0.0, 0.0), normal)
        direction_key = (
            round(direction.x, OPENING_DETECTOR_CONFIG.gap_group_direction_precision),
            round(direction.y, OPENING_DETECTOR_CONFIG.gap_group_direction_precision),
        )
        direction_groups.setdefault(direction_key, []).append((line_offset, wall, wall.start, direction))

    gaps: list[Gap] = []

    for grouped in direction_groups.values():
        if len(grouped) < 2:
            continue

        grouped.sort(key=lambda item: item[0])
        offset_clusters: list[list[tuple[float, Wall, Point, Point]]] = [[grouped[0]]]

        for item in grouped[1:]:
            if abs(item[0] - offset_clusters[-1][-1][0]) <= thresholds.gap_axis_offset_tolerance:
                offset_clusters[-1].append(item)
            else:
                offset_clusters.append([item])

        for cluster in offset_clusters:
            if len(cluster) < 2:
                continue

            axis_origin = cluster[0][2]
            direction = cluster[0][3]
            normal = Point(x=-direction.y, y=direction.x)

            intervals = []
            for _, wall, _, _ in cluster:
                start_t = _project(wall.start, axis_origin, direction)
                end_t = _project(wall.end, axis_origin, direction)
                intervals.append((wall, min(start_t, end_t), max(start_t, end_t)))

            intervals.sort(key=lambda item: (item[1], item[2]))

            current_wall, _, current_end = intervals[0]
            for next_wall, next_start, next_end in intervals[1:]:
                gap_length = next_start - current_end
                if gap_length >= thresholds.min_opening_length:
                    gap_start = _point_along(axis_origin, direction, current_end)
                    gap_end = _point_along(axis_origin, direction, next_start)
                    gaps.append(
                        Gap(
                            id=f"{current_wall.id}:{next_wall.id}",
                            support_wall_ids=(current_wall.id, next_wall.id),
                            start=gap_start,
                            end=gap_end,
                            center=_midpoint(gap_start, gap_end),
                            direction=direction,
                            normal=normal,
                            length=round(gap_length, 6),
                        )
                    )

                if next_end > current_end:
                    current_wall, current_end = next_wall, next_end

    return gaps


def _segments_inside_gap(gap: Gap, segments: list[Segment], thresholds: OpeningThresholds) -> list[Segment]:
    matching_segments: list[Segment] = []

    gap_start_t = _project(gap.start, gap.start, gap.direction)
    gap_end_t = _project(gap.end, gap.start, gap.direction)
    lower_t = min(gap_start_t, gap_end_t)
    upper_t = max(gap_start_t, gap_end_t)

    for segment in segments:
        orientation = _orientation(segment.start, segment.end, thresholds)
        gap_orientation = _orientation(gap.start, gap.end, thresholds)
        if orientation is None or gap_orientation is None or orientation != gap_orientation:
            continue

        midpoint = _midpoint(segment.start, segment.end)
        midpoint_normal = abs(_project(midpoint, gap.center, gap.normal))
        if midpoint_normal > thresholds.perp_dist_max:
            continue

        segment_start_t = _project(segment.start, gap.start, gap.direction)
        segment_end_t = _project(segment.end, gap.start, gap.direction)
        seg_lower = min(segment_start_t, segment_end_t)
        seg_upper = max(segment_start_t, segment_end_t)

        overlap = min(seg_upper, upper_t) - max(seg_lower, lower_t)
        if overlap > 0.0:
            matching_segments.append(segment)

    return matching_segments


def _classify_gap_hint(gap: Gap, context: OpeningContext) -> GapHint | None:
    block_hint = _nearest_block_hint(gap, context)
    if block_hint is not None:
        return block_hint

    text_hint = _nearest_text_hint(gap, context.texts, context.thresholds)
    if text_hint is not None:
        return text_hint

    door_segments = _segments_inside_gap(gap, context.direct_door_segments, context.thresholds)
    if door_segments:
        return GapHint(kind="door", source_id=door_segments[0].source_entity_id, anchor=_midpoint(door_segments[0].start, door_segments[0].end))

    window_segments = _segments_inside_gap(gap, context.direct_window_segments, context.thresholds)
    if window_segments:
        return GapHint(kind="window", source_id=window_segments[0].source_entity_id, anchor=_midpoint(window_segments[0].start, window_segments[0].end))

    return None


def _nearest_arc(gap: Gap, arcs: list[ArcEntity], thresholds: OpeningThresholds) -> ArcEntity | None:
    nearby = []
    gap_start_t = _project(gap.start, gap.start, gap.direction)
    gap_end_t = _project(gap.end, gap.start, gap.direction)
    lower_t = min(gap_start_t, gap_end_t)
    upper_t = max(gap_start_t, gap_end_t)

    for arc in arcs:
        axis_t = _project(arc.center, gap.start, gap.direction)
        normal_t = abs(_project(arc.center, gap.center, gap.normal))
        if axis_t < lower_t - thresholds.gap_axis_overhang or axis_t > upper_t + thresholds.gap_axis_overhang:
            continue
        if normal_t > max(thresholds.gap_arc_distance, arc.radius + thresholds.arc_search_radius):
            continue
        nearby.append(arc)

    if not nearby:
        return None
    return min(
        nearby,
        key=lambda arc: (
            abs(_project(arc.center, gap.center, gap.direction)),
            abs(_project(arc.center, gap.center, gap.normal)),
        ),
    )


def _nearest_arc_to_point(
    anchor: Point,
    arcs: list[ArcEntity],
    thresholds: OpeningThresholds,
    expected_length: float | None = None,
) -> ArcEntity | None:
    radius = max(thresholds.arc_search_radius, (expected_length or 0.0) * 1.25)
    nearby = [arc for arc in arcs if _dist(anchor, arc.center) <= max(radius, arc.radius + thresholds.arc_search_radius)]
    if not nearby:
        return None
    return min(nearby, key=lambda arc: _dist(anchor, arc.center))


def _nearest_block_hint(gap: Gap, context: OpeningContext) -> GapHint | None:
    candidates: list[tuple[float, GapHint]] = []

    for entity in context.inserts:
        kind = _insert_hint_kind(entity, context.insert_content_ids)
        if kind is None:
            continue

        axis_distance = abs(_project(entity.insert, gap.center, gap.direction))
        normal_distance = abs(_project(entity.insert, gap.center, gap.normal))
        if axis_distance > gap.length / 2.0 + context.thresholds.gap_axis_overhang:
            continue
        if normal_distance > context.thresholds.insert_hint_normal_distance:
            continue

        candidates.append((normal_distance + axis_distance, GapHint(kind=kind, source_id=entity.id, anchor=entity.insert)))

    if not candidates:
        return None

    _, hint = min(candidates, key=lambda item: item[0])
    return hint


def _nearest_text_hint(gap: Gap, texts: list[TextEntity], thresholds: OpeningThresholds) -> GapHint | None:
    candidates: list[tuple[float, GapHint]] = []

    for entity in texts:
        normalized_text = " ".join(entity.text.strip().upper().split())
        tokens = re.findall(r"[A-Z0-9]+", normalized_text)
        if _is_garage_door_label(tokens):
            continue

        expected_length = _parse_label_width(normalized_text)
        normal_limit = max(thresholds.gap_hint_distance, (expected_length or 0.0) * 0.2)

        axis_distance = abs(_project(entity.insert, gap.center, gap.direction))
        normal_distance = abs(_project(entity.insert, gap.center, gap.normal))
        if axis_distance > gap.length / 2.0 + thresholds.gap_axis_overhang:
            continue
        if normal_distance > normal_limit:
            continue

        if _is_sliding_door_label(tokens) or _is_swing_door_label(normalized_text):
            candidates.append((
                normal_distance + axis_distance,
                GapHint(
                    kind="sliding_door" if _is_sliding_door_label(tokens) else "door",
                    source_id=entity.id,
                    anchor=entity.insert,
                    expected_length=expected_length,
                ),
            ))
        elif any(token in tokens for token in OPENING_DETECTOR_CONFIG.window_operation_tokens):
            candidates.append((
                normal_distance + axis_distance,
                GapHint(
                    kind="window",
                    source_id=entity.id,
                    anchor=entity.insert,
                    expected_length=expected_length,
                ),
            ))

    if not candidates:
        return None

    _, hint = min(candidates, key=lambda item: item[0])
    return hint


def _nearest_gap_to_text_hint(
    text_entity: TextEntity,
    gaps: list[Gap],
    thresholds: OpeningThresholds,
    expected_length: float | None = None,
) -> Gap | None:
    normalized_text = " ".join(text_entity.text.strip().upper().split())
    tokens = re.findall(r"[A-Z0-9]+", normalized_text)
    if not (_is_swing_door_label(normalized_text) or _is_sliding_door_label(tokens)):
        return None

    normal_limit = max(thresholds.gap_hint_distance, (expected_length or 0.0) * 0.75)
    axis_limit_extra = max(thresholds.gap_axis_overhang, (expected_length or 0.0) * 0.25)
    candidates: list[tuple[float, float, float, Gap]] = []

    for gap in gaps:
        axis_distance = abs(_project(text_entity.insert, gap.center, gap.direction))
        normal_distance = abs(_project(text_entity.insert, gap.center, gap.normal))
        if axis_distance > gap.length / 2.0 + axis_limit_extra:
            continue
        if normal_distance > normal_limit:
            continue

        length_penalty = abs((expected_length or gap.length) - gap.length)
        candidates.append((normal_distance + axis_distance, length_penalty, axis_distance, gap))

    if not candidates:
        return None

    _, _, _, gap = min(candidates, key=lambda item: (item[0], item[1], item[2], item[3].id))
    return gap


def _midpoint(a: Point, b: Point) -> Point:
    return Point(x=round((a.x + b.x) / 2, 6), y=round((a.y + b.y) / 2, 6))


def _dist(a: Point, b: Point) -> float:
    return ((b.x - a.x) ** 2 + (b.y - a.y) ** 2) ** 0.5


def _orientation(a: Point, b: Point, thresholds: OpeningThresholds) -> Orientation | None:
    if abs(a.y - b.y) <= thresholds.orientation_axis_tolerance:
        return "horizontal"
    if abs(a.x - b.x) <= thresholds.orientation_axis_tolerance:
        return "vertical"
    return None


def _is_opening_layer(layer: str) -> bool:
    normalized = layer.strip().lower()
    return any(keyword in normalized for keyword in OPENING_DETECTOR_CONFIG.opening_layer_keywords) and not any(
        keyword in normalized for keyword in OPENING_DETECTOR_CONFIG.garage_layer_keywords
    )


def _is_header_layer(layer: str) -> bool:
    normalized = layer.strip().lower()
    return any(keyword in normalized for keyword in OPENING_DETECTOR_CONFIG.header_layer_keywords) and not any(
        keyword in normalized for keyword in OPENING_DETECTOR_CONFIG.garage_layer_keywords
    )


def _is_door_layer(layer: str) -> bool:
    normalized = layer.strip().lower()
    return any(keyword in normalized for keyword in OPENING_DETECTOR_CONFIG.door_layer_keywords) and not any(
        keyword in normalized for keyword in OPENING_DETECTOR_CONFIG.garage_layer_keywords
    )


def _is_window_layer(layer: str) -> bool:
    normalized = layer.strip().lower()
    return any(keyword in normalized for keyword in OPENING_DETECTOR_CONFIG.window_layer_keywords)


def _explicit_insert_hint_kind(entity: InsertEntity) -> Literal["door", "window"] | None:
    normalized_name = entity.block_name.strip().lower().replace("_", "-")
    if any(keyword in normalized_name for keyword in OPENING_DETECTOR_CONFIG.garage_layer_keywords):
        return None
    if _matches_block_keywords(normalized_name, OPENING_DETECTOR_CONFIG.door_block_keywords):
        return "door"
    if _matches_block_keywords(normalized_name, OPENING_DETECTOR_CONFIG.window_block_keywords):
        return "window"
    return None


def _insert_hint_kind(entity: InsertEntity, insert_content_ids: frozenset[str]) -> Literal["door", "window"] | None:
    explicit_kind = _explicit_insert_hint_kind(entity)
    if explicit_kind is not None:
        return explicit_kind
    if entity.id not in insert_content_ids:
        return None
    if _is_door_layer(entity.layer):
        return "door"
    if _is_window_layer(entity.layer):
        return "window"
    return None


def _matches_block_keywords(normalized_name: str, keywords: tuple[str, ...]) -> bool:
    tokens = set(re.findall(r"[a-zа-я0-9]+", normalized_name))
    for keyword in keywords:
        normalized_keyword = keyword.strip().lower().replace("_", "-")
        if len(normalized_keyword) <= 2:
            if normalized_keyword in tokens:
                return True
            continue
        if normalized_keyword in tokens or normalized_keyword in normalized_name:
            return True
    return False


def _entity_to_segments(entity: NormalizedEntity) -> list[Segment]:
    if isinstance(entity, LineEntity):
        return [Segment(entity.start, entity.end, entity.id, entity.layer)]

    if not isinstance(entity, PolylineEntity) or len(entity.points) < 2:
        return []

    segments = [
        Segment(start, end, f"{entity.id}:{index + 1}", entity.layer)
        for index, (start, end) in enumerate(zip(entity.points, entity.points[1:]))
    ]
    if entity.closed:
        segments.append(Segment(entity.points[-1], entity.points[0], f"{entity.id}:closing", entity.layer))

    return segments


def _segment_to_line(seg: Segment, source_id: str) -> LineEntity:
    return LineEntity(
        id=source_id,
        layer=seg.layer,
        start=Point(x=round(seg.start.x, 6), y=round(seg.start.y, 6)),
        end=Point(x=round(seg.end.x, 6), y=round(seg.end.y, 6)),
    )


def _line_to_door(entity: LineEntity, context: OpeningContext) -> Door:
    host_wall, support_wall_ids = _resolve_wall_binding(entity, context)
    wall_id = _resolve_host_wall_id(host_wall, support_wall_ids, context)
    opens_towards_wall_side, swing, hinge_side = _resolve_door_opening(entity, context, host_wall)
    return Door(
        id=entity.id,
        layer=entity.layer,
        start=entity.start,
        end=entity.end,
        length=round(_dist(entity.start, entity.end), 6),
        wall_id=wall_id,
        support_wall_ids=support_wall_ids,
        opens_towards_wall_side=opens_towards_wall_side,
        swing=swing,
        hinge_side=hinge_side,
        source_entity_ids=[entity.id],
    )


def _line_to_window(entity: LineEntity, context: OpeningContext) -> Window:
    host_wall, support_wall_ids = _resolve_wall_binding(entity, context)
    wall_id = _resolve_host_wall_id(host_wall, support_wall_ids, context)
    return Window(
        id=entity.id,
        layer=entity.layer,
        start=entity.start,
        end=entity.end,
        length=round(_dist(entity.start, entity.end), 6),
        wall_id=wall_id,
        support_wall_ids=support_wall_ids,
        source_entity_ids=[entity.id],
    )


def _entity_to_lines(entity: NormalizedEntity) -> list[LineEntity]:
    if isinstance(entity, LineEntity):
        return [entity]
    if isinstance(entity, PolylineEntity):
        return [_segment_to_line(segment, segment.source_entity_id) for segment in _entity_to_segments(entity)]
    return []


def _collect_segments(entities: list[NormalizedEntity], layer_pred) -> list[Segment]:
    return [segment for entity in entities if layer_pred(entity.layer) for segment in _entity_to_segments(entity)]


def _top_level_insert_hints(context: OpeningContext, kind: Literal["door", "window"]) -> list[InsertEntity]:
    return [
        entity
        for entity in context.inserts
        if entity.source_insert_id is None and _insert_hint_kind(entity, context.insert_content_ids) == kind
    ]


def _pair_insert_hints_to_gaps(
    context: OpeningContext,
    kind: Literal["door", "window"],
    insert_by_id: dict[str, InsertEntity],
) -> list[tuple[InsertEntity, Gap]]:
    candidates: list[tuple[float, float, str, str]] = []

    for insert in insert_by_id.values():
        for gap in context.gaps:
            axis_distance = abs(_project(insert.insert, gap.center, gap.direction))
            normal_distance = abs(_project(insert.insert, gap.center, gap.normal))
            max_axis_distance = gap.length / 2.0 + max(
                context.thresholds.gap_axis_overhang,
                context.thresholds.segment_search_radius,
            )
            if axis_distance > max_axis_distance or normal_distance > context.thresholds.insert_hint_normal_distance:
                continue
            candidates.append((normal_distance + axis_distance, axis_distance, insert.id, gap.id))

    gaps_by_id = {gap.id: gap for gap in context.gaps}
    assigned_inserts: set[str] = set()
    assigned_gaps: set[str] = set()
    pairs: list[tuple[InsertEntity, Gap]] = []

    for _, _, insert_id, gap_id in sorted(candidates, key=lambda item: (item[0], item[1], item[2], item[3])):
        if insert_id in assigned_inserts or gap_id in assigned_gaps:
            continue
        insert = insert_by_id[insert_id]
        if _insert_hint_kind(insert, context.insert_content_ids) != kind:
            continue
        pairs.append((insert, gaps_by_id[gap_id]))
        assigned_inserts.add(insert_id)
        assigned_gaps.add(gap_id)

    return pairs


def _fallback_insert_span(
    kind: Literal["door", "window"],
    inferred_lengths: list[float],
    thresholds: OpeningThresholds,
) -> float:
    if inferred_lengths:
        return float(median(inferred_lengths))
    return thresholds.default_door_span if kind == "door" else thresholds.default_window_span


def _max_fallback_gap_span(kind: Literal["door", "window"], thresholds: OpeningThresholds) -> float:
    return thresholds.max_fallback_door_span if kind == "door" else thresholds.max_fallback_window_span


def _gap_to_line(
    gap: Gap,
    insert: InsertEntity,
    *,
    kind: Literal["door", "window"],
    fallback_length: float,
    thresholds: OpeningThresholds,
) -> LineEntity:
    line_length = gap.length
    if gap.length > _max_fallback_gap_span(kind, thresholds):
        line_length = min(gap.length, fallback_length)

    center_t = _project(insert.insert, gap.start, gap.direction)
    lower_t = max(0.0, center_t - line_length / 2.0)
    upper_t = min(gap.length, center_t + line_length / 2.0)
    if upper_t - lower_t < line_length:
        if lower_t <= OPENING_DETECTOR_CONFIG.vector_epsilon:
            upper_t = min(gap.length, line_length)
        elif upper_t >= gap.length - OPENING_DETECTOR_CONFIG.vector_epsilon:
            lower_t = max(0.0, gap.length - line_length)

    return LineEntity(
        id=insert.id,
        layer=insert.layer,
        start=_point_along(gap.start, gap.direction, lower_t),
        end=_point_along(gap.start, gap.direction, upper_t),
    )


def _gap_hint_to_line(
    gap: Gap,
    hint: GapHint,
    *,
    kind: Literal["door", "window"],
    thresholds: OpeningThresholds,
) -> LineEntity:
    line_length = gap.length
    if hint.expected_length is not None:
        line_length = min(gap.length, hint.expected_length)
    elif gap.length > _max_fallback_gap_span(kind, thresholds):
        line_length = _max_fallback_gap_span(kind, thresholds)

    center_t = _project(hint.anchor, gap.start, gap.direction)
    lower_t = max(0.0, center_t - line_length / 2.0)
    upper_t = min(gap.length, center_t + line_length / 2.0)
    if upper_t - lower_t < line_length:
        if lower_t <= OPENING_DETECTOR_CONFIG.vector_epsilon:
            upper_t = min(gap.length, line_length)
        elif upper_t >= gap.length - OPENING_DETECTOR_CONFIG.vector_epsilon:
            lower_t = max(0.0, gap.length - line_length)

    return LineEntity(
        id=hint.source_id,
        layer="opening",
        start=_point_along(gap.start, gap.direction, lower_t),
        end=_point_along(gap.start, gap.direction, upper_t),
    )


def _looks_like_symbolic_span(line: LineEntity, gap: Gap) -> bool:
    line_length = _dist(line.start, line.end)
    if line_length <= OPENING_DETECTOR_CONFIG.vector_epsilon:
        return True
    return gap.length > line_length * 1.5


def _resolve_wall_binding(entity: LineEntity, context: OpeningContext) -> tuple[Wall | None, tuple[str, ...]]:
    mapped_gap = _gap_for_insert_entity(entity.id, context)
    if mapped_gap is not None:
        support_walls = _support_walls_for_gap(mapped_gap, context.walls)
        host_wall = _select_host_wall(support_walls) if support_walls else None
        return host_wall, mapped_gap.support_wall_ids

    gap = _nearest_gap_for_segment(entity, context.gaps, context.thresholds)
    if gap is not None:
        support_walls = _support_walls_for_gap(gap, context.walls)
        host_wall = _select_host_wall(support_walls) if support_walls else None
        return host_wall, gap.support_wall_ids

    line_direction = _unit_vector(entity.start, entity.end)
    if line_direction is None:
        return None, ()

    normal = Point(x=-line_direction.y, y=line_direction.x)
    midpoint = _midpoint(entity.start, entity.end)
    axis_origin = entity.start
    entity_start_t = _project(entity.start, axis_origin, line_direction)
    entity_end_t = _project(entity.end, axis_origin, line_direction)
    lower_t = min(entity_start_t, entity_end_t)
    upper_t = max(entity_start_t, entity_end_t)

    candidates: list[tuple[float, Wall]] = []
    for wall in context.walls:
        wall_direction = _unit_vector(wall.start, wall.end)
        if wall_direction is None or not _same_axis_direction(line_direction, wall_direction):
            continue

        wall_midpoint = _midpoint(wall.start, wall.end)
        normal_offset = abs(_project(wall_midpoint, midpoint, normal))
        if normal_offset > max(_effective_wall_width(wall), context.thresholds.perp_dist_max):
            continue

        wall_start_t = _project(wall.start, axis_origin, line_direction)
        wall_end_t = _project(wall.end, axis_origin, line_direction)
        wall_lower = min(wall_start_t, wall_end_t)
        wall_upper = max(wall_start_t, wall_end_t)
        axis_gap = max(lower_t - wall_upper, wall_lower - upper_t, 0.0)
        if axis_gap > context.thresholds.segment_search_radius:
            continue

        candidates.append((normal_offset + axis_gap, wall))

    if not candidates:
        return None, ()

    candidates.sort(key=lambda item: (item[0], item[1].id))
    host_wall = candidates[0][1]
    return host_wall, (host_wall.id,)


def _is_bindable_line(entity: LineEntity, context: OpeningContext) -> bool:
    host_wall, support_wall_ids = _resolve_wall_binding(entity, context)
    return host_wall is not None or bool(support_wall_ids)


def _nearest_gap_for_segment(entity: LineEntity, gaps: list[Gap], thresholds: OpeningThresholds) -> Gap | None:
    direction = _unit_vector(entity.start, entity.end)
    if direction is None:
        return None

    midpoint = _midpoint(entity.start, entity.end)
    best_gap: Gap | None = None
    best_score: tuple[float, float] | None = None

    for gap in gaps:
        gap_orientation = _orientation(gap.start, gap.end, thresholds)
        entity_orientation = _orientation(entity.start, entity.end, thresholds)
        if gap_orientation is None or entity_orientation is None or gap_orientation != entity_orientation:
            continue

        axis_distance = abs(_project(midpoint, gap.center, gap.direction))
        normal_distance = abs(_project(midpoint, gap.center, gap.normal))
        if axis_distance > gap.length / 2.0 + thresholds.segment_overhang:
            continue
        if normal_distance > max(thresholds.perp_dist_max, thresholds.segment_search_radius + thresholds.segment_overhang):
            continue

        gap_start_t = _project(gap.start, gap.start, gap.direction)
        gap_end_t = _project(gap.end, gap.start, gap.direction)
        seg_start_t = _project(entity.start, gap.start, gap.direction)
        seg_end_t = _project(entity.end, gap.start, gap.direction)
        overlap = min(max(seg_start_t, seg_end_t), max(gap_start_t, gap_end_t)) - max(min(seg_start_t, seg_end_t), min(gap_start_t, gap_end_t))
        if overlap <= 0.0:
            continue

        score = (normal_distance + axis_distance, abs(gap.length - _dist(entity.start, entity.end)))
        if best_score is None or score < best_score:
            best_gap = gap
            best_score = score

    return best_gap


def _find_wall_by_id(walls: list[Wall], wall_id: str) -> Wall | None:
    for wall in walls:
        if wall.id == wall_id:
            return wall
    return None


def _build_door_block_geometries(
    entities: list[NormalizedEntity],
    inserts: list[InsertEntity],
    insert_content_ids: frozenset[str],
) -> dict[str, DoorBlockGeometry]:
    children_by_insert_id: dict[str, list[NormalizedEntity]] = {}
    for entity in entities:
        if entity.source_insert_id is None:
            continue
        children_by_insert_id.setdefault(entity.source_insert_id, []).append(entity)

    geometries: dict[str, DoorBlockGeometry] = {}
    for insert in inserts:
        if insert.source_insert_id is not None:
            continue
        if _insert_hint_kind(insert, insert_content_ids) != "door":
            continue

        geometry = _extract_door_block_geometry(insert, children_by_insert_id.get(insert.id, []))
        if geometry is not None:
            geometries[insert.id] = geometry

    return geometries


def _build_window_block_lines(
    entities: list[NormalizedEntity],
    inserts: list[InsertEntity],
    insert_content_ids: frozenset[str],
) -> dict[str, LineEntity]:
    children_by_insert_id: dict[str, list[NormalizedEntity]] = {}
    for entity in entities:
        if entity.source_insert_id is None:
            continue
        children_by_insert_id.setdefault(entity.source_insert_id, []).append(entity)

    lines: dict[str, LineEntity] = {}
    for insert in inserts:
        if insert.source_insert_id is not None:
            continue
        if _insert_hint_kind(insert, insert_content_ids) != "window":
            continue

        line = _extract_window_block_line(insert, children_by_insert_id.get(insert.id, []))
        if line is not None:
            lines[insert.id] = line

    return lines


def _extract_door_block_geometry(
    insert: InsertEntity,
    children: list[NormalizedEntity],
) -> DoorBlockGeometry | None:
    segments = []
    for child in children:
        if not isinstance(child, PolylineEntity) or len(child.points) < 2:
            continue
        start = child.points[0]
        end = child.points[-1]
        length = _dist(start, end)
        if length <= OPENING_DETECTOR_CONFIG.vector_epsilon:
            continue
        direction = _unit_vector(start, end)
        if direction is None:
            continue
        segments.append((start, end, length, (round(direction.x, 3), round(direction.y, 3))))

    if not segments:
        return None

    direction_groups: dict[tuple[float, float], list[tuple[Point, Point, float, tuple[float, float]]]] = {}
    for segment in segments:
        direction_groups.setdefault(segment[3], []).append(segment)

    line_segments = max(
        direction_groups.values(),
        key=lambda group: (sum(segment[2] for segment in group), len(group)),
    )
    if sum(segment[2] for segment in line_segments) <= OPENING_DETECTOR_CONFIG.vector_epsilon:
        return None

    line_endpoints = _merged_segment_endpoints(line_segments)
    if line_endpoints is None:
        return None
    line_start, line_end = line_endpoints

    non_line_segments = [segment for segment in segments if segment not in line_segments]
    non_line_endpoint_counts = _segment_endpoint_counts(non_line_segments)

    line_start_count = non_line_endpoint_counts.get(_snap_point(line_start), 0)
    line_end_count = non_line_endpoint_counts.get(_snap_point(line_end), 0)
    if line_start_count and not line_end_count:
        open_point = line_start
        hinge_point = line_end
    elif line_end_count and not line_start_count:
        open_point = line_end
        hinge_point = line_start
    else:
        return None

    closed_points = [
        Point(x=point[0], y=point[1])
        for point, count in non_line_endpoint_counts.items()
        if count == 1 and not _same_point_coords(point, open_point)
    ]
    if len(closed_points) != 1:
        return None

    closed_free_point = closed_points[0]
    closed_line_start, closed_line_end = _canonical_segment_points(hinge_point, closed_free_point)
    hinge_side = "start" if _same_point(hinge_point, closed_line_start) else "end"

    return DoorBlockGeometry(
        closed_line=LineEntity(
            id=insert.id,
            layer=insert.layer,
            start=closed_line_start,
            end=closed_line_end,
        ),
        hinge_point=hinge_point,
        open_point=open_point,
        hinge_side=hinge_side,
    )


def _extract_window_block_line(
    insert: InsertEntity,
    children: list[NormalizedEntity],
) -> LineEntity | None:
    points: list[Point] = []
    for child in children:
        for segment in _entity_to_segments(child):
            if _dist(segment.start, segment.end) <= OPENING_DETECTOR_CONFIG.vector_epsilon:
                continue
            points.extend((segment.start, segment.end))

    if len(points) < 2:
        return None

    min_x = min(point.x for point in points)
    max_x = max(point.x for point in points)
    min_y = min(point.y for point in points)
    max_y = max(point.y for point in points)
    span_x = max_x - min_x
    span_y = max_y - min_y

    if max(span_x, span_y) <= OPENING_DETECTOR_CONFIG.vector_epsilon:
        return None

    if span_x >= span_y:
        center_y = (min_y + max_y) / 2.0
        start = Point(x=min_x, y=center_y)
        end = Point(x=max_x, y=center_y)
    else:
        center_x = (min_x + max_x) / 2.0
        start = Point(x=center_x, y=min_y)
        end = Point(x=center_x, y=max_y)

    return LineEntity(
        id=insert.id,
        layer=insert.layer,
        start=start,
        end=end,
    )


def _merged_segment_endpoints(
    segments: list[tuple[Point, Point, float, tuple[float, float]]],
) -> tuple[Point, Point] | None:
    unique_points: list[Point] = []
    for start, end, _, _ in segments:
        if not any(_same_point(start, point) for point in unique_points):
            unique_points.append(start)
        if not any(_same_point(end, point) for point in unique_points):
            unique_points.append(end)

    if len(unique_points) < 2:
        return None

    best_pair: tuple[Point, Point] | None = None
    best_distance = -1.0
    for index, left in enumerate(unique_points):
        for right in unique_points[index + 1 :]:
            distance = _dist(left, right)
            if distance > best_distance:
                best_distance = distance
                best_pair = (left, right)

    return best_pair


def _segment_endpoint_counts(
    segments: list[tuple[Point, Point, float, tuple[float, float]]],
) -> dict[tuple[float, float], int]:
    endpoint_counts: dict[tuple[float, float], int] = {}
    for start, end, _, _ in segments:
        endpoint_counts[_snap_point(start)] = endpoint_counts.get(_snap_point(start), 0) + 1
        endpoint_counts[_snap_point(end)] = endpoint_counts.get(_snap_point(end), 0) + 1
    return endpoint_counts


def _canonical_segment_points(start: Point, end: Point) -> tuple[Point, Point]:
    dx = end.x - start.x
    dy = end.y - start.y
    epsilon = OPENING_DETECTOR_CONFIG.vector_epsilon
    if dx > epsilon:
        return start, end
    if dx < -epsilon:
        return end, start
    if dy >= -epsilon:
        return start, end
    return end, start


def _snap_point(point: Point) -> tuple[float, float]:
    return (round(point.x, 5), round(point.y, 5))


def _same_point_coords(point: tuple[float, float], reference: Point) -> bool:
    return abs(point[0] - reference.x) <= 1e-5 and abs(point[1] - reference.y) <= 1e-5


def _same_point(left: Point, right: Point) -> bool:
    return _same_point_coords((left.x, left.y), right)


def _build_opening_context(
    entities: list[NormalizedEntity],
    walls: list[Wall],
    thresholds: OpeningThresholds,
) -> OpeningContext:
    inserts = [entity for entity in entities if isinstance(entity, InsertEntity)]
    insert_content_ids = frozenset({
        entity.source_insert_id
        for entity in entities
        if entity.source_insert_id is not None
    })
    door_block_geometries = _build_door_block_geometries(entities, inserts, insert_content_ids)
    window_block_lines = _build_window_block_lines(entities, inserts, insert_content_ids)
    base_context = OpeningContext(
        walls=walls,
        opening_segments=_collect_segments(entities, _is_opening_layer),
        header_segments=_collect_segments(entities, _is_header_layer),
        direct_door_segments=_collect_segments(entities, _is_door_layer),
        direct_window_segments=_collect_segments(entities, _is_window_layer),
        arcs=[entity for entity in entities if isinstance(entity, ArcEntity) and _is_opening_layer(entity.layer)],
        texts=[entity for entity in entities if isinstance(entity, TextEntity)],
        inserts=inserts,
        insert_content_ids=insert_content_ids,
        door_block_geometries=door_block_geometries,
        window_block_lines=window_block_lines,
        insert_gap_ids={},
        gaps=_detect_gaps(walls, thresholds),
        wall_host_ids={},
        thresholds=thresholds,
    )
    return OpeningContext(
        walls=base_context.walls,
        opening_segments=base_context.opening_segments,
        header_segments=base_context.header_segments,
        direct_door_segments=base_context.direct_door_segments,
        direct_window_segments=base_context.direct_window_segments,
        arcs=base_context.arcs,
        texts=base_context.texts,
        inserts=base_context.inserts,
        insert_content_ids=base_context.insert_content_ids,
        door_block_geometries=base_context.door_block_geometries,
        window_block_lines=base_context.window_block_lines,
        insert_gap_ids=_build_insert_gap_ids(base_context),
        gaps=base_context.gaps,
        wall_host_ids=base_context.wall_host_ids,
        thresholds=base_context.thresholds,
    )


def _with_wall_host_ids(context: OpeningContext, openings: list[LineEntity]) -> OpeningContext:
    return OpeningContext(
        walls=context.walls,
        opening_segments=context.opening_segments,
        header_segments=context.header_segments,
        direct_door_segments=context.direct_door_segments,
        direct_window_segments=context.direct_window_segments,
        arcs=context.arcs,
        texts=context.texts,
        inserts=context.inserts,
        insert_content_ids=context.insert_content_ids,
        door_block_geometries=context.door_block_geometries,
        window_block_lines=context.window_block_lines,
        insert_gap_ids=context.insert_gap_ids,
        gaps=context.gaps,
        wall_host_ids=_build_wall_host_ids(context, openings),
        thresholds=context.thresholds,
    )


def _build_wall_host_ids(context: OpeningContext, openings: list[LineEntity]) -> dict[str, str]:
    parents = {wall.id: wall.id for wall in context.walls}
    gaps_by_id = {gap.id: gap for gap in context.gaps}

    def find(wall_id: str) -> str:
        parent = parents[wall_id]
        if parent != wall_id:
            parents[wall_id] = find(parent)
        return parents[wall_id]

    def union(left_id: str, right_id: str) -> None:
        left_root = find(left_id)
        right_root = find(right_id)
        if left_root == right_root:
            return
        if left_root < right_root:
            parents[right_root] = left_root
        else:
            parents[left_root] = right_root

    for wall in context.walls:
        if wall.run_id is not None and wall.run_id != wall.id and wall.run_id in parents:
            union(wall.id, wall.run_id)

    for opening in openings:
        mapped_gap_id = context.insert_gap_ids.get(opening.id)
        gap = gaps_by_id.get(mapped_gap_id) if mapped_gap_id is not None else _nearest_gap_for_segment(opening, context.gaps, context.thresholds)
        if gap is not None:
            union(gap.support_wall_ids[0], gap.support_wall_ids[1])

    groups: dict[str, list[Wall]] = {}
    for wall in context.walls:
        groups.setdefault(find(wall.id), []).append(wall)

    host_ids: dict[str, str] = {}
    for walls in groups.values():
        representative = _select_host_wall(walls)
        for wall in walls:
            host_ids[wall.id] = representative.id

    return host_ids


def _build_insert_gap_ids(context: OpeningContext) -> dict[str, str]:
    insert_gap_ids: dict[str, str] = {}
    for kind in ("door", "window"):
        insert_by_id = {
            entity.id: entity
            for entity in _top_level_insert_hints(context, kind)
        }
        for insert, gap in _pair_insert_hints_to_gaps(context, kind, insert_by_id):
            insert_gap_ids[insert.id] = gap.id
    return insert_gap_ids


def _select_host_wall(walls: list[Wall]) -> Wall:
    structural_walls = [wall for wall in walls if _effective_wall_width(wall) > 0.0]
    candidates = structural_walls if structural_walls else walls
    return max(candidates, key=lambda wall: (wall.length, wall.id))


def _effective_wall_width(wall: Wall) -> float:
    if wall.geometry_role == "boundary_face":
        return 0.0
    return wall.width


def _support_walls_for_gap(gap: Gap, walls: list[Wall]) -> list[Wall]:
    return [
        wall
        for wall_id in gap.support_wall_ids
        if (wall := _find_wall_by_id(walls, wall_id)) is not None
    ]


def _gap_for_insert_entity(entity_id: str, context: OpeningContext) -> Gap | None:
    gap_id = context.insert_gap_ids.get(entity_id)
    if gap_id is None:
        return None
    for gap in context.gaps:
        if gap.id == gap_id:
            return gap
    return None


def _resolve_host_wall_id(host_wall: Wall | None, support_wall_ids: tuple[str, ...], context: OpeningContext) -> str | None:
    for wall_id in support_wall_ids:
        host_id = context.wall_host_ids.get(wall_id)
        if host_id is not None:
            return host_id

    if host_wall is None:
        return None

    return context.wall_host_ids.get(host_wall.id, host_wall.run_id or host_wall.id)


def _resolve_door_opening(
    entity: LineEntity,
    context: OpeningContext,
    host_wall: Wall | None,
) -> tuple[str | None, str | None, str | None]:
    direction = _unit_vector(
        host_wall.start if host_wall is not None else entity.start,
        host_wall.end if host_wall is not None else entity.end,
    )
    if direction is None:
        return None, None, None

    door_block_geometry = context.door_block_geometries.get(entity.id)
    if door_block_geometry is not None:
        normal = Point(x=-direction.y, y=direction.x)
        open_point = door_block_geometry.open_point
        axis_origin = host_wall.start if host_wall is not None else entity.start
        signed_offset = _project(open_point, axis_origin, normal)
        if abs(signed_offset) <= OPENING_DETECTOR_CONFIG.vector_epsilon:
            return None, "single_swing", door_block_geometry.hinge_side
        return (
            "positive_normal" if signed_offset > 0.0 else "negative_normal",
            "single_swing",
            door_block_geometry.hinge_side,
        )

    text_hint = next((text for text in context.texts if text.id == entity.id), None)
    if text_hint is not None:
        tokens = re.findall(r"[A-Z0-9]+", " ".join(text_hint.text.strip().upper().split()))
        if _is_sliding_door_label(tokens):
            return None, "sliding", None

    arc = _nearest_arc_to_segment(entity, context.arcs, context.thresholds)
    if arc is None:
        return None, None, None

    normal = Point(x=-direction.y, y=direction.x)
    arc_point = _arc_sample_point(arc)
    axis_origin = host_wall.start if host_wall is not None else entity.start
    signed_offset = _project(arc_point, axis_origin, normal)
    if abs(signed_offset) <= OPENING_DETECTOR_CONFIG.vector_epsilon:
        return None, "single_swing", None

    return ("positive_normal" if signed_offset > 0.0 else "negative_normal"), "single_swing", None


def _nearest_arc_to_segment(entity: LineEntity, arcs: list[ArcEntity], thresholds: OpeningThresholds) -> ArcEntity | None:
    direction = _unit_vector(entity.start, entity.end)
    if direction is None:
        return None

    normal = Point(x=-direction.y, y=direction.x)
    midpoint = _midpoint(entity.start, entity.end)
    start_t = _project(entity.start, entity.start, direction)
    end_t = _project(entity.end, entity.start, direction)
    lower_t = min(start_t, end_t)
    upper_t = max(start_t, end_t)

    nearby: list[tuple[float, ArcEntity]] = []
    for arc in arcs:
        axis_t = _project(arc.center, entity.start, direction)
        if axis_t < lower_t - thresholds.gap_axis_overhang or axis_t > upper_t + thresholds.gap_axis_overhang:
            continue

        normal_offset = abs(_project(arc.center, midpoint, normal))
        if normal_offset > max(thresholds.gap_arc_distance, arc.radius + thresholds.arc_search_radius):
            continue

        nearby.append((_dist(midpoint, arc.center), arc))

    if not nearby:
        return None

    nearby.sort(key=lambda item: item[0])
    return nearby[0][1]


def _match_segment(
    anchor: Point,
    segments: list[Segment],
    expected_length: float | None = None,
    preferred_orientation: Orientation | None = None,
    thresholds: OpeningThresholds | None = None,
    axis_overhang_limit: float | None = None,
    perp_dist_limit: float | None = None,
) -> Segment | None:
    thresholds = thresholds or build_opening_thresholds(None)
    search_radius = max(thresholds.segment_search_radius, (expected_length or 0) * 2.0)
    min_length = max(thresholds.min_opening_length, (expected_length or 0) * 0.5)
    axis_overhang_limit = thresholds.segment_overhang if axis_overhang_limit is None else axis_overhang_limit
    perp_dist_limit = thresholds.perp_dist_max if perp_dist_limit is None else perp_dist_limit

    candidates: list[tuple[float, float, float, Orientation, Segment]] = []

    for segment in segments:
        orientation = _orientation(segment.start, segment.end, thresholds)
        if orientation is None:
            continue
        if preferred_orientation is not None and orientation != preferred_orientation:
            continue

        length = _dist(segment.start, segment.end)
        if length < min_length:
            continue

        midpoint = _midpoint(segment.start, segment.end)
        if _dist(anchor, midpoint) > search_radius:
            continue

        if orientation == "horizontal":
            x0, x1 = sorted([segment.start.x, segment.end.x])
            if not (x0 - axis_overhang_limit <= anchor.x <= x1 + axis_overhang_limit):
                continue
            if abs(anchor.y - midpoint.y) > perp_dist_limit:
                continue
        else:
            y0, y1 = sorted([segment.start.y, segment.end.y])
            if not (y0 - axis_overhang_limit <= anchor.y <= y1 + axis_overhang_limit):
                continue
            if abs(anchor.x - midpoint.x) > perp_dist_limit:
                continue

        length_penalty = abs(length - expected_length) if expected_length else 0.0
        candidates.append((length_penalty, _dist(anchor, midpoint), -length, orientation, segment))

    if not candidates:
        return None

    _, _, _, orientation, primary = min(candidates, key=lambda item: (item[0], item[1], item[2]))

    group = [
        segment
        for _, _, _, candidate_orientation, segment in candidates
        if candidate_orientation == orientation and _are_parallel_twins(primary, segment, orientation, thresholds)
    ]

    return _merge_segments(group, orientation)


def _label_axis_overhang(expected_length: float | None, thresholds: OpeningThresholds) -> float:
    if expected_length is None:
        return thresholds.segment_overhang
    return max(thresholds.segment_overhang, expected_length * 0.25)


def _label_perpendicular_tolerance(expected_length: float | None, thresholds: OpeningThresholds) -> float:
    if expected_length is None:
        return thresholds.perp_dist_max
    return max(thresholds.perp_dist_max, expected_length * 0.75)


def _are_parallel_twins(a: Segment, b: Segment, orient: Orientation, thresholds: OpeningThresholds) -> bool:
    length_a = _dist(a.start, a.end)
    length_b = _dist(b.start, b.end)
    if min(length_a, length_b) / max(length_a, length_b) < OPENING_DETECTOR_CONFIG.parallel_length_min_ratio:
        return False

    midpoint_a = _midpoint(a.start, a.end)
    midpoint_b = _midpoint(b.start, b.end)

    if orient == "horizontal":
        if abs(midpoint_a.y - midpoint_b.y) > thresholds.parallel_midline_max_offset:
            return False
        a0, a1 = sorted([a.start.x, a.end.x])
        b0, b1 = sorted([b.start.x, b.end.x])
    else:
        if abs(midpoint_a.x - midpoint_b.x) > thresholds.parallel_midline_max_offset:
            return False
        a0, a1 = sorted([a.start.y, a.end.y])
        b0, b1 = sorted([b.start.y, b.end.y])

    if a0 < b1 and b0 < a1:
        return True

    gap = max(b0 - a1, a0 - b1)
    return gap <= thresholds.parallel_span_gap_max


def _merge_segments(group: list[Segment], orient: Orientation) -> Segment:
    if orient == "horizontal":
        x0 = min(min(segment.start.x, segment.end.x) for segment in group)
        x1 = max(max(segment.start.x, segment.end.x) for segment in group)
        y = sum(_midpoint(segment.start, segment.end).y for segment in group) / len(group)
        return Segment(
            Point(x0, round(y, 6)),
            Point(x1, round(y, 6)),
            group[0].source_entity_id,
            group[0].layer,
        )

    y0 = min(min(segment.start.y, segment.end.y) for segment in group)
    y1 = max(max(segment.start.y, segment.end.y) for segment in group)
    x = sum(_midpoint(segment.start, segment.end).x for segment in group) / len(group)
    return Segment(
        Point(round(x, 6), y0),
        Point(round(x, 6), y1),
        group[0].source_entity_id,
        group[0].layer,
    )


def _parse_label_width(text: str) -> float | None:
    digits = "".join(character for character in text if character.isdigit())
    if len(digits) < 2:
        return None
    return float(int(digits[0]) * 12 + int(digits[1]))


def _is_swing_door_label(text: str) -> bool:
    return len(text) == 4 and text.isdigit()


def _is_sliding_door_label(tokens: list[str]) -> bool:
    collapsed = "".join(tokens)
    return any(token in tokens or token in collapsed for token in OPENING_DETECTOR_CONFIG.sliding_door_tokens)


def _is_garage_door_label(tokens: list[str]) -> bool:
    token_set = set(tokens)
    return {"O", "H", "DOOR"} <= token_set or {"OH", "DOOR"} <= token_set


def _is_precise_match(seg: Segment, expected: float | None, thresholds: OpeningThresholds) -> bool:
    return expected is not None and abs(_dist(seg.start, seg.end) - expected) <= thresholds.precise_length_tolerance


def _infer_segment_orientation(anchor: Point, arc: ArcEntity | None, thresholds: OpeningThresholds) -> Orientation | None:
    if arc is None:
        return None
    return _orientation(anchor, arc.center, thresholds)


def _same_axis_direction(left: Point, right: Point) -> bool:
    return abs(left.x - right.x) <= 0.01 and abs(left.y - right.y) <= 0.01


def _arc_sample_point(arc: ArcEntity) -> Point:
    if arc.start_angle <= arc.end_angle:
        mid_angle = (arc.start_angle + arc.end_angle) / 2.0
    else:
        sweep = (arc.end_angle + 360.0) - arc.start_angle
        mid_angle = arc.start_angle + sweep / 2.0

    angle_radians = radians(mid_angle % 360.0)
    return Point(
        x=round(arc.center.x + cos(angle_radians) * arc.radius, 6),
        y=round(arc.center.y + sin(angle_radians) * arc.radius, 6),
    )


def _unit_vector(start: Point, end: Point) -> Point | None:
    dx = end.x - start.x
    dy = end.y - start.y
    length = _dist(start, end)
    if length <= OPENING_DETECTOR_CONFIG.vector_epsilon:
        return None

    unit = Point(x=dx / length, y=dy / length)
    if unit.x < 0.0 or (abs(unit.x) <= OPENING_DETECTOR_CONFIG.vector_epsilon and unit.y < 0.0):
        return Point(x=-unit.x, y=-unit.y)
    return unit


def _project(point: Point, origin: Point, direction: Point) -> float:
    return (point.x - origin.x) * direction.x + (point.y - origin.y) * direction.y


def _point_along(origin: Point, direction: Point, distance: float) -> Point:
    return Point(
        x=round(origin.x + direction.x * distance, 6),
        y=round(origin.y + direction.y * distance, 6),
    )


def _dedupe(entities: list[LineEntity]) -> list[LineEntity]:
    seen: set[tuple[float, float, float, float]] = set()
    output: list[LineEntity] = []

    for entity in entities:
        key = (
            round(entity.start.x, 6),
            round(entity.start.y, 6),
            round(entity.end.x, 6),
            round(entity.end.y, 6),
        )
        reverse = key[2:] + key[:2]
        if key in seen or reverse in seen:
            continue

        seen.add(key)
        output.append(entity)

    return output
