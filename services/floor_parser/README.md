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
cmd/floor_parser/main.py        <- точка входа
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


```bash
source services/floor_parser/.venv/bin/activate
PYTHONPATH=. python -m services.floor_parser.cmd.floor_parser.main
```

После запуска сервис будет доступен по адресу:

```bash
http://127.0.0.1:8080
```

Проверить, что он поднялся, можно так:

```bash
curl http://127.0.0.1:8080/health
```

## Как отправить файл

Файл можно отправить через `POST /parse`.

Пример:

```bash
curl -X POST http://127.0.0.1:8080/parse \
  -F "file=@services/floor_parser/tests/square_room.dxf"
```

Если хочется сразу сохранить ответ в отдельный JSON-файл:

```bash
curl -X POST http://127.0.0.1:8080/parse \
  -F "file=@services/floor_parser/tests/square_room.dxf" \
  | jq . > services/floor_parser/tests/square_room.json
```

## Тестовые файлы

Для локальной проверки в проекте есть несколько DXF-примеров:

- `square_room.dxf`
- `apartment_partition_lines.dxf`
- `apartment_outline_polyline.dxf`

И соответствующие ожидаемые JSON-ответы:

- `square_room.json`
- `apartment_partition_lines.json`
- `apartment_outline_polyline.json`
