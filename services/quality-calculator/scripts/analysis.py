import re
from typing import List, Dict

class ReviewMinerNLP:
    """
    Прототип лексико-ориентированного анализатора отзывов (по мотивам Pan et al., 2020).
    Извлекает аппаратные и программные причины сбоев, а также метрику MTBF.
    """

    def __init__(self):
        # Словари для классификации инцидентов
        self.lex_network = [
            "отваливается", "теряет связь", "не подключается",
            "постоянно офлайн", "отпадает от шлюза", "сброс сети", "нет пинга"
        ]
        self.lex_hardware = [
            "сгорел", "мерцает", "свистит", "перегрев",
            "пищит", "заклинило", "короткое замыкание", "треск"
        ]
        # Регулярные выражения для выявления MTBF
        self.regex_ttf = [
            r"через\s+(\d+)\s*(день|дня|дней)",
            r"спустя\s+(\d+)\s*(недел[юиь])",
            r"хватило\s+на\s+(\d+)\s*(месяц[аев]*)"
        ]

    def extract_ttf(self, text: str) -> int:
        """
        Извлекает время до наступления сбоя (Time-to-Failure) в днях.
        Возвращает -1, если временной маркер не найден.
        """
        text_lower = text.lower()
        for pattern in self.regex_ttf:
            match = re.search(pattern, text_lower)
            if match:
                value = int(match.group(1))
                unit = match.group(2)
                if "день" in unit or "дня" in unit or "дней" in unit:
                    return value
                elif "недел" in unit:
                    return value * 7
                elif "месяц" in unit:
                    return value * 30
        return -1

    def analyze_device_reviews(self, reviews: List[Dict[str, str]]) -> Dict[str, float]:
        """
        Анализирует пакет отзывов и генерирует понижающие коэффициенты
        (penalties) для интеграции в матрицу оценки качества Q_total.
        """
        total_reviews = len(reviews)
        if total_reviews == 0:
            return {"penalty_E": 1.0, "penalty_NS": 1.0, "estimated_mtbf_days": 0.0}

        net_fails, hw_fails = 0, 0
        ttf_records = []

        for rev in reviews:
            text = rev.get("text", "").lower()

            # Проверка пересечений с лексиконами
            if any(term in text for term in self.lex_network):
                net_fails += 1
            if any(term in text for term in self.lex_hardware):
                hw_fails += 1

            # Попытка извлечения MTBF
            ttf = self.extract_ttf(text)
            if ttf > 0:
                ttf_records.append(ttf)

        # Вычисление доли критических упоминаний
        net_fail_rate = net_fails / total_reviews
        hw_fail_rate = hw_fails / total_reviews

        # Формирование штрафа. Защита от избыточного обнуления (мин. 0.5)
        # Если 25% отзывов говорят об отвале сети, штраф будет 1.0 - (0.25 * 2) = 0.5
        penalty_e = max(0.5, 1.0 - (net_fail_rate * 2))
        penalty_ns = max(0.5, 1.0 - (hw_fail_rate * 2))

        avg_mtbf = sum(ttf_records) / len(ttf_records) if ttf_records else 0.0

        return {
            "penalty_E": round(penalty_e, 2),
            "penalty_NS": round(penalty_ns, 2),
            "estimated_mtbf_days": round(avg_mtbf, 1)
        }