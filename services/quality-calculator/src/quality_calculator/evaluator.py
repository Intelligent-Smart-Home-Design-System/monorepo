from __future__ import annotations

from typing import Any


# Сетевая топология протоколов. Совпадение определяется по подстроке,
# т.к. в каталоге встречаются составные значения вида "matter-over-thread".
MESH_MARKERS = ("zigbee", "thread", "matter", "z-wave", "zwave")
STAR_MARKERS = ("wifi", "wi-fi")
P2P_MARKERS = ("bluetooth", "ble")


class QualityEvaluator:
    """
    Математическое ядро оценки качества IoT-устройств (Data-Driven).

    Параметры конфигурации не хардкодятся: правила нормализации тянутся из
    канонической таксономии (device_types.json -> traits/types), а веса
    характеристик — из выбранной стратегии (config/strategies + evaluation_traits.json).

    Поддерживаемые типы правил оценки характеристики:
      * числовое:        {"weight": w, "inverse": bool?}    -> нормализация по [minimum, maximum] из таксономии
      * булев бонус:     {"bonus": b}                        -> +b к оценке трейта, если значение True
      * порядковая шкала:{"weight": w, "scale": {val: 0..1}} -> маппинг строкового значения в [0..1]
      * мощность массива:{"weight": w, "count_max": n}       -> len(value)/n (версатильность)
    """

    def __init__(
        self,
        tech_schema: dict[str, Any],
        eval_strategy: dict[str, Any],
        weights: tuple[float, float, float] = (0.3, 0.4, 0.3),
        reputation_mode: str = "bayesian",
        bayes_m: int = 50,
        bayes_c: float = 4.2,
        rating_floor: float = 3.5,
        rating_ceil: float = 5.0,
    ):
        self.w_r, self.w_s, self.w_e = weights
        self.reputation_mode = reputation_mode
        self.bayes_m = bayes_m
        self.bayes_c = bayes_c
        self.rating_floor = rating_floor
        self.rating_ceil = rating_ceil

        self.traits_schema: dict[str, Any] = tech_schema.get("traits", {})
        self.types_schema: dict[str, Any] = tech_schema.get("types", {})
        self.config = self._build_config(eval_strategy)

    def _build_config(self, eval_strategy: dict[str, Any]) -> dict[str, Any]:
        """Слияние правил оценки со схемой таксономии (границы minimum/maximum, тип поля)."""
        config: dict[str, Any] = {}
        for trait_name, props_strategy in eval_strategy.items():
            if trait_name not in self.traits_schema:
                continue
            tech_props = self.traits_schema[trait_name].get("properties", {})
            merged: dict[str, Any] = {"properties": {}}
            for prop_name, eval_params in props_strategy.items():
                base = tech_props.get(prop_name, {})
                merged["properties"][prop_name] = {**base, **eval_params}
            config[trait_name] = merged
        return config

    def traits_for_type(self, type_name: str) -> list[str]:
        """Трейты, из которых состоит тип устройства (по таксономии)."""
        return self.types_schema.get(type_name, {}).get("traits", [])

    # ---------- нормализация ----------

    @staticmethod
    def normalize(value: float, min_val: float, max_val: float, inverse: bool = False) -> float | None:
        if not isinstance(value, (int, float)) or isinstance(value, bool):
            return None
        if max_val is None or min_val is None:
            return None
        if max_val <= min_val:
            return 1.0 if value >= max_val else 0.0
        clipped = max(min_val, min(value, max_val))
        norm = (clipped - min_val) / (max_val - min_val)
        return 1.0 - norm if inverse else norm

    # ---------- N(R): репутация ----------

    def eval_reputation(self, rating: float | None, count: int) -> float | None:
        if rating is None or count <= 0:
            return None
        if self.reputation_mode == "mean":
            effective = rating
        else:  # bayesian
            effective = (count * rating + self.bayes_m * self.bayes_c) / (count + self.bayes_m)
        norm = self.normalize(effective, self.rating_floor, self.rating_ceil)
        return norm

    # ---------- E: протокол / топология ----------

    def eval_protocol(self, protocols: list[str] | None) -> float | None:
        if not protocols:
            return None
        low = [str(p).lower() for p in protocols]

        def has(markers):
            return any(any(m in p for m in markers) for p in low)

        if has(MESH_MARKERS):
            score = 1.0
        elif has(STAR_MARKERS):
            score = 0.7
        elif has(P2P_MARKERS):
            score = 0.4
        else:
            score = 0.2

        # Синергия: мультипротокольное устройство может работать мостом/шлюзом.
        if score < 1.0 and len(set(low)) > 1:
            score = min(1.0, score + 0.1)
        return score

    # ---------- N(S): характеристики ----------

    def _score_trait(self, trait_key: str, specs: dict[str, Any]) -> float | None:
        """Оценка одного трейта. None означает 'нет данных для оценки этого трейта'."""
        trait_cfg = self.config.get(trait_key)
        if not trait_cfg:
            return None

        score = 0.0
        total_weight = 0.0
        got_signal = False

        for prop_name, rules in trait_cfg["properties"].items():
            val = specs.get(prop_name)
            if val is None:
                continue

            # булев бонус
            if "bonus" in rules and "weight" not in rules:
                if val is True:
                    score += rules["bonus"]
                    got_signal = True
                continue

            weight = rules.get("weight", 0.0)
            contribution: float | None = None

            if "scale" in rules:  # порядковая шкала строк
                contribution = rules["scale"].get(val)
            elif "count_max" in rules and isinstance(val, list):  # мощность массива
                contribution = min(1.0, len(val) / rules["count_max"]) if rules["count_max"] else None
            elif "minimum" in rules and "maximum" in rules:  # числовое
                contribution = self.normalize(val, rules["minimum"], rules["maximum"], rules.get("inverse", False))

            if contribution is not None:
                score += contribution * weight
                total_weight += weight
                got_signal = True

        if not got_signal:
            return None
        if total_weight > 0:
            return score / total_weight  # бонусы прибавляются поверх взвешенного среднего
        return score  # трейт состоит только из бонусов

    def eval_specs(self, trait_keys: list[str], specs: dict[str, Any]) -> float | None:
        """Среднее по трейтам устройства, у которых есть данные. None — спеков нет вовсе."""
        scores = [s for tk in trait_keys if (s := self._score_trait(tk, specs)) is not None]
        if not scores:
            return None
        return sum(scores) / len(scores)

    # ---------- агрегированная оценка ----------

    def evaluate_device(self, device_data: dict[str, Any]) -> dict[str, Any]:
        """
        device_data:
          category: str            (тип из таксономии; трейты резолвятся автоматически)
          eval_traits: list[str]   (опционально; переопределяет резолв по category)
          specs: dict              (device_attributes)
          reviews: {rating, count} (агрегировано по листингам)
          protocol: list[str]
          price: number
        """
        category = device_data.get("category")
        trait_keys = device_data.get("eval_traits") or self.traits_for_type(category)

        reviews = device_data.get("reviews", {})
        n_r = self.eval_reputation(reviews.get("rating"), reviews.get("count", 0))
        n_s = self.eval_specs(trait_keys, device_data.get("specs", {}))
        e = self.eval_protocol(device_data.get("protocol", []))

        # Взвешенная сумма только по доступным компонентам, с перенормировкой весов.
        components = [(n_r, self.w_r), (n_s, self.w_s), (e, self.w_e)]
        active = [(v, w) for v, w in components if v is not None and w > 0]
        wsum = sum(w for _, w in active)
        q_total = sum(v * w for v, w in active) / wsum if wsum > 0 else None

        price = device_data.get("price")
        value_score = (q_total * 1000) / price if (q_total is not None and price) else None

        return {
            "id": device_data.get("id"),
            "name": device_data.get("name", "Unknown"),
            "category": category,
            "price": price,
            "Q_total": round(q_total, 4) if q_total is not None else None,
            "N_R": round(n_r, 4) if n_r is not None else None,
            "N_S": round(n_s, 4) if n_s is not None else None,
            "E": round(e, 4) if e is not None else None,
            "Value": round(value_score, 3) if value_score is not None else None,
        }
