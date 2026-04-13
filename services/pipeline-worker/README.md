# Оркестрация конвейера построения каталога

## Обзор

Конвейер построения каталога запускается ежедневно и состоит из последовательности сервисов. Оркестрацией занимается Temporal — он планирует запуск, следит за выполнением каждого шага и выполняет ретраи при ошибках.

Каждый сервис конвейера запускается как Docker-контейнер. Отдельный Temporal worker (`pipeline-worker`) постоянно крутится в фоне и умеет запускать контейнеры по запросу от Temporal.

```
Temporal Scheduler
      │  (cron: каждый день в 11:00)
      ▼
CatalogPipelineWorkflow
      │
      ├─ activity: RunContainer(scraper scrape)
      ├─ activity: RunContainer(scraper parse)
      ├─ activity: RunContainer(extractor run)
      ├─ activity: RunContainer(catalog-builder run)
      └─ activity: RunContainer(quality-calculator run)
```

---

## Pipeline Worker

`pipeline-worker` — это Go-сервис, который:
- подключается к Temporal и регистрирует activity `RunContainer`
- при получении задачи запускает Docker-контейнер с заданными образом, командой и переменными окружения
- ждёт завершения контейнера
- возвращает успех если exit code = 0, иначе возвращает ошибку (Temporal выполнит ретрай)

```go
type RunContainerParams struct {
    Image   string            // "scraper:latest"
    Command []string          // ["scrape", "-c", "config.toml"]
    Env     map[string]string // переменные окружения для контейнера
}
```

---

## Секреты и конфигурация

Каждый сервис читает конфигурацию из `config.toml`, секреты — из переменных окружения. `pipeline-worker` при запуске контейнера передаёт нужные переменные окружения из своего окружения.

Все секреты прописаны как переменные окружения самого `pipeline-worker`, и он пробрасывает нужные в каждый контейнер явно:

```yaml
pipeline-worker:
  environment:
    - DATABASE_PASSWORD=secret
    - YANDEX_CLOUD_API_KEY=key
```

В конфигурации воркера прописана конфигурация для каждой джобы:
- образ docker
- command
- путь к конфигу (передаем внутрь контейнера)
- какие env пробрасывать в контейнер

```toml
[scraper_job]
image = "scraper:latest"
config_path = "scraper_config.toml"
command = ["scrape", "-c", "config.toml"]
env_mapping = { "SCRAPER_DATABASE_PASSWORD" = "DATABASE_PASSWORD" }

[parser_job]
image = "scraper:latest"
config_path = "scraper_config.toml"
command = ["parse", "-c", "config.toml"]
env_mapping = { "SCRAPER_DATABASE_PASSWORD" = "DATABASE_PASSWORD" }

[extractor_job]
image = "extractor:latest"
config_path = "extractor_config.toml"
commands = ["/app/.venv/bin/extractor", "run", "-c", "config.toml"]
env_mapping = { "EXTRACTOR_DATABASE__PASSWORD" = "DATABASE_PASSWORD", "YANDEX_CLOUD_API_KEY" = "YANDEX_CLOUD_API_KEY" }

[[jobs]]
image = "catalog-builder:latest"
config_path = "catalog-builder_config.toml"
command = ["run", "-c", "config.toml"]
env_mapping = { "DATABASE_PASSWORD" = "DATABASE_PASSWORD" }

[[jobs]]
image = "quality-calculator:latest"
config_path = "quality-calculator_config.toml"
command = ["/app/.venv/bin/quality-calculator", "run", "-c", "config.toml"]
env_mapping = { "QUALITY_CALCULATOR__DATABASE_PASSWORD" = "DATABASE_PASSWORD" }
```

`env_mapping` — ключ это имя переменной которая получит джоба, значение это имя переменной из окружения pipeline-worker

---


## Политика запуска и ретраев

**Расписание** задаётся при регистрации Temporal Schedule. В `pipeline-worker/main.go`:

```go
scheduleClient.Create(ctx, client.ScheduleOptions{
    ID: "catalog-pipeline-daily",
    Spec: client.ScheduleSpec{
        CronExpressions: []string{"0 11 * * *"}, // каждый день в 11:00 например ! это тоже вынести в конфиг
    },
    Action: &client.ScheduleWorkflowAction{
        Workflow: CatalogPipelineWorkflow,
    },
})
```

**Ретраи**

```go
// это тоже вынести в конфиг
activityOptions := workflow.ActivityOptions{
    StartToCloseTimeout: 2 * time.Hour,
    RetryPolicy: &temporal.RetryPolicy{
        MaximumAttempts: 3,
        InitialInterval: 1 * time.Minute,
    },
}
```

Если контейнер завершился с ненулевым кодом — activity возвращает ошибку, Temporal ретраит до 3 раз с интервалом 1 минута. Если все попытки исчерпаны — workflow помечается как failed, что видно в Temporal UI.

---

## Мониторинг и наблюдаемость

В существующем `docker-compose.yml` уже есть Jaeger, Loki, Promtail и Grafana. Используем их.

**Temporal UI** (порт 8080) — основной инструмент для наблюдения за workflow: видно историю запусков, статус каждого activity, логи ошибок и ретраев. Для диплома этого достаточно для мониторинга конвейера.

**Логи** — каждый сервис пишет структурированные логи в stdout. Promtail собирает логи всех контейнеров и отправляет в Loki. В Grafana настраивается дашборд с фильтрацией по имени контейнера — можно видеть логи каждого шага конвейера.

**Трейсы** — сервисы поддерживают OpenTelemetry, экспортируют трейсы в Jaeger (уже настроен в docker-compose на порту 4317).

**Метрики** — каждый сервис экспортирует Prometheus-метрики на стандартном порту. Добавить в docker-compose сервис Prometheus:

В Grafana добавить Prometheus как datasource и настроить дашборды для каждого сервиса.
