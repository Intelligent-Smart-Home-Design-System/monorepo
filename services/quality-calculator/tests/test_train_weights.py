import numpy as np

from quality_calculator.evaluation.train_weights import (
    softmax, predict_ns, mse, fit_weights, fit_component_weights, field_contribution,
)


def test_softmax_is_simplex():
    w = softmax(np.array([0.0, 1.0, -2.0]))
    assert abs(w.sum() - 1.0) < 1e-9
    assert np.all(w > 0)


def test_predict_ns_handles_missing():
    C = np.array([[0.8, np.nan], [0.2, 0.6]])
    mask = ~np.isnan(C)
    w = np.array([0.5, 0.5])
    pred = predict_ns(C, mask, w)
    # первая строка: только поле 0 присутствует -> N(S)=0.8 (перенормировка веса)
    assert abs(pred[0] - 0.8) < 1e-9
    # вторая: (0.5*0.2+0.5*0.6)/1.0 = 0.4
    assert abs(pred[1] - 0.4) < 1e-9


def test_field_contribution_numeric_and_scale():
    # numeric: значение посередине диапазона -> 0.5
    c = field_contribution(900, {"weight": 1.0}, {"minimum": 800, "maximum": 1000})
    assert abs(c - 0.5) < 1e-9
    # scale (ordinal)
    c2 = field_contribution("4K", {"scale": {"720p": 0.25, "4K": 1.0}}, {})
    assert c2 == 1.0
    # count_max
    c3 = field_contribution(["a", "b"], {"count_max": 4}, {})
    assert c3 == 0.5
    # отсутствующее значение
    assert field_contribution(None, {"weight": 1.0}, {"minimum": 0, "maximum": 1}) is None


def test_gd_recovers_informative_feature():
    rng = np.random.default_rng(0)
    n = 200
    x0, x1 = rng.random(n), rng.random(n)
    y = x0.copy()  # цель = только первый признак
    C = np.column_stack([x0, x1])
    w, hist = fit_weights(C, y, n_iter=800, lr=0.8)
    assert w[0] > 0.9          # вес ушёл на информативный признак
    assert hist[-1] < hist[0]  # потеря упала


def test_component_weights_prefer_predictive_signal():
    rng = np.random.default_rng(1)
    n = 100
    nr = rng.random(n)
    ns = rng.random(n)
    e = rng.random(n)
    y = ns.copy()  # общая оценка = только N(S)
    F = np.column_stack([nr, ns, e])
    w, hist = fit_component_weights(F, y, n_iter=800, lr=0.8)
    assert w[1] > 0.8  # вес specs доминирует
    assert hist[-1] < hist[0]


def test_mse_zero_on_perfect():
    y = np.array([0.0, 0.5, 1.0])
    assert mse(y, y) == 0.0
