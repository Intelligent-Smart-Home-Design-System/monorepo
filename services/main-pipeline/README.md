# main-pipeline

Temporal workflow-orchestrator: `floor-parser` → `layout` → `device-selection`.

## Команды

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

## Ссылки

| Сервис | URL | Логин |
|--------|-----|-------|
| Приложение (nginx) | http://localhost:8090 | — |
| Temporal UI | http://localhost:8088 | — |
| Grafana | http://localhost:3000 | `admin` / `admin` |
| Jaeger | http://localhost:16686 | — |
| Prometheus | http://localhost:9090 | — |

## Переменные окружения

### main-pipeline (`docker-compose.apps.yaml` / `docker-compose.apps.prod.yaml`)

| Переменная | Дефолт | Где задан дефолт | Описание | Менять для прода? |
|------------|--------|------------------|----------|:-:|
| `CATALOG_DB_USER` | `catalog` | `docker-compose.apps.yaml` | Пользователь PostgreSQL | ✅ |
| `CATALOG_DB_PASSWORD` | `catalog` | `docker-compose.apps.yaml` | Пароль PostgreSQL | ✅ |
| `CATALOG_DB_NAME` | `smart_home` | `docker-compose.apps.yaml` | Имя базы данных | — |
| `CATALOG_DB_HOST` | `catalog-postgresql` | `docker-compose.apps.prod.yaml` | Хост БД (прод — внешний) | ✅ |
| `CATALOG_DB_PORT` | `5432` | `docker-compose.apps.prod.yaml` | Порт БД | — |
| `CATALOG_DB_SSLMODE` | `disable` | `docker-compose.apps.prod.yaml` | SSL-режим PostgreSQL | ✅ `require` |
| `JWT_SECRET` | `dev-jwt-secret` | `docker-compose.apps.yaml` | Секрет для JWT-токенов | ✅ |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `otel-collector:4318` | `docker-compose.apps.yaml` | Адрес OTEL Collector | — |
| `TEMPORAL_DB_USER` | `temporal` | `docker-compose.apps.yaml` | Пользователь Temporal DB | — |
| `TEMPORAL_DB_PASSWORD` | `temporal` | `docker-compose.apps.yaml` | Пароль Temporal DB | ✅ |

### pipeline-worker (`services/pipeline-worker/docker-compose.yaml`)

| Переменная | Дефолт | Где задан дефолт | Описание | Менять для прода? |
|------------|--------|------------------|----------|:-:|
| `CATALOG_DATABASE_PASSWORD` | `catalog` | `docker-compose.yaml` | Пароль PostgreSQL | ✅ |
| `YANDEX_CLOUD_API_KEY` | `dummy-yandex-api-key` | `docker-compose.yaml` | API-ключ Yandex Cloud | ✅ |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `otel-collector:4317` | `docker-compose.yaml` | Адрес OTEL Collector | — |
| `LOG_LEVEL` | `info` | `docker-compose.yaml` | Уровень логирования | — |

### Мониторинг (`docker-compose.monitoring.yaml`)

| Переменная | Дефолт | Где задан дефолт | Описание | Менять для прода? |
|------------|--------|------------------|----------|:-:|
| `DEPLOYMENT_ENVIRONMENT` | `local` | `docker-compose.monitoring.yaml` | Лейбл среды в телеметрии | ✅ `production` |
| `GF_SECURITY_ADMIN_PASSWORD` | `admin` | `docker-compose.monitoring.yaml` | Пароль Grafana | ✅ |

### Как задавать

- **Локально**: дефолты работают из коробки, ничего менять не нужно.
- **Тестинг/прод**: создать `.env` в корне монорепозитория (см. `services/main-pipeline/.env.example`) или задать через `export`.

## Подключение OTLP к новому Go-сервису

В `main()`:

```go
telemetry := otelsetup.New(ctx, "service-name")
defer telemetry.Shutdown()
log := telemetry.Log
```

Пакеты: `shared/telemetry/go/{otelsetup,otellog,oteltrace,otelzerolog}`.

В `docker-compose.apps.yaml`:

```yaml
environment:
  OTEL_EXPORTER_OTLP_ENDPOINT: ${OTEL_EXPORTER_OTLP_ENDPOINT:-otel-collector:4318}
  OTEL_EXPORTER_OTLP_INSECURE: "true"
networks:
  - default
  - monorepo-monitoring
```

Если `OTEL_EXPORTER_OTLP_ENDPOINT` пуст — writer работает как no-op.

## Тестовые запросы

```bash
# Регистрация
curl -X POST http://localhost:8090/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@example.com","password":"demo-password"}'

# Логин → скопировать access_token
curl -X POST http://localhost:8090/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@example.com","password":"demo-password"}'

# Запуск workflow
curl -X POST http://localhost:8090/start \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <access_token>" \
  --data-binary @examples/security_basic.json

# Результат
curl http://localhost:8090/result/<workflow-id> \
  -H "Authorization: Bearer <access_token>"

# API каталога
curl http://localhost:8090/api/v1/device-types
curl http://localhost:8090/api/v1/ecosystems
curl http://localhost:8090/api/v1/presets
curl http://localhost:8090/api/v1/plans
```
