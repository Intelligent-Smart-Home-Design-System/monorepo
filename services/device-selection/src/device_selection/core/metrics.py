"""
Indicator functions for evaluating Pareto fronts.
"""

from __future__ import annotations

import numpy as np
from pymoo.indicators.hv import HV
from pymoo.indicators.igd import IGD
from pymoo.indicators.igd_plus import IGDPlus

from device_selection.core.pareto import ObjectiveBounds, ParetoArchive, _weakly_dominates


def best_known_front(archives: list[ParetoArchive]) -> ParetoArchive:
    """
    Merge every point from every archive into a single non-dominated front.
    """
    merged = ParetoArchive()
    for archive in archives:
        for point in archive.points:
            merged.add(point)
    return merged


def hypervolume(
    archive: ParetoArchive,
    bounds: ObjectiveBounds,
    ref: tuple[float, float, float] = (1.05, 1.05, 1.05),
) -> float:
    """
    Hypervolume dominated by the archive's front (larger = better).
    """
    vecs = archive.as_min_vectors(bounds)
    if not vecs:
        return 0.0

    ind = HV(ref_point=np.array(ref, dtype=float))
    return float(ind(np.array(vecs, dtype=float)))


def igd(
    archive: ParetoArchive,
    reference: ParetoArchive,
    bounds: ObjectiveBounds,
) -> float:
    """
    Inverted Generational Distance — IGD
    """
    A = archive.as_min_vectors(bounds)
    R = reference.as_min_vectors(bounds)
    if not A or not R:
        return float("inf")

    ind = IGD(np.array(R, dtype=float))
    return float(ind(np.array(A, dtype=float)))


def igd_plus(
    archive: ParetoArchive,
    reference: ParetoArchive,
    bounds: ObjectiveBounds,
) -> float:
    """
    IGD+
    """
    A = archive.as_min_vectors(bounds)
    R = reference.as_min_vectors(bounds)
    if not A or not R:
        return float("inf")

    ind = IGDPlus(np.array(R, dtype=float))
    return float(ind(np.array(A, dtype=float)))
