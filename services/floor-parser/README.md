# floor-parser

Это микросервис `floor-parser` для парсинга планов квартир в формат `floor.json`.

На вход приходит файл плана, дальше он проходит через несколько этапов обработки, и на выходе получается нормальный JSON, с которым уже можно работать в других сервисах.

## Что делает сервис

`floor-parser` имеет следующий сценарий работы:

1. принять файл
2. прочитать его как чертеж
3. достать из него геометрию
4. привести данные к более удобному виду
5. определить, что именно обозначают сущности
6. собрать итоговую модель плана
7. вернуть результат в `json`

## Структура проекта

Архитектура сейчас такая:

```text
app/floor_parser/main.py        <- точка входа
internal/
├── api/                  <- HTTP API
├── classification/       <- определение смысла сущностей
├── entities/             <- внутренние модели данных
├── export/               <- сборка итогового floor.json
├── normalization/        <- приведение геометрии к удобному виду
├── readers/              <- чтение исходных форматов
│   └── dxf/              <- чтение и извлечение сущностей из DXF
├── topology/             <- сборка итоговой модели плана
└── pipeline.py           <- сценарий парсинга
```

## Как запустить

Версия Python для сервиса указана в `.python-version`: `3.12.3`.

Сначала нужно создать виртуальное окружение и поставить зависимости. `.venv` в репозиторий не коммитится, поэтому после клонирования его нужно создать локально:

```bash
cd services/floor-parser
python -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
```

После установки сервис можно запустить так:

```bash
python -m app.floor_parser.main
```

По умолчанию сервис слушает порт `8080`:

```bash
http://127.0.0.1:8080
```

Порт можно изменить через переменную окружения `PARSER_PORT`:

```bash
PARSER_PORT=8090 python -m app.floor_parser.main
```

Проверить, что он поднялся, можно так:

```bash
curl http://127.0.0.1:8080/health
```

## Логирование

Сервис использует `structlog` для структурированных JSON-логов. Формат и уровень логирования настраиваются через переменные окружения:

| Переменная | Описание | Значение по умолчанию |
|------------|----------|-----------------------|
| `LOG_FORMAT` | Формат вывода логов: `json` или `console` | `json` |
| `LOG_LEVEL` | Уровень логирования: `DEBUG`, `INFO`, `WARNING`, `ERROR`, `CRITICAL` | `INFO` |

Пример запуска с настройками по умолчанию (JSON, INFO):

```bash
python -m app.floor_parser.temporal_worker
```

Для разработки с читаемым выводом в консоль:

```bash
LOG_FORMAT=console LOG_LEVEL=DEBUG python -m app.floor_parser.temporal_worker
```

В JSON-режиме каждый лог — это JSON-строка с полями `event`, `level`, `timestamp`, `logger` и дополнительными контекстными полями (`request_id`, `filename` и т.д.). Это позволяет парсить логи в системах агрегации (ELK, Loki, Cloud Logging).

Корреляция запросов осуществляется через заголовок `X-Request-ID` в HTTP-запросах. Если заголовок не передан, генерируется UUID. Значение `request_id` автоматически добавляется во все логи, связанные с этим запросом, включая вызовы Temporal-активностей.

## Как отправить файл

Файл можно отправить через `POST /parse`.

Пример:

```bash
curl -X POST http://127.0.0.1:8080/parse \
  -F "file=@data/square_room.dxf"
```

Если хочется сразу сохранить ответ в отдельный JSON-файл:

```bash
curl -X POST http://127.0.0.1:8080/parse \
  -F "file=@data/square_room.dxf" \
  | jq . > data/square_room.expected.json
```

Команды выше предполагают, что ты находишься в папке `services/floor-parser`.

## Единицы измерения

Координаты и длины в итоговом `floor.json` при экспорте переводятся в миллиметры, если исходные единицы DXF распознаны. В этом случае в `meta.units` будет значение `mm`.

Площадь комнаты остаётся в `area_m2`, то есть в квадратных метрах.

## Пример результата

```json
{
  "schema_version": "0.1.0",
  "meta": {
    "source": "dxf",
    "source_ref": "example.dxf",
    "units": "mm"
  },
  "doors": [
    {
      "id": "door_1",
      "points": [[760.0, 2338.5], [1660.0, 2338.5]],
      "width": 900.0,
      "rooms": ["room_2", "room_4"],
      "opens_towards_room": "room_4",
      "swing": "single_swing",
      "hinge_side": "end"
    }
  ]
}
```

Поле `hinge_side` показывает, на каком конце отрезка двери находится петлевая сторона:
- `start` означает, что петли находятся в первой точке из `points`
- `end` означает, что петли находятся во второй точке из `points`


## Тестовые файлы

Для локальной проверки входные DXF и ожидаемые JSON лежат отдельно от кода тестов.
Соглашение по именам такое:
- входные файлы: `*.dxf`
- ожидаемые ответы парсера: `*.expected.json`

```text
data/
├── square_room.dxf
├── square_room.expected.json
├── apartment_partition_lines.dxf
├── apartment_partition_lines.expected.json
├── room_with_door_and_window.dxf
├── room_with_door_and_window.expected.json
├── apartment_first_floor_insert_blocks.dxf
├── apartment_first_floor_insert_blocks.expected.json
├── us_house_plan.dxf
├── us_house_plan.expected.json
├── two_bedroom_ensuite_apartment.dxf
└── two_bedroom_ensuite_apartment.expected.json
```

Запустить тесты можно так из папки `services/floor-parser`:

```bash
python -m unittest discover tests
```
