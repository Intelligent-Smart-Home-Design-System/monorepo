from dataclasses import dataclass, field
from typing import Any


@dataclass
class ListingResult:
    listing_id: int
    listing_name: str
    expected_type: str
    actual_type: str
    type_confidence: float
    type_correct: bool
    perfect: bool
    error: str | None
    expected_output: dict[str, Any]
    actual_output: dict[str, Any]


@dataclass
class ModelMetrics:
    model_name: str
    listing_results: list[ListingResult] = field(default_factory=list)

    def compute_summary(self, taxonomy: dict) -> dict:
        n = len(self.listing_results)
        if n == 0:
            return {}

        type_accuracy = sum(r.type_correct for r in self.listing_results) / n
        perfect_extraction_rate = sum(r.perfect for r in self.listing_results) / n

        scalar_metrics = self._compute_scalar_metrics(taxonomy)
        set_metrics = self._compute_set_metrics(taxonomy)
        confusion = self._compute_confusion_matrix()

        return {
            "total_listings": n,
            "type_accuracy": type_accuracy,
            "perfect_extraction_rate": perfect_extraction_rate,
            "scalar_field_metrics": scalar_metrics,
            "set_field_metrics": set_metrics,
            "type_confusion_matrix": confusion,
        }

    def _get_field_type(self, field_name: str, taxonomy: dict) -> str | None:
        """Get field type from taxonomy schema. Returns 'scalar', 'string_enum_set', or 'array'."""
        for type_data in taxonomy.values():
            schema = type_data.get("schema", {})
            props = schema.get("properties", {})
            if field_name in props:
                f = props[field_name]
                if f.get("type") == "array":
                    items = f.get("items", {})
                    if items.get("type") == "string" and "enum" in items:
                        return "string_enum_set"
                    return "array"
                return "scalar"
        return None

    def _get_enum_values(self, field_name: str, taxonomy: dict) -> list[str] | None:
        for type_data in taxonomy.values():
            props = type_data.get("schema", {}).get("properties", {})
            if field_name in props:
                items = props[field_name].get("items", {})
                return items.get("enum")
        return None

    def _compute_scalar_metrics(self, taxonomy: dict) -> dict:
        # collect all scalar fields across all listings
        all_scalar_fields: set[str] = set()
        for r in self.listing_results:
            for field_name in r.expected_output:
                if self._get_field_type(field_name, taxonomy) == "scalar":
                    all_scalar_fields.add(field_name)

        coverage: dict[str, int] = {f: 0 for f in all_scalar_fields}
        correct_value_count: dict[str, int] = {f: 0 for f in all_scalar_fields}
        wrong_value_count: dict[str, int] = {f: 0 for f in all_scalar_fields}
        actual_count: dict[str, int] = {f: 0 for f in all_scalar_fields}
        miss_count: dict[str, int] = {f: 0 for f in all_scalar_fields}
        hallucination_count: dict[str, int] = {f: 0 for f in all_scalar_fields}
        total: dict[str, int] = {f: 0 for f in all_scalar_fields}

        for r in self.listing_results:
            if r.error:
                continue
            for field_name in all_scalar_fields:
                expected = r.expected_output.get(field_name)
                actual = r.actual_output.get(field_name)
                total[field_name] += 1

                if actual is not None:
                    actual_count[field_name] += 1

                if expected is not None:
                    coverage[field_name] += 1
                    if actual is not None:
                        if actual == expected:
                            correct_value_count[field_name] += 1
                        else:
                            wrong_value_count[field_name] += 1
                    else:
                        miss_count[field_name] += 1
                elif actual is not None:
                    hallucination_count[field_name] += 1

        null_expected_count: dict[str, int] = {f: total[f] - coverage[f] for f in all_scalar_fields}

        return {
            "recall": {
                "description": "Rate of correct values when non-null expected",
                "values": {f: correct_value_count[f] / coverage[f] for f in all_scalar_fields if coverage[f]},
            },
            "wrong_value_rate": {
                "description": "Rate of incorrect non-null values when non-null expected",
                "values": {f: wrong_value_count[f] / coverage[f] for f in all_scalar_fields if coverage[f]},
            },
            "miss_rate": {
                "description": "Rate of null outputs when non-null expected",
                "values": {f: miss_count[f] / coverage[f] for f in all_scalar_fields if coverage[f]},
            },
            "precision": {
                "description": "Rate of correct values when non-null output produced",
                "values": {f: correct_value_count[f] / actual_count[f] for f in all_scalar_fields if actual_count[f]},
            },
            "hallucination_rate": {
                "description": "Rate of non-null outputs when null expected",
                "values": {f: hallucination_count[f] / null_expected_count[f] for f in all_scalar_fields if null_expected_count[f]},
            },
            "coverage": {
                "description": "Number of listings where ground truth was non-null",
                "values": {f: coverage[f] for f in all_scalar_fields},
            },
        }

    def _compute_set_metrics(self, taxonomy: dict) -> dict:
        all_set_fields: set[str] = set()
        for r in self.listing_results:
            for field_name in r.expected_output:
                ft = self._get_field_type(field_name, taxonomy)
                if ft in ("string_enum_set", "array"):
                    all_set_fields.add(field_name)

        exact_match: dict[str, list] = {f: [] for f in all_set_fields}
        value_recall: dict[str, list] = {}
        value_halluc: dict[str, list] = {}

        for field_name in all_set_fields:
            enum_values = self._get_enum_values(field_name, taxonomy)
            if enum_values:
                for v in enum_values:
                    value_recall[f"{field_name}_{v}"] = []
                    value_halluc[f"{field_name}_{v}"] = []

        for r in self.listing_results:
            if r.error:
                continue
            for field_name in all_set_fields:
                expected = r.expected_output.get(field_name)
                actual = r.actual_output.get(field_name)

                # skip if both null
                if expected is None and actual is None:
                    continue

                expected_set = set(expected) if expected else set()
                actual_set = set(actual) if actual else set()

                exact_match[field_name].append(1 if expected_set == actual_set else 0)

                enum_values = self._get_enum_values(field_name, taxonomy)
                if not enum_values:
                    continue

                for v in enum_values:
                    key = f"{field_name}_{v}"
                    if v in expected_set:
                        value_recall[key].append(1 if v in actual_set else 0)
                    else:
                        value_halluc[key].append(1 if v in actual_set else 0)

        def avg(lst): return sum(lst) / len(lst) if lst else None

        return {
            "exact_match_rate": {
                "description": "Rate of exact set matches",
                "values": {f: avg(exact_match[f]) for f in all_set_fields if exact_match[f]},
            },
            "value_recall": {
                "description": "Rate of value appearing in actual when expected",
                "values": {k: avg(v) for k, v in value_recall.items() if v},
            },
            "value_hallucination_rate": {
                "description": "Rate of value appearing in actual when not expected",
                "values": {k: avg(v) for k, v in value_halluc.items() if v},
            },
        }

    def _compute_confusion_matrix(self) -> dict[str, dict[str, int]]:
        from collections import defaultdict
        matrix: dict[str, dict[str, int]] = defaultdict(lambda: defaultdict(int))
        for r in self.listing_results:
            matrix[r.expected_type][r.actual_type] += 1
        return {k: dict(v) for k, v in matrix.items()}
