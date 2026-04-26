from __future__ import annotations

from dataclasses import dataclass
from math import sqrt
from typing import NamedTuple

from internal.entities.floor import Wall
from internal.entities.geometry import LineEntity, NormalizedEntity, Point, PolylineEntity


MIN_WALL_WIDTH_MM = 2.0
PARALLEL_CROSS_TOLERANCE = 0.01
MIN_OVERLAP_RATIO = 0.6
MAX_WIDTH_TO_LENGTH = 0.35
LINE_OFFSET_TOLERANCE_MM = 2.0
MAX_RUN_GAP_MM = 250.0
MAX_FALLBACK_RUN_GAP_MM = 25.0
RUN_WIDTH_TOLERANCE_MM = 50.0

WALL_LAYER_MARKERS = (
    "wall",
    "walls",
    "стена",
    "стены",
)

UNITS_TO_MILLIMETERS: dict[str, float] = {
    "mm": 1.0,
    "millimeter": 1.0,
    "millimeters": 1.0,
    "cm": 10.0,
    "centimeter": 10.0,
    "centimeters": 10.0,
    "m": 1000.0,
    "meter": 1000.0,
    "meters": 1000.0,
    "in": 25.4,
    "inch": 25.4,
    "inches": 25.4,
    "ft": 304.8,
    "foot": 304.8,
    "feet": 304.8,
}


@dataclass(frozen=True)
class WallBoundary:
    id: str
    layer: str
    start: Point
    end: Point
    source_entity_ids: list[str]


class PartnerScore(NamedTuple):
    width: float
    neg_overlap: float


