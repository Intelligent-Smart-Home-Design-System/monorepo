# Smart Home Design System — монорепозиторий

Интеллектуальный планировщик умного дома: по плану квартиры и требованиям
подбирает наборы устройств из реального каталога, собранного парсингом маркетплейсов.

## Архитектура:

Система делится на две независимо разворачиваемые части и общий стек мониторинга.

- **Часть 1 — пайплайн построения каталога.** Скрейпинг, извлечение данных
  (LLM), построение каталога и расчёт качества, оркестрируемые через Temporal.
  Разворачивается через **Docker Compose** (локально) или **Terraform + Docker** (прод).
- **Часть 2 — фронтенд + бэкенд.** Next.js-фронтенд, `frontend-api` (отдаёт
  каталог и планы), `api-gateway` (auth + запуск Temporal-воркфлоу), воркеры и
  nginx как единая точка входа. Разворачивается через **Docker Compose**
  (`docker-compose.apps.yaml` / `docker-compose.apps.prod.yaml` в корне).
  Все сервисы части 2 ходят в **одну БД `smart_home`**.
- **Мониторинг — централизованный стек.** OpenTelemetry Collector принимает
  логи, трейсы и метрики от всех сервисов по OTLP и маршрутизирует в Loki
  (логи), Jaeger (трейсы) и Prometheus (метрики). Grafana — единый UI.
  Разворачивается через `docker-compose.monitoring.yaml`.

Единая точка входа — **nginx на `:8090`**:

| Путь | Куда проксируется |
|------|-------------------|
| `/` и `/_next/*` | фронтенд (Next.js) |
| `/api/v1/auth/*` | `api-gateway` (`/auth/*`) — регистрация / вход / refresh |
| `/api/v1/*` | `frontend-api` — каталог, планы |
| `/start`, `/result`, `/auth/*`, `/healthz` | `api-gateway` |

## Быстрый старт

Для локального запуска нужны только Docker и Docker Compose.

```bash
make help          # все команды

make up            # полный стек: мониторинг + pipeline-worker + app (prod)
make down          # остановить всё

make up-test       # мониторинг + main-pipeline (--profile test, с seed БД)
make down-test     # остановить

make monitoring-up # только мониторинг
make pipeline-up   # только pipeline-worker
make app-up        # только main-pipeline (prod)

make deploy        # git pull + пересобрать prod
```

> **Важно:** стек мониторинга создаёт Docker-сеть `monorepo-monitoring`, к которой
> подключаются сервисы обеих частей. Поэтому `make monitoring-up` нужно запускать
> **до** `make pipeline-up` / `make app-up`. Команды `make up` и `make up-test` делают это
> автоматически в правильном порядке.

Сервисы и порты:

| Сервис | Порт | Описание |
|--------|------|----------|
| nginx | `8090` | Единая точка входа |
| Temporal UI | `8088` | Оркестрация и мониторинг workflow |
| Grafana | `3000` | Дашборды (admin/admin) |
| Jaeger | `16686` | Трассировка |
| Prometheus | `9090` | Метрики |
| OTEL Collector (gRPC) | `4317` | Приём телеметрии (OTLP/gRPC) |
| OTEL Collector (HTTP) | `4318` | Приём телеметрии (OTLP/HTTP) |
| Catalog DB | `5432` | PostgreSQL (catalog/smart_home) |

## Мониторинг и наблюдаемость

Все сервисы отправляют телеметрию в централизованный **OpenTelemetry Collector**
через переменную окружения `OTEL_EXPORTER_OTLP_ENDPOINT`.

```text
┌─────────────┐     OTLP/gRPC      ┌──────────────────┐
│  api-gateway│────────────────────▶│                  │──▶ Loki   (логи)
│  main-pipe  │                     │  OTEL Collector  │──▶ Jaeger (трейсы)
│  pipeline-wk│────────────────────▶│                  │──▶ Prometheus (метрики)
└─────────────┘                     └──────────────────┘
                                            │
                                     ┌──────┴──────┐
                                     │   Grafana   │
                                     │  (единый UI)│
                                     └─────────────┘
```

Конфигурация:
- `otel-collector-config.yaml` — маршрутизация телеметрии
- `observability/loki/loki-config.yaml` — конфигурация Loki
- `observability/prometheus/prometheus.yml` — scrape-конфигурация Prometheus
- `observability/grafana/provisioning/` — автопровизионинг datasources в Grafana

## Структура проекта

```text
.
├── Makefile                      # общий Makefile (make help)
├── README.md
├── docker-compose.monitoring.yaml # централизованный стек мониторинга
├── docker-compose.apps.yaml       # часть 2: dev/test (с локальной БД и seed)
├── docker-compose.apps.prod.yaml  # часть 2: prod (внешняя БД)
├── otel-collector-config.yaml    # конфигурация OTEL Collector
├── observability/                # конфиги Loki, Prometheus, Grafana
│   ├── loki/loki-config.yaml
│   ├── prometheus/prometheus.yml
│   └── grafana/provisioning/datasources/
├── db/
│   └── catalog/migrations/
├── frontend/
│   ├── apps/
│   │   ├── web/
│   │   ├── sim-ui/
│   │   └── apartment-ui/
│   ├── packages/
│   ├── Dockerfile
│   └── docker-compose.yml
├── infra/
│   ├── README.md
│   └── terraform/
├── services/
│   ├── main-pipeline/
│   │   ├── cmd/{main-pipeline,api-gateway}/
│   │   ├── nginx/
│   │   ├── config/
│   │   └── Dockerfile
│   ├── frontend-api/
│   ├── layout/
│   ├── floor-parser/
│   ├── device-selection/
│   ├── pipeline-worker/
│   ├── scraper/
│   ├── extractor/
│   ├── catalog-builder/
│   ├── quality-calculator/
│   └── simulation/
└── shared/
    └── schemas/
```

Для нового Go-сервиса заводится отдельный модуль:

```bash
go mod init github.com/Intelligent-Smart-Home-Design-System/monorepo/services/your-service-name
```

Раскладка Go-сервиса — по [golang-standards/project-layout](https://github.com/golang-standards/project-layout/blob/master/README_ru.md).

## Ветки и PR

- `main` — главная ветка (develop). Сюда мержим готовые версии задач через PR.
- `{task_number}` — ветка под задачу (например, `SH-47`), отводится от `main`.

Название PR: `feat/fix {название сервиса}: {описание изменений}`. В описании
PR прикрепляйте ссылку на задачу в Yougile.
