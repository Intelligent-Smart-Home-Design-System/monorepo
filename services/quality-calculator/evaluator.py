import json
from typing import Dict, Any, List, Union


class QualityEvaluator:
    def __init__(self, traits_schema: Dict[str, Any], weights: tuple = (0.3, 0.4, 0.3)):
        self.traits = traits_schema.get("traits", {})
        self.w_r, self.w_s, self.w_e = weights
        self.bayes_m = 50
        self.bayes_C = 4.2

    @staticmethod
    def normalize(value: float, min_val: float, max_val: float, inverse: bool = False) -> float:
        if value is None or not isinstance(value, (int, float)): return 0.0
        if max_val <= min_val: return 1.0 if value >= max_val else 0.0
        clipped = max(min_val, min(value, max_val))
        norm = (clipped - min_val) / (max_val - min_val)
        return 1.0 - norm if inverse else norm

    def eval_reputation(self, rating: float, count: int) -> float:
        if rating is None or count == 0: return 0.0
        bayes_rating = (count * rating + self.bayes_m * self.bayes_C) / (count + self.bayes_m)
        return self.normalize(bayes_rating, 3.5, 5.0)

    def eval_protocol(self, protocols: List[str]) -> float:
        """Оценка на основе сетевой топологии с учетом синергии протоколов."""
        if not protocols: return 0.2

        p_lower = set(p.lower() for p in protocols)
        mesh_protocols = {"zigbee", "matter", "thread"}

        score = 0.2
        if p_lower.intersection(mesh_protocols):
            score = 1.0  # Mesh (Ячеистая сеть)
        elif "wi-fi" in p_lower or "wifi" in p_lower:
            score = 0.7  # Star (Звезда)
        elif "bluetooth" in p_lower or "ble" in p_lower:
            score = 0.4  # Point-to-Point

        # Синергетический бонус: если устройство поддерживает несколько разных протоколов,
        # оно потенциально выступает мостом (например, Wi-Fi роутер + Matter)
        if score < 1.0 and len(p_lower) > 1:
            score = min(1.0, score + 0.1)

        return score

    def _eval_single_trait(self, trait_key: str, specs: Dict[str, Any]) -> float:
        """Внутренний метод оценки одного конкретного трейта."""
        trait_schema = self.traits.get(trait_key)
        if not trait_schema: return 0.5

        properties = trait_schema.get("properties", {})
        score, total_weight = 0.0, 0.0

        for prop_name, rules in properties.items():
            val = specs.get(prop_name)
            weight = rules.get("weight", 0.0)

            if val is not None and "min" in rules and "max" in rules:
                inverse = rules.get("inverse", False)
                score += self.normalize(val, rules["min"], rules["max"], inverse) * weight
                total_weight += weight
            elif rules.get("type") == "boolean" and val is True:
                bonus = rules.get("bonus", 0.0)
                score += bonus
                total_weight += bonus

        return score / total_weight if total_weight > 0 else 0.5

    def eval_specs(self, trait_keys: Union[str, List[str]], specs: Dict[str, Any]) -> float:
        """
        Агрегация спецификаций. Поддерживает комбинированные устройства (например, камера + шлюз).
        """
        if isinstance(trait_keys, str):
            trait_keys = [trait_keys]

        if not trait_keys: return 0.5

        # Считаем оценку для каждой роли устройства и берем среднее
        scores = [self._eval_single_trait(tk, specs) for tk in trait_keys]
        return sum(scores) / len(scores)

    def evaluate_device(self, device_data: Dict[str, Any]) -> Dict[str, Any]:
        price = device_data.get("price", 1.0)
        # Теперь ожидаем массив трейтов (или строку для обратной совместимости)
        eval_traits = device_data.get("eval_traits", [])

        reviews = device_data.get("reviews", {})
        n_r = self.eval_reputation(reviews.get("rating", 0.0), reviews.get("count", 0))
        n_s = self.eval_specs(eval_traits, device_data.get("specs", {}))
        e = self.eval_protocol(device_data.get("protocol", []))

        q_total = (self.w_r * n_r) + (self.w_s * n_s) + (self.w_e * e)
        value_score = (q_total * 1000) / price if price > 0 else 0.0

        return {
            "name": device_data.get("name", "Unknown"),
            "price": price,
            "Q_total": round(q_total, 3),
            "N_R": round(n_r, 3),
            "N_S": round(n_s, 3),
            "E": round(e, 3),
            "Value": round(value_score, 2)
        }


if __name__ == "__main__":
    import os

    # Строим правильный путь к общей таксономии в монорепозитории
    # Предполагаем, что скрипт запускается из корня services/catalog_evaluation/
    schema_path = os.path.join("..", "..", "shared", "schemas", "devices", "device_types.json")

    try:
        with open(schema_path, 'r', encoding='utf-8') as f:
            schema = json.load(f)

        # Создаем эвалюатор
        evaluator = QualityEvaluator(traits_schema=schema)

        # Имитируем один спарсенный товар
        mock_device = {
            "name": "Яндекс Станция Миди (Колонка + Хаб)",
            "price": 14990,
            "eval_traits": ["eval_media", "eval_infra"],
            "protocol": ["wifi", "zigbee"],
            "reviews": {"rating": 4.9, "count": 1250},
            "specs": {"max_child_devices": 128}
        }

        # Проверяем
        result = evaluator.evaluate_device(mock_device)
        print(json.dumps(result, indent=2, ensure_ascii=False))

    except FileNotFoundError:
        print(
            f"Файл таксономии не найден по пути: {schema_path}. Убедитесь, что запускаете код из правильной директории.")