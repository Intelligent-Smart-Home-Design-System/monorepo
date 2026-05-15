# main-pipeline

`main-pipeline` - это orchestration-сервис на `Temporal`, который запускает и связывает между собой три независимых шага:

1. `floor-parser` - парсинг плана квартиры.
2. `layout` - построение раскладки устройств по плану.
3. `device-selection` - подбор устройств под входной запрос.

Сам `main-pipeline` не выполняет бизнес-логику этих шагов. Его задача - принять запрос, запустить workflow в Temporal и последовательно отправить activity в нужные очереди.

## Что входит в сервис

- `cmd/main-pipeline-worker` - воркер-оркестратор, который регистрирует workflow и слушает очередь `main-pipeline-orchestration`.
- `cmd/main-pipeline-trigger` - утилита для ручного старта workflow из JSON-файла.
- `workflows/main_pipeline.go` - сценарий выполнения pipeline.
- `config/pipeline.toml` - конфигурация Temporal, очередей и retry-политики.
- `docker-compose.yml` - локальное окружение с Temporal, воркерами зависимых сервисов и observability-стеком.

## Как работает workflow

Workflow запускает шаги строго по порядку:

1. `floor_parser.parse_floor`
2. `layout.build_layout`
3. `device_selection.select_devices_from_file`

Шаг запускается только если соответствующий блок присутствует в JSON-запросе. Если блок отсутствует, шаг просто пропускается.

Важно: `main-pipeline` не подставляет результаты одного шага в другой автоматически. Связь между шагами задается путями в самом запросе. Например, если `layout` должен читать результат `floor-parser`, это нужно явно указать через одинаковый путь:

```json
"floor_parser": {
  "output_path": "/data/artifacts/floor.json"
},
"layout": {
  "apartment_path": "/data/artifacts/floor.json"
}
```

## Конфигурация

Основной конфиг лежит в `config/pipeline.toml`.

### Секции конфига

- `[temporal]` - адрес Temporal, namespace, orchestration queue, число попыток подключения и префикс `workflow_id`.
- `[metrics]` - адрес HTTP-сервера с `GET /metrics` и `GET /healthz`.
- `[queues]` - очереди, в которые отправляются activity зависимых сервисов.
- `[activity]` - timeout'ы и retry-политика для всех activity.
- `[trigger]` - путь к JSON-запросу по умолчанию для `main-pipeline-trigger`.

### Переменные окружения

- `PIPELINE_CONFIG` - путь к TOML-конфигу. По умолчанию: `config/pipeline.toml`.
- `PIPELINE_REQUEST` - путь к JSON-запросу для `main-pipeline-trigger`.
- `WAIT_FOR_COMPLETION` - ждать завершения workflow или только стартовать его. По умолчанию: `true`.
- `LOG_LEVEL` - уровень логирования `zerolog`.
- `TRACING_ENABLED` - включение OpenTelemetry tracing.
- `OTEL_EXPORTER_OTLP_ENDPOINT` - OTLP endpoint для отправки трейсов.
- `OTEL_EXPORTER_OTLP_INSECURE` - insecure-режим для OTLP.
- `SERVICE_VERSION`, `APP_ENV` - служебные атрибуты трассировки.

## Формат входного запроса

`main-pipeline-trigger` читает JSON и преобразует его в вход workflow.

Пример:

```json
{
  "request_id": "demo-main-pipeline",
  "floor_parser": {
    "source_path": "/workspace/path/to/apartment.dxf",
    "output_path": "/data/artifacts/floor.json"
  },
  "layout": {
    "apartment_path": "/data/artifacts/floor.json",
    "output_path": "/data/artifacts/layout.json",
    "selected_levels": {
      "lighting": "base",
      "security": "standard"
    }
  },
  "device_selection": {
    "request_path": "/workspace/path/to/device-selection-request.json",
    "output_path": "/data/artifacts/device-selection.json"
  },
  "metadata": {
    "initiator": "manual-run"
  }
}
```

### Что важно про пути

При запуске через `docker compose` все сервисы видят:

- репозиторий как `/workspace`
- общую папку артефактов как `/data/artifacts`

Из-за этого:

- входные файлы из репозитория удобно передавать как `/workspace/...`
- промежуточные и итоговые результаты удобно писать в `/data/artifacts/...`

Если папки `artifacts` рядом с сервисом еще нет, ее нужно создать перед запуском `docker compose`, иначе смонтировать ее будет неудобно.

## Локальный запуск через Docker Compose

Из директории `services/main-pipeline`:

```bash
mkdir artifacts
docker compose up --build -d temporal temporal-ui jaeger loki promtail prometheus grafana main-pipeline-worker floor-parser-worker layout-worker device-selection-worker
```

После этого можно вручную стартовать workflow через trigger:

```bash
docker compose --profile tools run --rm -e PIPELINE_REQUEST=/data/artifacts/request.json main-pipeline-trigger
```

Ожидается, что файл `services/main-pipeline/artifacts/request.json` уже существует на хосте.

## Локальный запуск без Docker Compose

Если Temporal и зависимые воркеры уже подняты отдельно, сервис можно запускать напрямую:

```bash
go run ./cmd/main-pipeline-worker
```

Старт workflow:

```powershell
$env:PIPELINE_REQUEST="artifacts/request.json"
go run ./cmd/main-pipeline-trigger
```

В этом режиме особенно важно, чтобы:

- `config/pipeline.toml` указывал на доступный Temporal instance
- очереди в `[queues]` совпадали с очередями реально запущенных воркеров
- пути в JSON-запросе были доступны процессам, которые исполняют activity

## Как взаимодействовать с сервисом

Обычный сценарий такой:

1. Поднять Temporal и все зависимые воркеры.
2. Подготовить JSON-запрос для `main-pipeline-trigger`.
3. Убедиться, что пути к входным и выходным файлам корректны для всех контейнеров или процессов.
4. Запустить `main-pipeline-trigger`.
5. Проверить статус workflow в Temporal UI и итоговые артефакты на диске.

## Observability

В локальном `docker-compose` уже подготовлены:

- Temporal UI: `http://localhost:8080`
- Grafana: `http://localhost:3000`
- Prometheus: `http://localhost:9090`
- Jaeger UI: `http://localhost:16686`
- Loki: `http://localhost:3100`

Метрики самого `main-pipeline-worker` по умолчанию доступны на `http://localhost:2112/metrics`, healthcheck - на `http://localhost:2112/healthz`.

## Полезные замечания

- Workflow ID формируется как `<workflow_id_prefix>-<UTC timestamp>`.
- Если `request_id` не задан, будет использовано значение `main-pipeline-request`.
- При старте trigger по умолчанию ждет завершения workflow и выводит итоговый результат в логи.
- Ошибка любого activity завершает workflow с ошибкой.
- Повторные попытки activity управляются секцией `[activity]` в `pipeline.toml`.
