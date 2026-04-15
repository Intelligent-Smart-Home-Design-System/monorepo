# Extractor service

Сервис обрабатывает распарсенные листинги товаров из серебряного слоя, определяя тип устройства умного дома и извлекая характеристики в соответствии с таксономией, сохраняя результаты в золотой слой.

В пайплайне построения каталога запускается после сервиса parser.

## Usage

Для запуска сервиса и обработки всех снепшотов серебряного слоя:
```sh
uv sync
uv run extractor run [args]
```

Для запуска тестов с метриками:
```sh
uv run extractor run-evaluation [args]
```

Для запуска интеграционных тестов:
```sh
uv run pytest
```

Для запуска примера
```sh
uv run extractor run-sample [args]
```

## Конфигурация

Задается через конфиг `config.toml`
```toml
# путь к таксономии
[taxonomy]
path = "taxonomy_schema.json"

# id облака и LLM модель - aistudio.yandex.ru
[yandex_cloud]
folder = "b1ghdbj0nn88kkalmtlj"
llm_model = "gpt-oss-120b/latest"
api_key = "secret" # задается через envar YANDEX_CLOUD_API_KEY

# база данных каталога
[database]
host = "localhost"
port = 5432
name = "smart_home"
user = "extractor"
password = "secret" # задается через envar EXTRACTOR_DATABASE__PASSWORD

[logging]
level = "INFO"
format = "json"  # or "console"

# Подсказки извлечения для полей - нужны для составления промптов.
[extraction]

# Добавляются к описаниям полей из таксономии
[extraction.hints]
ecosystem = "Only output each ecosystem once"
protocol = "example"

# Строка добавляется в описание поля для каждого возможного значения enum - значение подставляется в шаблон {value}
[extraction.hint_templates]
ecosystem = "Include {value} if and only if the device can connect to the {value} ecosystem."
protocol = "Include {value} if and only if the device explicitly supports the {value} protocol."

```


## Оценка качества

Команда run-evaluation запускает оценку качества извлечения на тестовых данных: набор листингов и идеальных результатов извлечения.

Тестовые данные лежат в [evaluation/listings.json](evaluation/listings.json).

Результат пишется в json файл в evaluation/results - там будут метрики и полученные выходные данные для каждого листинга.

Собираемые метрики:  
* type_accuracy - доля листингов с правильно определенным типом устройства
* perfect_extraction_rate - доля идеально извлеченных листингов, то есть правильно определен тип и все поля верные

* scalar_field_metrics - метрики для каждого скалярного поля 
  * recall - доля верных значений, когда ожидался не null
  * wrong_value_rate - доля неверных значений, когда ожидался не null
  * miss_rate - доля null, когда ожидался не null
  * precision - доля верных значений, когда модель выдала не null
  * hallucination_rate - доля не null значений, когда ожидался null
  * coverage - количество листингов из тестовых данных, где ожидаемое значение не null

* set_field_metrics - метрики для полей-множеств, например множество поддерживаемых экосистем
  * exact_match_rate - доля полного совпадения полученного и ожидаемого множеств
  * value_recall - доля того, что значение есть в полученном множестве, когда оно ожидалось
  * value_hallucination_rate - доля того, что получили значение, когда оно не ожидалось

* type_confusion_matrix - количества полученных типов для каждого ожидаемого типа 
