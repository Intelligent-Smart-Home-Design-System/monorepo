from __future__ import annotations

from dataclasses import dataclass
from typing import NamedTuple

from device_selection.core.model import ParetoPoint


def _weakly_dominates(a: ParetoPoint, b: ParetoPoint) -> bool:
    """
    Returns True if a weakly dominates b (better or equal in all objectives).

    Objectives:
      avg_quality  — maximise
      num_ecosystems — minimise
      num_hubs       — minimise
    """
    return (
        a.avg_quality >= b.avg_quality
        and a.num_ecosystems <= b.num_ecosystems
        and a.num_hubs <= b.num_hubs
    )


class ObjectiveBounds(NamedTuple):
    """
    Bounds used to normalize objectives before passing to pymoo indicators.

    quality      in [q_min,   q_max]   (usually [0, 1])
    ecosystems   in [eco_min, eco_max] (e.g. [1, 6] if max 5 bridges)
    hubs         in [hub_min, hub_max] (e.g. [0, 4])
    """
    q_min: float = 0.0
    q_max: float = 1.0
    eco_min: int = 1
    eco_max: int = 6
    hub_min: int = 0
    hub_max: int = 6


def _clamp01(x: float) -> float:
    if x < 0.0:
        return 0.0
    if x > 1.0:
        return 1.0
    return x


def _norm(x: float, lo: float, hi: float) -> float:
    if hi <= lo:
        return 0.0
    return _clamp01((x - lo) / (hi - lo))


def _as_minimization_vector(
    p: ParetoPoint,
    b: ObjectiveBounds,
) -> tuple[float, float, float]:
    """
    Map a ParetoPoint to a 3-D vector in [0, 1]^3 that is to be MINIMISED.

      dim 0: q_cost    = 1 - normalised quality   (low quality => high cost)
      dim 1: eco_norm  = normalised num_ecosystems
      dim 2: hub_norm  = normalised num_hubs
    """
    q_norm   = _norm(p.avg_quality,        b.q_min,   b.q_max)
    eco_norm = _norm(float(p.num_ecosystems), float(b.eco_min), float(b.eco_max))
    hub_norm = _norm(float(p.num_hubs),       float(b.hub_min), float(b.hub_max))
    return (1.0 - q_norm, eco_norm, hub_norm)


@dataclass(slots=True)
class ParetoArchive:
    points: list[ParetoPoint]

    def __init__(self) -> None:
        self.points = []

    def add(self, p: ParetoPoint) -> None:
        """
        Add p to the archive if it is not weakly dominated by any existing point.
        Removes any existing points that p weakly dominates.
        """
        for q in self.points:
            if _weakly_dominates(q, p):
                return
        self.points = [q for q in self.points if not _weakly_dominates(p, q)]
        self.points.append(p)

    # ---- helpers used by solvers / metrics module ----

    def as_min_vectors(
        self,
        bounds: ObjectiveBounds,
    ) -> list[tuple[float, float, float]]:
        """
        Return all stored points as normalised minimisation vectors.
        Pass the result directly to pymoo indicators (HV, IGD, Epsilon …).
        """
        return [_as_minimization_vector(p, bounds) for p in self.points]


    def front_size(self) -> int:
        return len(self.points)

    def objective_ranges(self) -> dict[str, tuple[float, float]]:
        """
        Min/max of each objective over the stored front (original units).
        """
        if not self.points:
            return {
                "avg_quality":    (0.0, 0.0),
                "num_ecosystems": (0.0, 0.0),
                "num_hubs":       (0.0, 0.0),
                "total_cost":     (0.0, 0.0),
            }

        return {
            "avg_quality":    (min(p.avg_quality    for p in self.points),
                               max(p.avg_quality    for p in self.points)),
            "num_ecosystems": (float(min(p.num_ecosystems for p in self.points)),
                               float(max(p.num_ecosystems for p in self.points))),
            "num_hubs":       (float(min(p.num_hubs       for p in self.points)),
                               float(max(p.num_hubs       for p in self.points))),
            "total_cost":     (min(p.total_cost     for p in self.points),
                               max(p.total_cost     for p in self.points)),
        }
