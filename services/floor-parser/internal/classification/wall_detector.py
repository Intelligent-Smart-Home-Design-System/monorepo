<<<<<<< HEAD
from __future__ import annotations

from collections import defaultdict
from dataclasses import dataclass
from statistics import median
from math import sqrt
from typing import NamedTuple

from internal.classification.config import WALL_DETECTOR_CONFIG, build_wall_thresholds
from internal.entities.floor import Wall
from internal.entities.geometry import LineEntity, NormalizedEntity, Point, PolylineEntity


@dataclass(frozen=True)
class WallBoundary:
    id: str
    layer: str
    start: Point
    end: Point
    source_entity_id: str | None
    source_is_closed_polyline: bool
    source_entity_ids: list[str]


class PartnerScore(NamedTuple):
    width: float
    neg_overlap: float


class WallDetector:
    def detect(self, entities: list[NormalizedEntity], units: str | None = None) -> list[Wall]:
        boundaries = self._collect_boundaries(entities)
        thresholds = build_wall_thresholds(units)
        min_width = thresholds.min_wall_width
        offset_tolerance = thresholds.line_offset_tolerance

        walls: list[Wall] = []
        used: set[str] = set()
        width_hints: dict[str, float] = {}

        for boundary in boundaries:
            if boundary.id in used:
                continue

            partners = self._find_all_partners(boundary, boundaries, used, min_width, offset_tolerance)
            if partners:
                for i, partner in enumerate(partners):
                    wall_id = boundary.id if i == 0 else f"{boundary.id}:m{i + 1}"
                    wall = self._build_wall(boundary, partner, wall_id=wall_id)
                    walls.append(wall)
                    if not self._can_reuse_partner(boundary, partner):
                        used.add(partner.id)
                    width_hints[boundary.id] = wall.width
                    width_hints[partner.id] = wall.width
                used.add(boundary.id)
            else:
                walls.append(self._build_fallback_wall(boundary))
                used.add(boundary.id)

        assigned_walls = self._assign_run_ids(
            walls,
            offset_tolerance,
            thresholds.max_run_gap,
            thresholds.max_fallback_run_gap,
            thresholds.run_width_tolerance,
        )
        repaired_walls = self._repair_boundary_face_walls(
            assigned_walls,
            boundaries,
            width_hints,
            min_width,
            thresholds.max_closed_polyline_width,
            offset_tolerance,
        )
        width_completed_walls = self._propagate_touching_boundary_widths(repaired_walls)
        return self._snap_collinear_walls(
            width_completed_walls,
            thresholds.collinear_snap_tolerance,
            thresholds.snap_drift_ratio_max,
        )

    def _collect_boundaries(self, entities: list[NormalizedEntity]) -> list[WallBoundary]:
        boundaries: list[WallBoundary] = []

        for entity in entities:
            if not self._is_wall_layer(entity.layer):
                continue

            if isinstance(entity, LineEntity):
                boundaries.extend(
                    self._boundaries_from_segment(
                        entity.id,
                        entity.layer,
                        entity.start,
                        entity.end,
                        entity.id,
                        False,
                        [entity.id],
                    )
                )
            elif isinstance(entity, PolylineEntity) and len(entity.points) >= 2:
                pts = entity.points
                segments = list(zip(pts, pts[1:]))
                if entity.closed:
                    segments.append((pts[-1], pts[0]))

                for i, (start, end) in enumerate(segments):
                    seg_id = f"{entity.id}:{i + 1}" if i < len(pts) - 1 else f"{entity.id}:closing"
                    boundaries.extend(
                        self._boundaries_from_segment(
                            seg_id,
                            entity.layer,
                            start,
                            end,
                            entity.id,
                            entity.closed,
                            [entity.id],
                        )
                    )

        return boundaries

    def _is_wall_layer(self, layer: str) -> bool:
        normalized_layer = layer.strip().lower()
        return any(marker in normalized_layer for marker in WALL_DETECTOR_CONFIG.wall_layer_markers)

    def _boundaries_from_segment(
        self,
        seg_id: str,
        layer: str,
        start: Point,
        end: Point,
        source_entity_id: str | None,
        source_is_closed_polyline: bool,
        source_ids: list[str],
    ) -> list[WallBoundary]:
        if self._length(start, end) == 0.0:
            return []
        return [
            WallBoundary(
                id=seg_id,
                layer=layer,
                start=start,
                end=end,
                source_entity_id=source_entity_id,
                source_is_closed_polyline=source_is_closed_polyline,
                source_entity_ids=source_ids,
            )
        ]

    def _can_reuse_partner(self, boundary: WallBoundary, partner: WallBoundary) -> bool:
        if (
            not boundary.source_is_closed_polyline
            or not partner.source_is_closed_polyline
            or boundary.source_entity_id is None
            or boundary.source_entity_id != partner.source_entity_id
        ):
            return False
        return self._length(partner.start, partner.end) > self._length(boundary.start, boundary.end) + 1e-6

    def _find_all_partners(
        self,
        boundary: WallBoundary,
        boundaries: list[WallBoundary],
        used: set[str],
        min_width: float,
        offset_tolerance: float,
    ) -> list[WallBoundary]:
        direction = self._unit_vector(boundary.start, boundary.end)
        if direction is None:
            return []

        scored: list[tuple[PartnerScore, WallBoundary]] = []
        for candidate in boundaries:
            if candidate.id == boundary.id or candidate.id in used:
                continue
            score = self._score_candidate(
                boundary,
                candidate,
                direction,
                min_width,
                offset_tolerance,
                max_width_to_length=self._pair_width_ratio_limit(boundary, candidate),
            )
            if score is not None:
                scored.append((score, candidate))

        if not scored:
            return []

        scored.sort(key=lambda item: (item[0].width, item[0].neg_overlap, item[1].id))

        selected: list[WallBoundary] = []
        claimed: list[tuple[float, float]] = []

        for _, candidate in scored:
            c0, c1 = sorted([
                self._project(candidate.start, boundary.start, direction),
                self._project(candidate.end, boundary.start, direction),
            ])
            if not any(c0 < ce and c1 > cs for cs, ce in claimed):
                selected.append(candidate)
                claimed.append((c0, c1))

        return selected

    def _score_candidate(
        self,
        boundary: WallBoundary,
        candidate: WallBoundary,
        direction: Point,
        min_width: float,
        offset_tolerance: float,
        max_width_to_length: float | None = None,
    ) -> PartnerScore | None:
        candidate_direction = self._unit_vector(candidate.start, candidate.end)
        if candidate_direction is None or not self._are_parallel(direction, candidate_direction):
            return None

        overlap = self._axis_overlap(boundary, candidate, direction)
        if overlap <= 0.0:
            return None

        shorter = min(self._length(boundary.start, boundary.end), self._length(candidate.start, candidate.end))
        if shorter == 0.0 or overlap / shorter < WALL_DETECTOR_CONFIG.min_overlap_ratio:
            return None

        offsets = self._signed_offsets(boundary, candidate, direction)
        if offsets is None:
            return None

        offset_start, offset_end = offsets
        width = (abs(offset_start) + abs(offset_end)) / 2.0
        if abs(offset_start - offset_end) > self._pair_offset_tolerance(boundary, candidate, offset_tolerance, width):
            return None
        width_to_length_limit = max_width_to_length or WALL_DETECTOR_CONFIG.max_width_to_length
        if width < min_width or width / overlap > width_to_length_limit:
            return None

        return PartnerScore(width=width, neg_overlap=-overlap)

    def _build_wall(self, left: WallBoundary, right: WallBoundary, wall_id: str | None = None) -> Wall:
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
            id=wall_id or left.id,
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
            geometry_role="boundary_face",
            source_entity_ids=boundary.source_entity_ids,
        )

    def _repair_boundary_face_walls(
        self,
        walls: list[Wall],
        boundaries: list[WallBoundary],
        width_hints: dict[str, float],
        min_width: float,
        max_closed_polyline_width: float,
        offset_tolerance: float,
    ) -> list[Wall]:
        boundary_by_id = {boundary.id: boundary for boundary in boundaries}
        repaired_width_hints = dict(width_hints)
        neighbors = self._build_boundary_neighbors(boundaries)

        for wall in walls:
            if wall.width > 0.0:
                continue

            boundary = boundary_by_id.get(wall.id)
            if boundary is None or not boundary.source_is_closed_polyline:
                continue

            repaired_width = self._infer_boundary_width(
                boundary,
                boundaries,
                min_width,
                max_closed_polyline_width,
                offset_tolerance,
            )
            if repaired_width is not None:
                repaired_width_hints[boundary.id] = repaired_width

        repaired_width_hints = self._propagate_width_hints(boundaries, neighbors, repaired_width_hints)
        return [self._apply_width_hint(wall, repaired_width_hints.get(wall.id)) for wall in walls]

    def _infer_boundary_width(
        self,
        boundary: WallBoundary,
        boundaries: list[WallBoundary],
        min_width: float,
        max_closed_polyline_width: float,
        offset_tolerance: float,
    ) -> float | None:
        direction = self._unit_vector(boundary.start, boundary.end)
        if direction is None:
            return None

        best_width: float | None = None
        best_score: PartnerScore | None = None

        for candidate in boundaries:
            if candidate.id == boundary.id:
                continue
            if (
                not candidate.source_is_closed_polyline
                or candidate.source_entity_id is None
                or candidate.source_entity_id != boundary.source_entity_id
            ):
                continue

            score = self._score_candidate(
                boundary,
                candidate,
                direction,
                min_width,
                offset_tolerance,
                self._pair_width_ratio_limit(boundary, candidate),
            )
            if score is not None and (best_score is None or score < best_score):
                best_score = score
                best_width = round(score.width, 6)

        if best_width is not None and best_width > max_closed_polyline_width:
            return None
        return best_width

    def _pair_width_ratio_limit(self, boundary: WallBoundary, candidate: WallBoundary) -> float:
        if (
            boundary.source_is_closed_polyline
            and candidate.source_is_closed_polyline
            and boundary.source_entity_id is not None
            and boundary.source_entity_id == candidate.source_entity_id
        ):
            return WALL_DETECTOR_CONFIG.closed_polyline_width_to_length
        return WALL_DETECTOR_CONFIG.max_width_to_length

    def _pair_offset_tolerance(
        self,
        boundary: WallBoundary,
        candidate: WallBoundary,
        base_tolerance: float,
        width: float,
    ) -> float:
        if (
            boundary.source_is_closed_polyline
            and candidate.source_is_closed_polyline
            and boundary.source_entity_id is not None
            and boundary.source_entity_id == candidate.source_entity_id
        ):
            return max(base_tolerance, width * WALL_DETECTOR_CONFIG.closed_polyline_offset_tolerance_ratio)
        return base_tolerance

    def _build_boundary_neighbors(self, boundaries: list[WallBoundary]) -> dict[str, set[str]]:
        neighbors: dict[str, set[str]] = defaultdict(set)
        for index, left in enumerate(boundaries):
            for right in boundaries[index + 1 :]:
                if self._boundaries_touch(left, right):
                    neighbors[left.id].add(right.id)
                    neighbors[right.id].add(left.id)

        return neighbors

    def _boundaries_touch(self, left: WallBoundary, right: WallBoundary) -> bool:
        return any(
            self._same_point(a, b)
            for a in (left.start, left.end)
            for b in (right.start, right.end)
        )

    def _same_point(self, left: Point, right: Point) -> bool:
        return abs(left.x - right.x) <= 1e-6 and abs(left.y - right.y) <= 1e-6

    def _propagate_width_hints(
        self,
        boundaries: list[WallBoundary],
        neighbors: dict[str, set[str]],
        width_hints: dict[str, float],
    ) -> dict[str, float]:
        propagated = dict(width_hints)
        boundary_by_id = {boundary.id: boundary for boundary in boundaries}

        changed = True
        while changed:
            changed = False
            for boundary in boundaries:
                if boundary.id in propagated:
                    continue

                weighted_neighbor_widths = [
                    (
                        propagated[neighbor_id],
                        max(self._length(boundary_by_id[neighbor_id].start, boundary_by_id[neighbor_id].end), 1e-6),
                    )
                    for neighbor_id in neighbors.get(boundary.id, set())
                    if propagated.get(neighbor_id, 0.0) > 0.0
                ]
                if not weighted_neighbor_widths:
                    continue

                total_weight = sum(weight for _, weight in weighted_neighbor_widths)
                weighted_width = sum(width * weight for width, weight in weighted_neighbor_widths) / total_weight
                propagated[boundary.id] = round(weighted_width, 6)
                changed = True

        return propagated

    def _apply_width_hint(self, wall: Wall, width_hint: float | None) -> Wall:
        if wall.width > 0.0:
            return wall

        return Wall(
            id=wall.id,
            layer=wall.layer,
            start=wall.start,
            end=wall.end,
            length=wall.length,
            width=round(width_hint, 6) if width_hint is not None else 0.0,
            geometry_role="boundary_face",
            run_id=wall.run_id,
            source_entity_ids=wall.source_entity_ids,
        )

    def _propagate_touching_boundary_widths(self, walls: list[Wall]) -> list[Wall]:
        neighbors: dict[str, set[str]] = defaultdict(set)
        wall_by_id = {wall.id: wall for wall in walls}

        for index, left in enumerate(walls):
            for right in walls[index + 1 :]:
                if self._walls_touch(left, right):
                    neighbors[left.id].add(right.id)
                    neighbors[right.id].add(left.id)

        inferred_widths = {
            wall.id: wall.width
            for wall in walls
            if wall.width > 0.0
        }

        changed = True
        while changed:
            changed = False
            for wall in walls:
                if wall.width > 0.0 or wall.id in inferred_widths:
                    continue
                if wall.geometry_role != "boundary_face":
                    continue

                neighbor_widths = [
                    inferred_widths[neighbor_id]
                    for neighbor_id in neighbors.get(wall.id, set())
                    if inferred_widths.get(neighbor_id, 0.0) > 0.0
                ]
                if not neighbor_widths:
                    continue

                inferred_widths[wall.id] = round(median(neighbor_widths), 6)
                changed = True

        return [self._apply_width_hint(wall, inferred_widths.get(wall.id)) for wall in walls]

    def _walls_touch(self, left: Wall, right: Wall) -> bool:
        return any(
            self._same_point(a, b)
            for a in (left.start, left.end)
            for b in (right.start, right.end)
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
                geometry_role=wall.geometry_role,
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

    def _snap_collinear_walls(
        self,
        walls: list[Wall],
        collinear_snap_tolerance: float,
        snap_drift_ratio_max: float,
    ) -> list[Wall]:
        if not walls:
            return walls

        origin = Point(0.0, 0.0)

        by_direction: dict[tuple[float, float], list[tuple[float, Wall]]] = defaultdict(list)

        for wall in walls:
            if wall.geometry_role == "boundary_face" or wall.width == 0.0:
                continue

            direction = self._unit_vector(wall.start, wall.end)
            if direction is None:
                continue

            dir_key = (round(direction.x, 3), round(direction.y, 3))
            normal = Point(x=-direction.y, y=direction.x)
            offset = self._project(wall.start, origin, normal)
            by_direction[dir_key].append((offset, wall))

        corrections: dict[str, tuple[Point, Point]] = {}

        for entries in by_direction.values():
            entries.sort(key=lambda e: e[0])

            groups: list[list[tuple[float, Wall]]] = [[entries[0]]]

            for entry in entries[1:]:
                e_offset, e_wall = entry
                group = groups[-1]
                anchor_offset, anchor_wall = group[0]

                effective_tolerance = min(
                    collinear_snap_tolerance,
                    anchor_wall.width * snap_drift_ratio_max,
                )
                if abs(e_offset - anchor_offset) > effective_tolerance:
                    groups.append([entry])
                    continue

                groups[-1].append(entry)

            for group in groups:
                if len(group) < 2:
                    continue

                offsets = [g[0] for g in group]
                if max(offsets) - min(offsets) < 1e-9:
                    continue

                direction = self._unit_vector(group[0][1].start, group[0][1].end)
                if direction is None:
                    continue

                normal = Point(x=-direction.y, y=direction.x)

                sorted_offsets = sorted(offsets)
                n = len(sorted_offsets)
                median_offset = (
                    sorted_offsets[n // 2]
                    if n % 2 == 1
                    else (sorted_offsets[n // 2 - 1] + sorted_offsets[n // 2]) / 2.0
                )

                for offset, wall in group:
                    delta = median_offset - offset
                    if abs(delta) < 1e-9:
                        continue

                    corrections[wall.id] = (
                        Point(
                            x=round(wall.start.x + normal.x * delta, 6),
                            y=round(wall.start.y + normal.y * delta, 6),
                        ),
                        Point(
                            x=round(wall.end.x + normal.x * delta, 6),
                            y=round(wall.end.y + normal.y * delta, 6),
                        ),
                    )

        if not corrections:
            return walls

        result = []
        for wall in walls:
            if wall.id not in corrections:
                result.append(wall)
            else:
                new_start, new_end = corrections[wall.id]
                result.append(
                    Wall(
                        id=wall.id,
                        layer=wall.layer,
                        start=new_start,
                        end=new_end,
                        length=wall.length,
                        width=wall.width,
                        geometry_role=wall.geometry_role,
                        run_id=wall.run_id,
                        source_entity_ids=wall.source_entity_ids,
                    )
                )
        return result

    def _unit_vector(self, start: Point, end: Point) -> Point | None:
        length = self._length(start, end)
        if length == 0.0:
            return None

        ux = (end.x - start.x) / length
        uy = (end.y - start.y) / length
        if ux < 0.0 or (abs(ux) <= WALL_DETECTOR_CONFIG.parallel_cross_tolerance and uy < 0.0):
            ux, uy = -ux, -uy

        return Point(x=ux, y=uy)

    def _are_parallel(self, a: Point, b: Point) -> bool:
        return abs(a.x * b.y - a.y * b.x) <= WALL_DETECTOR_CONFIG.parallel_cross_tolerance

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
=======
from __future__ import annotations

from dataclasses import dataclass
from math import sqrt
from typing import NamedTuple

from internal.classification.config import WALL_DETECTOR_CONFIG, build_wall_thresholds
from internal.entities.floor import Wall
from internal.entities.geometry import LineEntity, NormalizedEntity, Point, PolylineEntity


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
        thresholds = build_wall_thresholds(units)
        min_width = thresholds.min_wall_width
        offset_tolerance = thresholds.line_offset_tolerance

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

        return self._assign_run_ids(
            walls,
            offset_tolerance,
            thresholds.max_run_gap,
            thresholds.max_fallback_run_gap,
            thresholds.run_width_tolerance,
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
        return any(marker in normalized_layer for marker in WALL_DETECTOR_CONFIG.wall_layer_markers)

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
        if shorter == 0.0 or overlap / shorter < WALL_DETECTOR_CONFIG.min_overlap_ratio:
            return None

        offsets = self._signed_offsets(boundary, candidate, direction)
        if offsets is None:
            return None

        offset_start, offset_end = offsets
        if abs(offset_start - offset_end) > offset_tolerance:
            return None

        width = (abs(offset_start) + abs(offset_end)) / 2.0
        if width < min_width or width / overlap > WALL_DETECTOR_CONFIG.max_width_to_length:
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
        if ux < 0.0 or (abs(ux) <= WALL_DETECTOR_CONFIG.parallel_cross_tolerance and uy < 0.0):
            ux, uy = -ux, -uy

        return Point(x=ux, y=uy)

    def _are_parallel(self, a: Point, b: Point) -> bool:
        return abs(a.x * b.y - a.y * b.x) <= WALL_DETECTOR_CONFIG.parallel_cross_tolerance

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
>>>>>>> 4bf54f8 (hz)