class WallDetector:
    def detect(self, entities: list[NormalizedEntity], units: str | None = None) -> list[Wall]:
        boundaries = self._collect_boundaries(entities)
        min_width = self._mm_to_units(MIN_WALL_WIDTH_MM, units)
        offset_tolerance = self._mm_to_units(LINE_OFFSET_TOLERANCE_MM, units)

        walls: list[Wall] = []
        used: set[str] = set()

        for boundary in boundaries:
            if boundary.id in used:
                continue

            partner = self._find_partner(boundary, boundaries, used, min_width, offset_tolerance)
            if partner is not None:
                walls.append(self._build_wall(boundary, partner))
                used.add(boundary.id)
                used.add(partner.id)
            else:
                walls.append(self._build_fallback_wall(boundary))
                used.add(boundary.id)

        max_run_gap = self._mm_to_units(MAX_RUN_GAP_MM, units)
        max_fallback_run_gap = self._mm_to_units(MAX_FALLBACK_RUN_GAP_MM, units)
        run_width_tolerance = self._mm_to_units(RUN_WIDTH_TOLERANCE_MM, units)
        return self._assign_run_ids(
            walls,
            offset_tolerance,
            max_run_gap,
            max_fallback_run_gap,
            run_width_tolerance,
        )

    def _collect_boundaries(self, entities: list[NormalizedEntity]) -> list[WallBoundary]:
        boundaries: list[WallBoundary] = []

        for entity in entities:
            if not self._is_wall_layer(entity.layer):
                continue

            if isinstance(entity, LineEntity):
                boundaries.extend(self._boundaries_from_segment(entity.id, entity.layer, entity.start, entity.end, [entity.id]))
            elif isinstance(entity, PolylineEntity) and len(entity.points) >= 2:
                pts = entity.points
                segments = list(zip(pts, pts[1:]))
                if entity.closed:
                    segments.append((pts[-1], pts[0]))

                for i, (start, end) in enumerate(segments):
                    seg_id = f"{entity.id}:{i + 1}" if i < len(pts) - 1 else f"{entity.id}:closing"
                    boundaries.extend(self._boundaries_from_segment(seg_id, entity.layer, start, end, [entity.id]))

        return boundaries

    def _is_wall_layer(self, layer: str) -> bool:
        normalized_layer = layer.strip().lower()
        return any(marker in normalized_layer for marker in WALL_LAYER_MARKERS)

    def _boundaries_from_segment(
        self,
        seg_id: str,
        layer: str,
        start: Point,
        end: Point,
        source_ids: list[str],
    ) -> list[WallBoundary]:
        if self._length(start, end) == 0.0:
            return []
        return [WallBoundary(id=seg_id, layer=layer, start=start, end=end, source_entity_ids=source_ids)]

    def _find_partner(
        self,
        boundary: WallBoundary,
        boundaries: list[WallBoundary],
        used: set[str],
        min_width: float,
        offset_tolerance: float,
    ) -> WallBoundary | None:
        direction = self._unit_vector(boundary.start, boundary.end)
        if direction is None:
            return None

        best: WallBoundary | None = None
        best_score: PartnerScore | None = None

        for candidate in boundaries:
            if candidate.id == boundary.id or candidate.id in used:
                continue

            score = self._score_candidate(boundary, candidate, direction, min_width, offset_tolerance)
            if score is not None and (best_score is None or score < best_score):
                best = candidate
                best_score = score

        return best

    def _score_candidate(
        self,
        boundary: WallBoundary,
        candidate: WallBoundary,
        direction: Point,
        min_width: float,
        offset_tolerance: float,
    ) -> PartnerScore | None:
        candidate_direction = self._unit_vector(candidate.start, candidate.end)
        if candidate_direction is None or not self._are_parallel(direction, candidate_direction):
            return None

        overlap = self._axis_overlap(boundary, candidate, direction)
        if overlap <= 0.0:
            return None

        shorter = min(self._length(boundary.start, boundary.end), self._length(candidate.start, candidate.end))
        if shorter == 0.0 or overlap / shorter < MIN_OVERLAP_RATIO:
            return None

        offsets = self._signed_offsets(boundary, candidate, direction)
        if offsets is None:
            return None

        offset_start, offset_end = offsets
        if abs(offset_start - offset_end) > offset_tolerance:
            return None

        width = (abs(offset_start) + abs(offset_end)) / 2.0
        if width < min_width or width / overlap > MAX_WIDTH_TO_LENGTH:
            return None

        return PartnerScore(width=width, neg_overlap=-overlap)

    def _build_wall(self, left: WallBoundary, right: WallBoundary) -> Wall:
        direction = self._unit_vector(left.start, left.end)
        normal = Point(x=-direction.y, y=direction.x)

        offsets = self._signed_offsets(left, right, direction)
        offset_start, offset_end = offsets
        width = (abs(offset_start) + abs(offset_end)) / 2.0

        mean_normal_offset = (offset_start + offset_end) / 2.0
        axis_origin = Point(
            x=left.start.x + normal.x * (mean_normal_offset / 2.0),
            y=left.start.y + normal.y * (mean_normal_offset / 2.0),
        )

        proj_start, proj_end = self._overlap_interval(left, right, left.start, direction)
        start = self._point_along(axis_origin, direction, proj_start)
        end = self._point_along(axis_origin, direction, proj_end)

        return Wall(
            id=left.id,
            layer=left.layer,
            start=start,
            end=end,
            length=self._length(start, end),
            width=round(width, 6),
            source_entity_ids=sorted(set(left.source_entity_ids + right.source_entity_ids)),
        )

    def _build_fallback_wall(self, boundary: WallBoundary) -> Wall:
        return Wall(
            id=boundary.id,
            layer=boundary.layer,
            start=boundary.start,
            end=boundary.end,
            length=self._length(boundary.start, boundary.end),
            width=0.0,
            source_entity_ids=boundary.source_entity_ids,
        )

    def _assign_run_ids(
        self,
        walls: list[Wall],
        offset_tolerance: float,
        max_run_gap: float,
        max_fallback_run_gap: float,
        run_width_tolerance: float,
    ) -> list[Wall]:
        direction_groups: dict[tuple[float, float], list[tuple[float, Wall]]] = {}

        for wall in walls:
            direction = self._unit_vector(wall.start, wall.end)
            if direction is None:
                continue

            normal = Point(x=-direction.y, y=direction.x)
            line_offset = self._project(wall.start, Point(0.0, 0.0), normal)
            direction_key = (
                round(direction.x, 3),
                round(direction.y, 3),
            )
            direction_groups.setdefault(direction_key, []).append((line_offset, wall))

        run_ids: dict[str, str] = {}

        for grouped in direction_groups.values():
            grouped.sort(key=lambda item: item[0])
            offset_clusters: list[list[tuple[float, Wall]]] = [[grouped[0]]]

            for item in grouped[1:]:
                if abs(item[0] - offset_clusters[-1][-1][0]) <= offset_tolerance:
                    offset_clusters[-1].append(item)
                else:
                    offset_clusters.append([item])

            for cluster in offset_clusters:
                axis_origin = cluster[0][1].start
                direction = self._unit_vector(cluster[0][1].start, cluster[0][1].end)
                intervals: list[tuple[Wall, float, float]] = []

                for _, wall in cluster:
                    start_t = self._project(wall.start, axis_origin, direction)
                    end_t = self._project(wall.end, axis_origin, direction)
                    intervals.append((wall, min(start_t, end_t), max(start_t, end_t)))

                intervals.sort(key=lambda item: (item[1], item[2]))
                run_clusters: list[list[tuple[Wall, float, float]]] = [[intervals[0]]]
                current_end = intervals[0][2]

                for wall, start_t, end_t in intervals[1:]:
                    cluster_width = self._cluster_reference_width(run_clusters[-1])
                    gap = start_t - current_end
                    gap_tolerance = self._run_gap_tolerance(cluster_width, wall.width, max_run_gap, max_fallback_run_gap)
                    if gap <= gap_tolerance and self._widths_are_compatible(cluster_width, wall.width, run_width_tolerance):
                        run_clusters[-1].append((wall, start_t, end_t))
                        current_end = max(current_end, end_t)
                        continue

                    run_clusters.append([(wall, start_t, end_t)])
                    current_end = end_t

                for run_cluster in run_clusters:
                    representative = self._representative_run_wall(run_cluster)
                    for wall, _, _ in run_cluster:
                        run_ids[wall.id] = representative.id

        return [
            Wall(
                id=wall.id,
                layer=wall.layer,
                start=wall.start,
                end=wall.end,
                length=wall.length,
                width=wall.width,
                run_id=run_ids.get(wall.id, wall.id),
                source_entity_ids=wall.source_entity_ids,
            )
            for wall in walls
        ]

    def _widths_are_compatible(self, left_width: float, right_width: float, tolerance: float) -> bool:
        if left_width == 0.0 or right_width == 0.0:
            return True
        return abs(left_width - right_width) <= tolerance

    def _cluster_reference_width(self, run_cluster: list[tuple[Wall, float, float]]) -> float:
        nonzero_widths = [wall.width for wall, _, _ in run_cluster if wall.width > 0.0]
        if not nonzero_widths:
            return 0.0
        return sum(nonzero_widths) / len(nonzero_widths)

    def _run_gap_tolerance(
        self,
        cluster_width: float,
        wall_width: float,
        max_run_gap: float,
        max_fallback_run_gap: float,
    ) -> float:
        if cluster_width == 0.0 or wall_width == 0.0:
            return max_fallback_run_gap
        return max_run_gap

    def _representative_run_wall(self, run_cluster: list[tuple[Wall, float, float]]) -> Wall:
        structural_walls = [wall for wall, _, _ in run_cluster if wall.width > 0.0]
        candidates = structural_walls if structural_walls else [wall for wall, _, _ in run_cluster]
        return max(candidates, key=lambda wall: (wall.length, wall.id))

    def _unit_vector(self, start: Point, end: Point) -> Point | None:
        length = self._length(start, end)
        if length == 0.0:
            return None

        ux = (end.x - start.x) / length
        uy = (end.y - start.y) / length
        if ux < 0.0 or (abs(ux) <= PARALLEL_CROSS_TOLERANCE and uy < 0.0):
            ux, uy = -ux, -uy

        return Point(x=ux, y=uy)

    def _are_parallel(self, a: Point, b: Point) -> bool:
        return abs(a.x * b.y - a.y * b.x) <= PARALLEL_CROSS_TOLERANCE

    def _project(self, point: Point, origin: Point, direction: Point) -> float:
        return (point.x - origin.x) * direction.x + (point.y - origin.y) * direction.y

    def _axis_overlap(self, left: WallBoundary, right: WallBoundary, direction: Point) -> float:
        origin = left.start
        l0, l1 = sorted([self._project(left.start, origin, direction), self._project(left.end, origin, direction)])
        r0, r1 = sorted([self._project(right.start, origin, direction), self._project(right.end, origin, direction)])
        return round(min(l1, r1) - max(l0, r0), 6)

    def _overlap_interval(
        self,
        left: WallBoundary,
        right: WallBoundary,
        origin: Point,
        direction: Point,
    ) -> tuple[float, float]:
        l0, l1 = sorted([self._project(left.start, origin, direction), self._project(left.end, origin, direction)])
        r0, r1 = sorted([self._project(right.start, origin, direction), self._project(right.end, origin, direction)])
        return max(l0, r0), min(l1, r1)

    def _signed_offsets(self, reference: WallBoundary, candidate: WallBoundary, direction: Point):
        normal = Point(x=-direction.y, y=direction.x)
        start_offset = self._project(candidate.start, reference.start, normal)
        end_offset = self._project(candidate.end, reference.start, normal)
        return start_offset, end_offset

    def _point_along(self, origin: Point, direction: Point, distance: float) -> Point:
        return Point(x=round(origin.x + direction.x * distance, 6), y=round(origin.y + direction.y * distance, 6))

    def _length(self, start: Point, end: Point) -> float:
        return round(sqrt((end.x - start.x) ** 2 + (end.y - start.y) ** 2), 6)

    def _mm_to_units(self, value_mm: float, units: str | None) -> float:
        if units is None:
            return value_mm
        mm_per_unit = UNITS_TO_MILLIMETERS.get(units.strip().lower())
        return value_mm / mm_per_unit if mm_per_unit is not None else value_mm
