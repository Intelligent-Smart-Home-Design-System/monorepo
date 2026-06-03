from __future__ import annotations


def bayesian_quality(
    total_reviews: int,
    avg_rating: float,
    min_reviews: int = 10,
    global_avg: float = 4.0,
    rating_floor: float = 4.0,
    max_rating: float = 5.0,
) -> float:
    """
    Computes bayesian average rating, subtracts rating floor, and normalizes to [0, 1].
    """
    bayesian = (min_reviews * global_avg + total_reviews * avg_rating) / (min_reviews + total_reviews)
    return max(0, (bayesian - rating_floor) / (max_rating - rating_floor))
