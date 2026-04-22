import random
import pytest
from evaluator import QualityEvaluator

# --- МОК СХЕМЫ ДЛЯ ТЕСТОВ ---
MOCK_SCHEMA = {
    "traits": {
        "eval_lighting": {
            "properties": {
                "luminous_flux": {"min": 400, "max": 1500, "weight": 0.6},
                "cri": {"min": 70, "max": 100, "weight": 0.4}
            }
        },
        "eval_infra": {
            "properties": {
                "max_child_devices": {"min": 32, "max": 128, "weight": 1.0}
            }
        }
    }
}


@pytest.fixture
def evaluator():
    return QualityEvaluator(traits_schema=MOCK_SCHEMA, weights=(0.3, 0.4, 0.3))


# --- ФАБРИКИ (ГЕНЕРАТОРЫ СЛУЧАЙНЫХ УСТРОЙСТВ) ---

def generate_premium_lamp():
    """Генерирует премиальную лампу с отличными статами и Mesh-сетью."""
    return {
        "name": f"Premium Smart Bulb Gen-{random.randint(1, 9)}",
        "price": random.uniform(1500.0, 3000.0),
        "eval_traits": ["eval_lighting"],
        "protocol": random.choice([["thread"], ["zigbee", "wi-fi"], ["matter"]]),
        "reviews": {
            "rating": random.uniform(4.7, 5.0),
            "count": random.randint(200, 1500)
        },
        "specs": {
            "luminous_flux": random.randint(1200, 1500),
            "cri": random.randint(92, 100)
        }
    }


def generate_trash_lamp():
    """Генерирует мусорную лампу: тусклую, только Wi-Fi/BLE, с плохими отзывами."""
    return {
        "name": f"NoName Cheap Bulb v{random.randint(1, 9)}",
        "price": random.uniform(200.0, 600.0),
        "eval_traits": ["eval_lighting"],
        "protocol": random.choice([["wifi"], ["ble"]]),
        "reviews": {
            "rating": random.uniform(3.0, 4.0),
            "count": random.randint(1, 20)  # Мало отзывов -> Байес должен занизить скор
        },
        "specs": {
            "luminous_flux": random.randint(300, 600),
            "cri": random.randint(60, 75)
        }
    }


def generate_mixed_device():
    """Генерирует устройство 2-в-1 (Лампа + Шлюз)."""
    return {
        "name": "Smart Hub-Bulb Combo",
        "price": 4000.0,
        "eval_traits": ["eval_lighting", "eval_infra"],  # Агрегация двух трейтов
        "protocol": ["wifi", "zigbee"],
        "reviews": {"rating": 4.8, "count": 300},
        "specs": {
            "luminous_flux": 1000,
            "cri": 90,
            "max_child_devices": 64
        }
    }


# --- ТЕСТЫ ---

def test_randomized_premium_vs_trash(evaluator):
    """
    Генерируем случайный премиум и случайный мусор.
    Проверяем, что качество (Q_total) премиума всегда строго выше.
    """
    premium = generate_premium_lamp()
    trash = generate_trash_lamp()

    res_premium = evaluator.evaluate_device(premium)
    res_trash = evaluator.evaluate_device(trash)

    assert res_premium["Q_total"] > 0.8, f"Сгенерированный премиум недооценен: {res_premium}"
    assert res_trash["Q_total"] < 0.6, f"Сгенерированный мусор переоценен: {res_trash}"
    assert res_premium["Q_total"] > res_trash["Q_total"], "Премиум должен быть всегда лучше мусора"


def test_mixed_device_aggregation(evaluator):
    """
    Проверяем, что комбинированные устройства (eval_traits = list)
    корректно считывают характеристики из разных блоков схемы.
    """
    mixed_dev = generate_mixed_device()
    result = evaluator.evaluate_device(mixed_dev)

    # Убеждаемся, что алгоритм не упал и вернул адекватную оценку
    assert 0.0 <= result["N_S"] <= 1.0
    assert result["Q_total"] > 0.0