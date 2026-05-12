import random
import pytest
from quality_calculator.evaluator import QualityEvaluator

ITERATIONS = 10000

# --- РАСШИРЕННЫЕ МОКИ СХЕМ ---
MOCK_TECH_SCHEMA = {
    "traits": {
        "lighting": {
            "properties": {
                "luminous_flux": {"minimum": 400, "maximum": 1500},
                "cri": {"minimum": 70, "maximum": 100}
            }
        },
        "switch": {
            "properties": {
                "max_load_watt": {"minimum": 1000, "maximum": 3680},
                "has_energy_monitoring": {"type": "boolean"}
            }
        },
        "sensor": {
            "properties": {
                "battery_life_months": {"minimum": 6, "maximum": 36},
                "measurement_error_margin": {"minimum": 0.1, "maximum": 2.0}
            }
        },
        "infra": {
            "properties": {
                "max_child_devices": {"minimum": 32, "maximum": 128},
                "has_local_control": {"type": "boolean"}
            }
        }
    }
}

MOCK_EVAL_STRATEGY = {
    "lighting": {
        "luminous_flux": {"weight": 0.6},
        "cri": {"weight": 0.4}
    },
    "switch": {
        "max_load_watt": {"weight": 1.0},
        "has_energy_monitoring": {"bonus": 0.3}
    },
    "sensor": {
        "battery_life_months": {"weight": 0.6},
        "measurement_error_margin": {"weight": 0.4, "inverse": True}
    },
    "infra": {
        "max_child_devices": {"weight": 1.0},
        "has_local_control": {"bonus": 0.4}
    }
}


@pytest.fixture
def evaluator():
    return QualityEvaluator(tech_schema=MOCK_TECH_SCHEMA, eval_strategy=MOCK_EVAL_STRATEGY, weights=(0.3, 0.4, 0.3))


# --- ФАБРИКИ УСТРОЙСТВ ---

def generate_premium_lamp():
    return {
        "name": "Premium Bulb",
        "price": random.uniform(1500.0, 3000.0),
        "eval_traits": ["lighting"],
        "protocol": random.choice([["thread"], ["zigbee", "wi-fi"], ["matter"]]),
        "reviews": {"rating": random.uniform(4.7, 5.0), "count": random.randint(500, 1500)},
        "specs": {"luminous_flux": random.randint(1200, 1500), "cri": random.randint(92, 100)}
    }


def generate_trash_lamp():
    return {
        "name": "Cheap Bulb",
        "price": random.uniform(200.0, 600.0),
        "eval_traits": ["lighting"],
        "protocol": random.choice([["wifi"], ["ble"]]),
        "reviews": {"rating": random.uniform(3.0, 3.8), "count": random.randint(1, 30)},
        "specs": {"luminous_flux": random.randint(300, 600), "cri": random.randint(60, 75)}
    }


def generate_switch(has_monitoring: bool):
    return {
        "name": "Smart Plug",
        "price": random.uniform(800.0, 1500.0),
        "eval_traits": ["switch"],
        "protocol": ["zigbee"],
        "reviews": {"rating": random.uniform(4.0, 4.8), "count": random.randint(50, 500)},
        "specs": {
            "max_load_watt": random.randint(1000, 3680),
            "has_energy_monitoring": has_monitoring
        }
    }


def generate_sensor(error_margin: float):
    return {
        "name": "Temp Sensor",
        "price": random.uniform(500.0, 1200.0),
        "eval_traits": ["sensor"],
        "protocol": ["zigbee"],
        "reviews": {"rating": random.uniform(4.0, 4.9), "count": random.randint(100, 800)},
        "specs": {
            "battery_life_months": random.randint(6, 36),
            "measurement_error_margin": error_margin
        }
    }


# --- ТЕСТЫ (СТРЕСС-РЕЖИМ) ---

def test_randomized_premium_vs_trash(evaluator):
    for i in range(ITERATIONS):
        premium = generate_premium_lamp()
        trash = generate_trash_lamp()

        res_premium = evaluator.evaluate_device(premium)
        res_trash = evaluator.evaluate_device(trash)

        assert res_premium["Q_total"] > 0.8, f"Iter {i}: Premium underestimated: {res_premium}"
        assert res_trash["Q_total"] < 0.6, f"Iter {i}: Trash overestimated: {res_trash}"
        assert res_premium["Q_total"] > res_trash[
            "Q_total"], f"Iter {i}: Premium must beat trash. P:{res_premium} T:{res_trash}"


def test_mixed_device_aggregation(evaluator):
    for i in range(ITERATIONS):
        mixed_dev = {
            "name": "Smart Hub-Bulb",
            "price": random.uniform(2000.0, 5000.0),
            "eval_traits": ["lighting", "infra"],
            "protocol": ["wifi", "zigbee"],
            "reviews": {"rating": random.uniform(4.0, 5.0), "count": random.randint(50, 1000)},
            "specs": {
                "luminous_flux": random.randint(400, 1500),
                "cri": random.randint(70, 100),
                "max_child_devices": random.randint(32, 128),
                "has_local_control": random.choice([True, False])
            }
        }
        result = evaluator.evaluate_device(mixed_dev)
        assert 0.0 <= result["N_S"] <= 1.5, f"Iter {i}: Score out of bounds: {result}"
        assert result["Q_total"] > 0.0, f"Iter {i}: Q_total zero or negative: {result}"


def test_bonus_mechanics(evaluator):
    for i in range(ITERATIONS):
        switch_basic = generate_switch(has_monitoring=False)
        switch_pro = switch_basic.copy()
        switch_pro["specs"] = switch_pro["specs"].copy()
        switch_pro["specs"]["has_energy_monitoring"] = True

        res_basic = evaluator.evaluate_device(switch_basic)
        res_pro = evaluator.evaluate_device(switch_pro)

        assert res_pro["N_S"] > res_basic[
            "N_S"], f"Iter {i}: Bonus failed. Pro: {res_pro['N_S']}, Basic: {res_basic['N_S']}"
        assert abs((res_pro["N_S"] - res_basic["N_S"]) - 0.3) < 0.01, f"Iter {i}: Bonus math is wrong"


def test_inverse_scoring_mechanics(evaluator):
    for i in range(ITERATIONS):
        # Генерируем два сенсора с одинаковыми параметрами, кроме ошибки
        bad_sensor = generate_sensor(error_margin=random.uniform(1.5, 2.0))
        good_sensor = bad_sensor.copy()
        good_sensor["specs"] = good_sensor["specs"].copy()
        good_sensor["specs"]["measurement_error_margin"] = random.uniform(0.1, 0.5)

        res_bad = evaluator.evaluate_device(bad_sensor)
        res_good = evaluator.evaluate_device(good_sensor)

        assert res_good["N_S"] > res_bad[
            "N_S"], f"Iter {i}: Inverse scoring failed. Good: {res_good['N_S']}, Bad: {res_bad['N_S']}"


def test_eval_prefix_cleanup(evaluator):
    for i in range(ITERATIONS):
        lamp = generate_premium_lamp()
        lamp["eval_traits"] = ["eval_lighting"]

        result = evaluator.evaluate_device(lamp)
        assert result["N_S"] > 0.5, f"Iter {i}: Prefix cleanup failed"


def test_zero_price_handling(evaluator):
    for i in range(ITERATIONS):
        lamp = generate_premium_lamp()
        lamp["price"] = 0.0

        result = evaluator.evaluate_device(lamp)
        assert result["Value"] == 0.0, f"Iter {i}: Value should be 0.0 when price is zero"