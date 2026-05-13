from __future__ import annotations

from dataclasses import dataclass


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
class WallThresholds:
    min_wall_width: float
    line_offset_tolerance: float
    max_run_gap: float
    max_fallback_run_gap: float
    run_width_tolerance: float


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
class WallDetectorConfig:
    min_wall_width_mm: float = 2.0
    parallel_cross_tolerance: float = 0.01
    min_overlap_ratio: float = 0.4
    max_width_to_length: float = 0.35
    line_offset_tolerance_mm: float = 2.0
    max_run_gap_mm: float = 25.0
    max_fallback_run_gap_mm: float = 10.0
    run_width_tolerance_mm: float = 20.0
    wall_layer_markers: tuple[str, ...] = (
        "wall",
        "walls",
        "стена",
        "стены",
    )


@dataclass(frozen=True)
class OpeningDetectorConfig:
    segment_search_radius_mm: float = 80.0
    arc_search_radius_mm: float = 40.0
    min_opening_length_mm: float = 20.0
    precise_length_tolerance_mm: float = 2.5
    segment_overhang_mm: float = 24.0
    perp_dist_max_mm: float = 40.0
    parallel_midline_max_offset_mm: float = 8.0
    parallel_length_min_ratio: float = 0.7
    gap_group_direction_precision: int = 3
    gap_arc_distance_mm: float = 48.0
    gap_hint_distance_mm: float | None = None
    gap_axis_overhang_mm: float = 24.0
    gap_axis_offset_tolerance_mm: float = 8.0
    parallel_span_gap_max_mm: float = 48.0
    orientation_axis_tolerance_mm: float = 1.0
    vector_epsilon: float = 1e-6
    opening_layer_keywords: tuple[str, ...] = ("opening", "open", "door", "window", "wind", "glaz")
    header_layer_keywords: tuple[str, ...] = ("header", "frame")
    garage_layer_keywords: tuple[str, ...] = ("garage", "overhead")
    door_layer_keywords: tuple[str, ...] = ("door", "doors", "дверь", "двери")
    window_layer_keywords: tuple[str, ...] = ("window", "windows", "окно", "окна")
    door_block_keywords: tuple[str, ...] = ("door", "dr", "doorleaf", "door-panel")
    window_block_keywords: tuple[str, ...] = ("window", "win", "wn", "окно", "окн")
    window_operation_tokens: tuple[str, ...] = ("XO", "OX", "SH", "HS", "DH", "FX")
    sliding_door_tokens: tuple[str, ...] = ("SGD", "SLIDING", "PATIO")


WALL_DETECTOR_CONFIG = WallDetectorConfig()
OPENING_DETECTOR_CONFIG = OpeningDetectorConfig()


def mm_to_units(value_mm: float, units: str | None) -> float:
    if units is None:
        return value_mm

    factor = UNITS_TO_MILLIMETERS.get(units.strip().lower())
    if factor is None or factor == 0.0:
        return value_mm

    return value_mm / factor


def build_wall_thresholds(
    units: str | None,
    config: WallDetectorConfig = WALL_DETECTOR_CONFIG,
) -> WallThresholds:
    return WallThresholds(
        min_wall_width=mm_to_units(config.min_wall_width_mm, units),
        line_offset_tolerance=mm_to_units(config.line_offset_tolerance_mm, units),
        max_run_gap=mm_to_units(config.max_run_gap_mm, units),
        max_fallback_run_gap=mm_to_units(config.max_fallback_run_gap_mm, units),
        run_width_tolerance=mm_to_units(config.run_width_tolerance_mm, units),
    )


def build_opening_thresholds(
    units: str | None,
    config: OpeningDetectorConfig = OPENING_DETECTOR_CONFIG,
) -> OpeningThresholds:
    gap_hint_distance_mm = (
        config.gap_hint_distance_mm
        if config.gap_hint_distance_mm is not None
        else config.gap_arc_distance_mm * 2.0
    )
    return OpeningThresholds(
        segment_search_radius=mm_to_units(config.segment_search_radius_mm, units),
        arc_search_radius=mm_to_units(config.arc_search_radius_mm, units),
        min_opening_length=mm_to_units(config.min_opening_length_mm, units),
        precise_length_tolerance=mm_to_units(config.precise_length_tolerance_mm, units),
        segment_overhang=mm_to_units(config.segment_overhang_mm, units),
        perp_dist_max=mm_to_units(config.perp_dist_max_mm, units),
        parallel_midline_max_offset=mm_to_units(config.parallel_midline_max_offset_mm, units),
        gap_arc_distance=mm_to_units(config.gap_arc_distance_mm, units),
        gap_hint_distance=mm_to_units(gap_hint_distance_mm, units),
        gap_axis_overhang=mm_to_units(config.gap_axis_overhang_mm, units),
        gap_axis_offset_tolerance=mm_to_units(config.gap_axis_offset_tolerance_mm, units),
        parallel_span_gap_max=mm_to_units(config.parallel_span_gap_max_mm, units),
        orientation_axis_tolerance=mm_to_units(config.orientation_axis_tolerance_mm, units),
    )
