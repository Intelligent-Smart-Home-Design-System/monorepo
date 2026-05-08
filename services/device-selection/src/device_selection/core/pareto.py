from __future__ import annotations

from dataclasses import dataclass
from itertools import combinations
from typing import Iterable, NamedTuple, Sequence

from device_selection.core.model import ParetoPoint


# ------------------- dominance (your existing) -------------------

def _dominates(a: ParetoPoint, b: ParetoPoint) -> bool:
    better_or_equal = (
        a.avg_quality >= b.avg_quality
        and a.num_ecosystems <= b.num_ecosystems
        and a.num_hubs <= b.num_hubs
    )
    strictly_better = (
        a.avg_quality > b.avg_quality
        or a.num_ecosystems < b.num_ecosystems
        or a.num_hubs < b.num_hubs
    )
    return better_or_equal


# ------------------- normalization config -------------------

class ObjectiveBounds(NamedTuple):
    """
    Bounds used to normalize objectives for HV/epsilon/diversity.

    quality in [q_min, q_max] (usually [0,1])
    ecosystems in [eco_min, eco_max] (e.g., [1, 6] if max 5 bridges)
    hubs in [hub_min, hub_max] (e.g., [0, 4])
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


def _as_minimization_vector(p: ParetoPoint, b: ObjectiveBounds) -> tuple[float, float, float]:
    """
    Returns a 3D vector in [0,1]^3 to be MINIMIZED:
      (q_cost, eco_cost, hub_cost)
    """
    q_norm = _norm(p.avg_quality, b.q_min, b.q_max)
    eco_norm = _norm(float(p.num_ecosystems), float(b.eco_min), float(b.eco_max))
    hub_norm = _norm(float(p.num_hubs), float(b.hub_min), float(b.hub_max))

    q_cost = 1.0 - q_norm
    return (q_cost, eco_norm, hub_norm)


# ------------------- metrics helpers -------------------

def _hypervolume_2d(points: list[tuple[float, float]], ref: tuple[float, float]) -> float:
    """
    Exact 2D dominated hypervolume for MINIMIZATION, rectangles [y..ref_y]x[z..ref_z].
    points can contain dominated points; we compute union.
    Assumes all coordinates are within [0, ref].
    """
    ref_y, ref_z = ref
    if not points:
        return 0.0

    # Sort by y ascending (better to worse in y)
    pts = sorted(points, key=lambda t: t[0])

    area = 0.0
    best_z = ref_z  # running min z (smaller z is better)
    prev_y = None

    # Build a skyline with strictly decreasing best_z
    skyline: list[tuple[float, float]] = []
    for y, z in pts:
        if z < best_z:
            best_z = z
            skyline.append((y, best_z))

    # Union area under skyline
    for i, (y, z) in enumerate(skyline):
        y_next = skyline[i + 1][0] if i + 1 < len(skyline) else ref_y
        if y_next > y:
            area += (y_next - y) * (ref_z - z)

    return area


def hypervolume_3d(points: Sequence[tuple[float, float, float]], ref: tuple[float, float, float]) -> float:
    """
    Exact 3D dominated hypervolume for MINIMIZATION with a reference point.
    Computes union of boxes [x..ref_x]x[y..ref_y]x[z..ref_z].

    Complexity ~ O(n^2 log n) for small n; fine for Pareto fronts of size ~ 5..500.
    """
    ref_x, ref_y, ref_z = ref
    if not points:
        return 0.0

    # Sort by x ascending
    pts = sorted(points, key=lambda t: t[0])

    hv = 0.0
    active_yz: list[tuple[float, float]] = []

    for i, (x, y, z) in enumerate(pts):
        next_x = pts[i + 1][0] if i + 1 < len(pts) else ref_x
        if next_x <= x:
            continue

        # Add current point to active set for cross-section
        active_yz.append((y, z))

        # Cross-section area in YZ for all points with x <= current slice start
        area_yz = _hypervolume_2d(active_yz, ref=(ref_y, ref_z))
        hv += (next_x - x) * area_yz

    return hv


def additive_epsilon_indicator(
    A: Sequence[tuple[float, ...]],
    B: Sequence[tuple[float, ...]],
) -> float:
    """
    Additive epsilon indicator Iε+(A,B) for MINIMIZATION.

    Interpretation:
      Small ε means A is close to (or dominates) B.
      ε <= 0 means: for every b in B, there exists a in A such that a <= b (A weakly dominates B).

    Definition:
      Iε+(A,B) = max_{b in B} min_{a in A} max_i (a_i - b_i)
    """
    if not B:
        return float("-inf")  # nothing to cover
    if not A:
        return float("inf")

    eps = float("-inf")
    for b in B:
        best_for_b = float("inf")
        for a in A:
            worst_dim = max(a_i - b_i for a_i, b_i in zip(a, b, strict=True))
            if worst_dim < best_for_b:
                best_for_b = worst_dim
        if best_for_b > eps:
            eps = best_for_b
    return eps


def avg_pairwise_distance(points: Sequence[tuple[float, ...]]) -> float:
    """
    Average pairwise Euclidean distance in objective space.
    Use normalized minimization vectors for meaningful numbers.
    """
    n = len(points)
    if n < 2:
        return 0.0

    total = 0.0
    cnt = 0
    for i, j in combinations(range(n), 2):
        a = points[i]
        b = points[j]
        d2 = 0.0
        for a_i, b_i in zip(a, b, strict=True):
            d = a_i - b_i
            d2 += d * d
        total += d2**0.5
        cnt += 1
    return total / cnt


# ------------------- ParetoArchive with metrics -------------------

@dataclass(slots=True)
class ParetoArchive:
    points: list[ParetoPoint]

    def __init__(self) -> None:
        self.points = []

    def add(self, p: ParetoPoint) -> None:
        for q in self.points:
            if _dominates(q, p):
                return
        self.points = [q for q in self.points if not _dominates(p, q)]
        self.points.append(p)

    # ---- Monitoring / summary ----

    def front_size(self) -> int:
        """I. Count of non-dominated solutions stored."""
        return len(self.points)

    def objective_ranges(self) -> dict[str, tuple[float, float]]:
        """
        A. Ranges (min,max) on the stored set (in original units).
        Useful to show whether solutions are meaningfully different.
        """
        if not self.points:
            return {
                "avg_quality": (0.0, 0.0),
                "num_ecosystems": (0.0, 0.0),
                "num_hubs": (0.0, 0.0),
                "total_cost": (0.0, 0.0),
            }

        qs = [p.avg_quality for p in self.points]
        es = [p.num_ecosystems for p in self.points]
        hs = [p.num_hubs for p in self.points]
        cs = [p.total_cost for p in self.points]
        return {
            "avg_quality": (min(qs), max(qs)),
            "num_ecosystems": (float(min(es)), float(max(es))),
            "num_hubs": (float(min(hs)), float(max(hs))),
            "total_cost": (min(cs), max(cs)),
        }

    # ---- Normalized vectors ----

    def as_min_vectors(self, bounds: ObjectiveBounds) -> list[tuple[float, float, float]]:
        """Convert stored points to normalized minimization vectors for metrics."""
        return [_as_minimization_vector(p, bounds) for p in self.points]

    # ---- HV ----

    def hypervolume(
        self,
        bounds: ObjectiveBounds,
        ref: tuple[float, float, float] = (1.0, 1.0, 1.0),
    ) -> float:
        """
        1) Hypervolume (HV):
        Measures the volume dominated by the Pareto front up to a reference point.

        Intuition:
          - bigger HV => better front (better quality / fewer ecosystems / fewer hubs) AND usually better spread.
          - good for comparing algorithms or time budgets (anytime curves).
          - not a user-facing metric (users don't understand it), but great for evaluation.

        IMPORTANT:
          HV depends heavily on normalization + reference point.
          Use fixed bounds so HV is comparable between runs.
        """
        vecs = self.as_min_vectors(bounds)
        return hypervolume_3d(vecs, ref=ref)

    # ---- Epsilon indicator ----

    def epsilon_to(
        self,
        baseline: Sequence[ParetoPoint],
        bounds: ObjectiveBounds,
    ) -> float:
        """
        2) Additive epsilon indicator Iε+(A,B):
        'How far is A from covering B?'

        Here:
          A = this archive's front
          B = baseline front

        Interpretation:
          - ε <= 0: A weakly dominates baseline (great)
          - smaller ε: closer/better
        """
        A = self.as_min_vectors(bounds)
        B = [_as_minimization_vector(p, bounds) for p in baseline]
        return additive_epsilon_indicator(A, B)

    # ---- Diversity ----

    def diversity_avg_distance(self, bounds: ObjectiveBounds) -> float:
        """
        B) Diversity in objective space:
        average pairwise distance between points (normalized).

        Larger => solutions are more different trade-offs.
        Too large isn't always good, but tiny values indicate "all solutions are almost the same".
        """
        vecs = self.as_min_vectors(bounds)
        return avg_pairwise_distance(vecs)
