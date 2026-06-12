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
  | jq . > data/square_room.json
```

Команды выше предполагают, что ты находишься в папке `services/floor-parser`.

## Единицы измерения

Координаты и длины в итоговом `floor.json` остаются в тех единицах, которые указаны в DXF. Эти единицы попадают в поле `meta.units`.


## Тестовые файлы

Для локальной проверки входные DXF и ожидаемые JSON лежат отдельно от кода тестов:

```text
data/
├── square_room.dxf
├── square_room.json
├── apartment_partition_lines.dxf
├── apartment_partition_lines.json
├── apartment_outline_polyline.dxf
├── apartment_outline_polyline.json
├── door_and_window.dxf
├── door_and_window.json
├── floorplan.dxf
└── floorplan.json
```

Запустить тесты можно так из папки `services/floor-parser`:

```bash
python -m unittest discover tests
```
