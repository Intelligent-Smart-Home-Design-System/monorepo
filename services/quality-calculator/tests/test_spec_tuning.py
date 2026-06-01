import json
from pathlib import Path

from quality_calculator.evaluation.ground_truth import label_device, axis_tier
from quality_calculator.evaluation.spec_tuning import _assign_by_marginals, _macro_f1

RUBRIC = json.loads((Path(__file__).parent.parent / "config" / "ground_truth_rubric.json").read_text(encoding="utf-8"))


# --- axis tiers ---

def test_numeric_axis_tiers():
    ax = {"spec": "brightness_lm", "kind": "numeric", "thresholds": [800, 1600], "inverse": False}
    assert axis_tier(ax, {"brightness_lm": 500}) == 0
    assert axis_tier(ax, {"brightness_lm": 1000}) == 1
    assert axis_tier(ax, {"brightness_lm": 1600}) == 2
    assert axis_tier(ax, {}) is None


def test_inverse_axis_tier():
    ax = {"spec": "x", "kind": "numeric", "thresholds": [0.5, 1.5], "inverse": True}
    assert axis_tier(ax, {"x": 0.2}) == 2   # меньше = лучше
    assert axis_tier(ax, {"x": 1.0}) == 1
    assert axis_tier(ax, {"x": 3.0}) == 0


def test_ordinal_and_count_axes():
    ord_ax = {"spec": "resolution", "kind": "ordinal", "map": {"720p": 0, "1080p": 1, "2K": 2, "4K": 2}}
    assert axis_tier(ord_ax, {"resolution": "4K"}) == 2
    assert axis_tier(ord_ax, {"resolution": "720p"}) == 0
    cnt_ax = {"spec": "access_methods", "kind": "count", "thresholds": [2, 4]}
    assert axis_tier(cnt_ax, {"access_methods": ["app"]}) == 0
    assert axis_tier(cnt_ax, {"access_methods": ["app", "key", "rfid"]}) == 1
    assert axis_tier(cnt_ax, {"access_methods": ["app", "key", "rfid", "face", "fp"]}) == 2


# --- label_device (mean combine, real rubric) ---

def test_lamp_label_combines_axes():
    # brightness excellent(2) + cri good(1) -> mean 1.5 -> round half up -> excellent
    label, n = label_device("smart_lamp", {"brightness_lm": 1700, "cri": 85}, RUBRIC)
    assert label == "excellent" and n == 2


def test_label_none_without_data():
    label, n = label_device("smart_lamp", {"unrelated": 1}, RUBRIC)
    assert label is None and n == 0


def test_unknown_category_unlabeled():
    assert label_device("nonexistent", {"x": 1}, RUBRIC) == (None, 0)


# --- marginal assignment + macro F1 ---

def test_assign_by_marginals_orders_low_to_high():
    ns = [0.9, 0.1, 0.5, 0.2]
    # gt sizes: 2 bad, 1 good, 1 excellent -> lowest two are bad
    pred = _assign_by_marginals(ns, [2, 1, 1])
    assert pred == [2, 0, 1, 0]


def test_macro_f1_perfect():
    gt = [0, 1, 2, 0, 1, 2]
    macro, per = _macro_f1(gt, gt)
    assert macro == 1.0
    assert per["bad"]["f1"] == 1.0
