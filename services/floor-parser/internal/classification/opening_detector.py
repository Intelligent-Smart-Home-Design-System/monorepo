from __future__ import annotations

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
class OpeningThresholds:
    segment_search_radius: float
    arc_search_radius: float
    min_opening_length: float
    precise_length_tolerance: float
    segment_overhang: float
    perp_dist_max: float
    parallel_midline_max_offset: float
    gap_arc_distance: float
    gap_hint_distance: float
    gap_axis_overhang: float
    gap_axis_offset_tolerance: float
    parallel_span_gap_max: float
    orientation_axis_tolerance: float


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
        return (
            [_line_to_door(entity, resolved_context) for entity in detected_doors],
            [_line_to_window(entity, resolved_context) for entity in detected_windows],
        )

    def detect_doors(self, entities: list[NormalizedEntity], walls: list[Wall], units: str | None = None) -> list[Door]:
        doors, _ = self.detect(entities, walls, units=units)
        return doors

    def detect_windows(self, entities: list[NormalizedEntity], walls: list[Wall], units: str | None = None) -> list[Window]:
        _, windows = self.detect(entities, walls, units=units)
        return windows


def detect_by_layer(entities: list[NormalizedEntity], kind: Literal["door", "window"]) -> list[LineEntity]:
    pred = _is_door_layer if kind == "door" else _is_window_layer
    return [line for e in entities if pred(e.layer) for line in _entity_to_lines(e)]


def detect_by_block_name(
    entities: list[NormalizedEntity],
    kind: Literal["door", "window"],
    opening_segments: list[Segment],
    header_segments: list[Segment],
    thresholds: OpeningThresholds,
) -> list[LineEntity]:
    keywords = (
        OPENING_DETECTOR_CONFIG.door_block_keywords
        if kind == "door"
        else OPENING_DETECTOR_CONFIG.window_block_keywords
    )
    results: list[LineEntity] = []

    for entity in entities:
        if not isinstance(entity, InsertEntity):
            continue

        normalized_name = entity.block_name.strip().lower().replace("_", "-")
        if any(keyword in normalized_name for keyword in OPENING_DETECTOR_CONFIG.garage_layer_keywords):
            continue
        if not any(keyword in normalized_name for keyword in keywords):
            continue

        segment = (
            _match_segment(entity.insert, header_segments, thresholds=thresholds)
            or _match_segment(entity.insert, opening_segments, thresholds=thresholds)
        )
        if segment is not None:
            results.append(_segment_to_line(segment, entity.id))

    return results


def _detect_door_lines(entities: list[NormalizedEntity], context: OpeningContext) -> list[LineEntity]:
    gap_doors = detect_doors_from_gaps(context)
    return _dedupe([
        *detect_by_layer(entities, "door"),
        *detect_by_block_name(entities, "door", context.opening_segments, context.header_segments, context.thresholds),
        *gap_doors,
        *detect_doors_from_text_hints(context, {entity.id for entity in gap_doors}),
    ])


def _detect_window_lines(entities: list[NormalizedEntity], context: OpeningContext) -> list[LineEntity]:
    gap_windows = detect_windows_from_gaps(context)
    return _dedupe([
        *detect_by_layer(entities, "window"),
        *detect_by_block_name(entities, "window", context.opening_segments, context.header_segments, context.thresholds),
        *gap_windows,
        *detect_windows_from_text_hints(context, {entity.id for entity in gap_windows}),
    ])


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

        segment = (
            _match_segment(entity.insert, context.header_segments, expected_length, preferred_orientation, context.thresholds)
            or _match_segment(
                entity.insert,
                context.opening_segments + context.direct_door_segments,
                expected_length,
                preferred_orientation,
                context.thresholds,
            )
        )
        if segment is None:
            continue

        if nearest_arc is None and expected_length is not None and not _is_precise_match(segment, expected_length, context.thresholds):
            continue

        results.append(_segment_to_line(segment, entity.id))

    return results


