import json
import os
from typing import Dict, Any, List, Union


class QualityEvaluator:
    def __init__(self, tech_schema: Dict[str, Any], eval_strategy: Dict[str, Any], weights: tuple = (0.3, 0.4, 0.3)):
        self.w_r, self.w_s, self.w_e = weights
        self.bayes_m = 50
        self.bayes_C = 4.2
        self.config = self._build_config(tech_schema, eval_strategy)

    def _build_config(self, tech_schema: Dict[str, Any], eval_strategy: Dict[str, Any]) -> Dict[str, Any]:
        config = {}
        tech_traits = tech_schema.get("traits", {})

        for trait_name, props_strategy in eval_strategy.items():
            if trait_name not in tech_traits:
                continue

            config[trait_name] = {"properties": {}}
            tech_props = tech_traits[trait_name].get("properties", {})

            for prop_name, eval_params in props_strategy.items():
                if prop_name in tech_props:
                    config[trait_name]["properties"][prop_name] = {
                        **tech_props[prop_name],
                        **eval_params
                    }
        return config

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
        if not protocols: return 0.2

        p_lower = set(p.lower() for p in protocols)
        mesh_protocols = {"zigbee", "matter", "thread"}

        score = 0.2
        if p_lower.intersection(mesh_protocols):
            score = 1.0
        elif "wi-fi" in p_lower or "wifi" in p_lower:
            score = 0.7
        elif "bluetooth" in p_lower or "ble" in p_lower:
            score = 0.4

        if score < 1.0 and len(p_lower) > 1:
            score = min(1.0, score + 0.1)

        return score

    def _eval_single_trait(self, trait_key: str, specs: Dict[str, Any]) -> float:
        trait_schema = self.config.get(trait_key)
        if not trait_schema: return 0.5

        properties = trait_schema.get("properties", {})
        score, total_weight = 0.0, 0.0

        for prop_name, rules in properties.items():
            val = specs.get(prop_name)
            weight = rules.get("weight", 0.0)

            if val is not None and "minimum" in rules and "maximum" in rules:
                inverse = rules.get("inverse", False)
                score += self.normalize(val, rules["minimum"], rules["maximum"], inverse) * weight
                total_weight += weight
            elif rules.get("type") == "boolean" and val is True:
                bonus = rules.get("bonus", 0.0)
                score += bonus
        return score / total_weight if total_weight > 0 else 0.5

    def eval_specs(self, trait_keys: Union[str, List[str]], specs: Dict[str, Any]) -> float:
        if isinstance(trait_keys, str):
            trait_keys = [trait_keys]

        if not trait_keys: return 0.5

        scores = []
        for tk in trait_keys:
            # Очищаем префикс eval_, чтобы старые данные парсились корректно
            clean_key = tk.replace("eval_", "") if tk.startswith("eval_") else tk
            scores.append(self._eval_single_trait(clean_key, specs))

        return sum(scores) / len(scores) if scores else 0.5

    def evaluate_device(self, device_data: Dict[str, Any]) -> Dict[str, Any]:
        price = device_data.get("price", 1.0)
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