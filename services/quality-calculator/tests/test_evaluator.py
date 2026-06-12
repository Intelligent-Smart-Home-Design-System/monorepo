import pytest
from quality_calculator.evaluator import QualityEvaluator

# Мини-схема в стиле нового device_types.json (traits + types).
MOCK_TECH_SCHEMA = {
    "traits": {
        "lighting": {
            "properties": {
                "brightness_lm": {"type": "number", "minimum": 1, "maximum": 5000},
                "cri": {"type": "integer", "minimum": 60, "maximum": 100},
            }
        },
        "leak_detection": {
            "properties": {
                "min_detected_water_level_mm": {"type": "number", "minimum": 0.1, "maximum": 10},
                "detects_leaks_below": {"type": "boolean"},
            }
        },
        "camera": {
            "properties": {
                "resolution": {"type": "string"},
                "field_of_view_deg": {"type": "integer", "minimum": 60, "maximum": 360},
                "has_night_vision": {"type": "boolean"},
            }
        },
        "lock": {"properties": {"access_methods": {"type": "array"}}},
    },
    "types": {
        "smart_lamp": {"traits": ["lighting"]},
        "water_leak_sensor": {"traits": ["leak_detection"]},
        "smart_camera": {"traits": ["camera"]},
        "smart_lock": {"traits": ["lock"]},
    },
}

MOCK_STRATEGY = {
    "lighting": {"brightness_lm": {"weight": 0.7}, "cri": {"weight": 0.3}},
    "leak_detection": {
        "min_detected_water_level_mm": {"weight": 1.0, "inverse": True},
        "detects_leaks_below": {"bonus": 0.2},
    },
    "camera": {
        "resolution": {"weight": 0.5, "scale": {"720p": 0.25, "1080p": 0.5, "2K": 0.75, "4K": 1.0}},
        "field_of_view_deg": {"weight": 0.5},
        "has_night_vision": {"bonus": 0.1},
    },
    "lock": {"access_methods": {"weight": 1.0, "count_max": 5}},
}


@pytest.fixture
def ev():
    return QualityEvaluator(MOCK_TECH_SCHEMA, MOCK_STRATEGY, weights=(0.3, 0.4, 0.3))


# ---------- резолв трейтов по типу ----------

def test_traits_resolved_from_category(ev):
    assert ev.traits_for_type("smart_lamp") == ["lighting"]


# ---------- N(S): нормализация и сравнение ----------

def test_premium_lamp_beats_trash(ev):
    premium = {"category": "smart_lamp", "specs": {"brightness_lm": 4800, "cri": 98},
               "reviews": {"rating": 4.9, "count": 1000}, "protocol": ["zigbee"], "price": 2000}
    trash = {"category": "smart_lamp", "specs": {"brightness_lm": 300, "cri": 62},
             "reviews": {"rating": 3.6, "count": 5}, "protocol": ["ble"], "price": 400}
    assert ev.evaluate_device(premium)["Q_total"] > ev.evaluate_device(trash)["Q_total"]


def test_inverse_scoring(ev):
    # Меньшая чувствительность к протечке (большее значение) = хуже.
    good = {"category": "water_leak_sensor", "specs": {"min_detected_water_level_mm": 0.2},
            "reviews": {"rating": 4.5, "count": 100}, "protocol": ["zigbee"], "price": 1000}
    bad = {"category": "water_leak_sensor", "specs": {"min_detected_water_level_mm": 9.0},
           "reviews": {"rating": 4.5, "count": 100}, "protocol": ["zigbee"], "price": 1000}
    assert ev.evaluate_device(good)["N_S"] > ev.evaluate_device(bad)["N_S"]


def test_ordinal_scale(ev):
    base = {"category": "smart_camera", "reviews": {"rating": 4.5, "count": 100},
            "protocol": ["wifi"], "price": 3000}
    res4k = ev.evaluate_device({**base, "specs": {"resolution": "4K", "field_of_view_deg": 120}})
    res720 = ev.evaluate_device({**base, "specs": {"resolution": "720p", "field_of_view_deg": 120}})
    assert res4k["N_S"] > res720["N_S"]


def test_array_count(ev):
    few = {"category": "smart_lock", "specs": {"access_methods": ["key"]},
           "reviews": {"rating": 4.5, "count": 100}, "protocol": ["zigbee"], "price": 5000}
    many = {"category": "smart_lock", "specs": {"access_methods": ["key", "app", "fingerprint", "keypad", "rfid_card"]},
            "reviews": {"rating": 4.5, "count": 100}, "protocol": ["zigbee"], "price": 5000}
    assert ev.evaluate_device(many)["N_S"] > ev.evaluate_device(few)["N_S"]


def test_boolean_bonus_raises_score(ev):
    no = ev.evaluate_device({"category": "smart_camera", "specs": {"resolution": "1080p", "field_of_view_deg": 120},
                             "reviews": {"rating": 4.5, "count": 100}, "protocol": ["wifi"], "price": 3000})
    yes = ev.evaluate_device({"category": "smart_camera",
                              "specs": {"resolution": "1080p", "field_of_view_deg": 120, "has_night_vision": True},
                              "reviews": {"rating": 4.5, "count": 100}, "protocol": ["wifi"], "price": 3000})
    assert yes["N_S"] > no["N_S"]


# ---------- E: протокол ----------

def test_matter_over_thread_is_mesh(ev):
    assert ev.eval_protocol(["matter-over-thread"]) == 1.0


def test_wifi_is_star(ev):
    assert ev.eval_protocol(["wifi"]) == 0.7


def test_no_protocol_is_none(ev):
    assert ev.eval_protocol([]) is None


# ---------- N(R): репутация ----------

def test_bayesian_suppresses_low_count(ev):
    many = ev.eval_reputation(4.9, 5000)
    few = ev.eval_reputation(4.9, 3)
    assert many > few  # тот же рейтинг, но мало отзывов -> ниже


def test_zero_count_reputation_none(ev):
    assert ev.eval_reputation(4.9, 0) is None


# ---------- агрегированная оценка ----------

def test_missing_specs_renormalizes(ev):
    # Нет characteristics -> Q считается по репутации+протоколу, не нулём.
    d = {"category": "smart_lamp", "specs": {}, "reviews": {"rating": 4.8, "count": 500},
         "protocol": ["zigbee"], "price": 2000}
    res = ev.evaluate_device(d)
    assert res["N_S"] is None
    assert res["Q_total"] is not None and res["Q_total"] > 0


def test_zero_price_value_none(ev):
    d = {"category": "smart_lamp", "specs": {"brightness_lm": 4000, "cri": 95},
         "reviews": {"rating": 4.8, "count": 500}, "protocol": ["zigbee"], "price": 0}
    assert ev.evaluate_device(d)["Value"] is None