def detect_windows_from_text_hints(context: OpeningContext, existing_ids: set[str]) -> list[LineEntity]:
    results: list[LineEntity] = []

    for entity in context.texts:
        normalized_text = " ".join(entity.text.strip().upper().split())
        tokens = re.findall(r"[A-Z0-9]+", normalized_text)
        if entity.id in existing_ids or not any(token in tokens for token in OPENING_DETECTOR_CONFIG.window_operation_tokens):
            continue

        expected_length = _parse_label_width(normalized_text)
        segment = (
            _match_segment(entity.insert, context.header_segments, expected_length, thresholds=context.thresholds)
            or _match_segment(
                entity.insert,
                context.opening_segments + context.direct_window_segments,
                expected_length,
                thresholds=context.thresholds,
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
    block_hint = _nearest_block_hint(gap, context.inserts, context.thresholds)
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


def _nearest_block_hint(gap: Gap, inserts: list[InsertEntity], thresholds: OpeningThresholds) -> GapHint | None:
    candidates: list[tuple[float, GapHint]] = []

    for entity in inserts:
        normalized_name = entity.block_name.strip().lower().replace("_", "-")
        if any(keyword in normalized_name for keyword in OPENING_DETECTOR_CONFIG.garage_layer_keywords):
            continue

        axis_distance = abs(_project(entity.insert, gap.center, gap.direction))
        normal_distance = abs(_project(entity.insert, gap.center, gap.normal))
        if axis_distance > gap.length / 2.0 + thresholds.gap_axis_overhang:
            continue
        if normal_distance > thresholds.gap_hint_distance:
            continue

        if any(keyword in normalized_name for keyword in OPENING_DETECTOR_CONFIG.door_block_keywords):
            candidates.append((normal_distance + axis_distance, GapHint(kind="door", source_id=entity.id, anchor=entity.insert)))
        if any(keyword in normalized_name for keyword in OPENING_DETECTOR_CONFIG.window_block_keywords):
            candidates.append((normal_distance + axis_distance, GapHint(kind="window", source_id=entity.id, anchor=entity.insert)))

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

        axis_distance = abs(_project(entity.insert, gap.center, gap.direction))
        normal_distance = abs(_project(entity.insert, gap.center, gap.normal))
        if axis_distance > gap.length / 2.0 + thresholds.gap_axis_overhang:
            continue
        if normal_distance > thresholds.gap_hint_distance:
            continue

        if _is_sliding_door_label(tokens) or _is_swing_door_label(normalized_text):
            candidates.append((
                normal_distance + axis_distance,
                GapHint(
                    kind="sliding_door" if _is_sliding_door_label(tokens) else "door",
                    source_id=entity.id,
                    anchor=entity.insert,
                    expected_length=_parse_label_width(normalized_text),
                ),
            ))
        elif any(token in tokens for token in OPENING_DETECTOR_CONFIG.window_operation_tokens):
            candidates.append((
                normal_distance + axis_distance,
                GapHint(
                    kind="window",
                    source_id=entity.id,
                    anchor=entity.insert,
                    expected_length=_parse_label_width(normalized_text),
                ),
            ))

    if not candidates:
        return None

    _, hint = min(candidates, key=lambda item: item[0])
    return hint


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
    opens_towards_wall_side, swing = _resolve_door_opening(entity, context, host_wall)
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


def _resolve_wall_binding(entity: LineEntity, context: OpeningContext) -> tuple[Wall | None, tuple[str, ...]]:
    gap = _nearest_gap_for_segment(entity, context.gaps, context.thresholds)
    if gap is not None:
        support_walls = [
            wall
            for wall_id in gap.support_wall_ids
            if (wall := _find_wall_by_id(context.walls, wall_id)) is not None
        ]
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
        if normal_offset > max(wall.width, context.thresholds.perp_dist_max):
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

        gap_mid_distance = _dist(midpoint, gap.center)
        if gap_mid_distance > thresholds.segment_search_radius:
            continue

        gap_start_t = _project(gap.start, gap.start, gap.direction)
        gap_end_t = _project(gap.end, gap.start, gap.direction)
        seg_start_t = _project(entity.start, gap.start, gap.direction)
        seg_end_t = _project(entity.end, gap.start, gap.direction)
        overlap = min(max(seg_start_t, seg_end_t), max(gap_start_t, gap_end_t)) - max(min(seg_start_t, seg_end_t), min(gap_start_t, gap_end_t))
        if overlap <= 0.0:
            continue

        score = (gap_mid_distance, abs(gap.length - _dist(entity.start, entity.end)))
        if best_score is None or score < best_score:
            best_gap = gap
            best_score = score

    return best_gap


def _find_wall_by_id(walls: list[Wall], wall_id: str) -> Wall | None:
    for wall in walls:
        if wall.id == wall_id:
            return wall
    return None


def _build_opening_context(
    entities: list[NormalizedEntity],
    walls: list[Wall],
    thresholds: OpeningThresholds,
) -> OpeningContext:
    return OpeningContext(
        walls=walls,
        opening_segments=_collect_segments(entities, _is_opening_layer),
        header_segments=_collect_segments(entities, _is_header_layer),
        direct_door_segments=_collect_segments(entities, _is_door_layer),
        direct_window_segments=_collect_segments(entities, _is_window_layer),
        arcs=[entity for entity in entities if isinstance(entity, ArcEntity) and _is_opening_layer(entity.layer)],
        texts=[entity for entity in entities if isinstance(entity, TextEntity)],
        inserts=[entity for entity in entities if isinstance(entity, InsertEntity)],
        gaps=_detect_gaps(walls, thresholds),
        wall_host_ids={},
        thresholds=thresholds,
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
        gaps=context.gaps,
        wall_host_ids=_build_wall_host_ids(context, openings),
        thresholds=context.thresholds,
    )


def _build_wall_host_ids(context: OpeningContext, openings: list[LineEntity]) -> dict[str, str]:
    parents = {wall.id: wall.id for wall in context.walls}

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
        gap = _nearest_gap_for_segment(opening, context.gaps, context.thresholds)
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


def _select_host_wall(walls: list[Wall]) -> Wall:
    structural_walls = [wall for wall in walls if wall.width > 0.0]
    candidates = structural_walls if structural_walls else walls
    return max(candidates, key=lambda wall: (wall.length, wall.id))


def _resolve_host_wall_id(host_wall: Wall | None, support_wall_ids: tuple[str, ...], context: OpeningContext) -> str | None:
    for wall_id in support_wall_ids:
        host_id = context.wall_host_ids.get(wall_id)
        if host_id is not None:
            return host_id

    if host_wall is None:
        return None

    return context.wall_host_ids.get(host_wall.id, host_wall.run_id or host_wall.id)


def _resolve_door_opening(entity: LineEntity, context: OpeningContext, host_wall: Wall | None) -> tuple[str | None, str | None]:
    direction = _unit_vector(
        host_wall.start if host_wall is not None else entity.start,
        host_wall.end if host_wall is not None else entity.end,
    )
    if direction is None:
        return None, None

    text_hint = next((text for text in context.texts if text.id == entity.id), None)
    if text_hint is not None:
        tokens = re.findall(r"[A-Z0-9]+", " ".join(text_hint.text.strip().upper().split()))
        if _is_sliding_door_label(tokens):
            return None, "sliding"

    arc = _nearest_arc_to_segment(entity, context.arcs, context.thresholds)
    if arc is None:
        return None, None

    normal = Point(x=-direction.y, y=direction.x)
    arc_point = _arc_sample_point(arc)
    axis_origin = host_wall.start if host_wall is not None else entity.start
    signed_offset = _project(arc_point, axis_origin, normal)
    if abs(signed_offset) <= OPENING_DETECTOR_CONFIG.vector_epsilon:
        return None, "single_swing"

    return ("positive_normal" if signed_offset > 0.0 else "negative_normal"), "single_swing"


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
) -> Segment | None:
    thresholds = thresholds or build_opening_thresholds(None)
    search_radius = max(thresholds.segment_search_radius, (expected_length or 0) * 2.0)
    min_length = max(thresholds.min_opening_length, (expected_length or 0) * 0.5)

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
            if not (x0 - thresholds.segment_overhang <= anchor.x <= x1 + thresholds.segment_overhang):
                continue
            if abs(anchor.y - midpoint.y) > thresholds.perp_dist_max:
                continue
        else:
            y0, y1 = sorted([segment.start.y, segment.end.y])
            if not (y0 - thresholds.segment_overhang <= anchor.y <= y1 + thresholds.segment_overhang):
                continue
            if abs(anchor.x - midpoint.x) > thresholds.perp_dist_max:
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
