# Scraper & Pipeline — гайд для разработчика с `main`

Ветка **`renovate/scraper-pipeline-on-main`** — аккуратная интеграция рефакторинга скрейпера и пайплайна поверх привычного `main`.  
Документ для онбординга: что изменилось, где искать код и какие команды запускать.

> **Базовая ветка:** `main`  
> **Источник изменений:** `feature/scraper-pipeline-refactor`  
> **Renovate-ветка:** `renovate/scraper-pipeline-on-main`

---

## Содержание

1. [Краткое резюме](#краткое-резюме)
2. [Было → Стало](#было--стало)
3. [Корневой Makefile и Docker Compose](#1-корневой-makefile-и-docker-compose)
4. [Scraper: архитектура sources + pipeline](#2-scraper-архитектура-sources--pipeline)
5. [DNS: новый источник](#3-dns-новый-источник)
6. [Job-фильтры в TOML](#4-job-фильтры-в-toml)
7. [Pipeline-worker: 4 job'а scraper](#5-pipeline-worker-4-jobа-scraper)
8. [Observability](#6-observability)
9. [Полезные команды](#полезные-команды)
10. [Карта ключевых файлов](#карта-ключевых-файлов)

---

## Краткое резюме

На `main` скрейпер — монолитный CLI: в `scrape.go` вручную создавались scraper'ы WB/Yandex/Printer, discovery и listing шли одним потоком.

В renovate-ветке:

- **Корневой `Makefile`** — единая точка для мониторинга, pipeline и app.
- **`docker-compose.pipeline.yaml`** перенесён в корень репозитория (раньше — `services/pipeline-worker/docker-compose.yaml`).
- **Scraper** разбит на слои: `sources` (контракт маркетплейса) → `pipeline` (оркестрация шагов) → `scrapers/*` + `parsers/*` (реализация).
- **DNS** — полноценный источник: HTTP + Chrome (Qrator), BFS по каталогу, парсинг category/listing.
- **Job-фильтры** `[jobs.*.dns]` — лимиты, bootstrap-режим, фильтры по URL/датам для каждого шага Temporal-пайплайна.

---

## Было → Стало

| Область | На `main` | В renovate-ветке |
|--------|-----------|------------------|
| Запуск pipeline | `make pipeline-up` → compose внутри `services/pipeline-worker/` | `make pipeline-stack-up` / `make pipeline-up` → [`docker-compose.pipeline.yaml`](../../docker-compose.pipeline.yaml) в корне |
| Сборка образов | только `docker compose build` | [`make pipeline-build`](../../Makefile) — явная сборка scraper, extractor, worker, trigger |
| CLI scrape | один файл, ручная инициализация scraper'ов | [`DiscoveryPipeline`](../../services/scraper/internal/pipeline/discovery.go) / [`ListingPipeline`](../../services/scraper/internal/pipeline/listing.go) + [`sources.Registry`](../../services/scraper/internal/sources/registry.go) |
| Новый маркетплейс | правки в `scrape.go` / `parse.go` | реализовать [`sources.Source`](../../services/scraper/internal/sources/source.go) + `scrapers/<name>` + `parsers/<name>`; шаблон — [`parsers/example/doc.go`](../../services/scraper/internal/parsers/example/doc.go) |
| DNS | отсутствует | [`scrapers/dns`](../../services/scraper/internal/scrapers/dns/), [`parsers/dns`](../../services/scraper/internal/parsers/dns/) |
| Фильтрация задач | только CLI-флаги `--sources`, `--page-types` | дополнительно `[jobs.scrape_discovery.dns]` и др. в TOML — [`job_filter.go`](../../services/scraper/internal/config/job_filter.go) |
| Catalog pipeline | 2 job'а scraper (scrape + parse) | 4 job'а: discovery scrape → discovery parse → listing scrape → listing parse — [`pipeline.toml`](../../services/pipeline-worker/config/pipeline.toml) |
| Метрики scraper | минимальные | [`internal/metrics`](../../services/scraper/internal/metrics/metrics.go) + OTLP через shared telemetry |

---

## 1. Корневой Makefile и Docker Compose

### Было (`main`)

```makefile
COMPOSE_PIPELINE := docker compose -f services/pipeline-worker/docker-compose.yaml

pipeline-up: ## Поднять pipeline-worker
	$(COMPOSE_PIPELINE) up -d --build
```

Только `pipeline-up` / `pipeline-down`. Сборка образов — неявно через compose.

### Стало (renovate)

```makefile
COMPOSE_PIPELINE := docker compose -p catalog-pipeline -f docker-compose.pipeline.yaml

pipeline-build: ## Собрать образы pipeline (scraper, extractor, worker, …)
	docker build -f services/scraper/Dockerfile -t scraper:latest .
	# … extractor, catalog-builder, quality-calculator, pipeline-worker, pipeline-trigger

pipeline-stack-up: monitoring-up pipeline-build pipeline-up ## Отдельная машина: мониторинг + pipeline
	$(MAKE) pipeline-migrate

pipeline-trigger: ## Запустить catalog pipeline workflow
	$(COMPOSE_PIPELINE) --profile tools run --rm --no-deps pipeline-trigger
```

Файл: [`Makefile`](../../Makefile)

**Зачем:** один `make help` из корня; явная сборка образов перед прогоном; project name `catalog-pipeline` (избегает конфликтов со старым compose); команды migrate/seed/trigger/logs.

Compose перенесён в корень — [`docker-compose.pipeline.yaml`](../../docker-compose.pipeline.yaml):

```yaml
# Usage (from monorepo root):
#   make pipeline-stack-up
#   make pipeline-trigger
name: catalog-pipeline
```

> **Важно:** pipeline подключается к внешней сети `monorepo-monitoring`. Сначала `make monitoring-up`, потом pipeline (или `make pipeline-stack-up` — всё сразу).

---

## 2. Scraper: архитектура sources + pipeline

### Было (`main`)

В [`scrape.go`](../../services/scraper/internal/cli/scrape.go) на main — прямое создание scraper'ов:

```go
printerScraper := printer.NewPrinterScraper()
wildberriesScraper := wbScraper.NewScraper(...)
yandexScraper := yandexScraper.NewScraper(...)
// … цикл по taskRepo.GetTasks, worker pool
```

Каждый новый источник = правки в CLI.

### Стало (renovate)

**Контракт источника** — [`internal/sources/source.go`](../../services/scraper/internal/sources/source.go):

```go
// Source is the only contract pipeline/cli use per marketplace.
//
// Discovery always runs the same three steps:
//   A BootstrapDiscovery — tasks from config
//   B ExpandDiscovery    — in-memory catalog walk (DNS: BFS)
//   C DiscoveryScrapeTypes — page types to fetch from DB
type Source interface {
    Name() string
    Scraper() scraper.Scraper
    Warmup(ctx context.Context) error
    BootstrapDiscovery(cfg config.Config) []TaskSeed
    ExpandDiscovery(ctx context.Context, cfg config.Config) ([]TaskSeed, error)
    DiscoveryScrapeTypes(cfg config.Config) []domain.PageType
    // …
}
```

**Реестр** — [`internal/sources/registry.go`](../../services/scraper/internal/sources/registry.go):

```go
var builtinOrder = []string{
    domain.SourceDns,
    domain.SourceWildberries,
    domain.SourceYandex,
    domain.SourceSprut,
    domain.SourcePrinter,
}

func NewRegistry(cfg config.Config, log zerolog.Logger) (Registry, error) { … }
```

**CLI делегирует в pipeline** — [`internal/cli/scrape.go`](../../services/scraper/internal/cli/scrape.go):

```go
registry, err := sources.NewRegistry(cfg, logger)
selected := registry.Selected(sourcesFlag)

if discoveryOnly {
    return (&pipeline.DiscoveryPipeline{
        Sources: selected, ScraperMap: registry.ScraperMap(), …
    }).Run(ctx)
}
return (&pipeline.ListingPipeline{ … }).Run(ctx)
```

**Discovery pipeline** (шаги A → B → C) — [`internal/pipeline/discovery.go`](../../services/scraper/internal/pipeline/discovery.go):

```go
// DiscoveryPipeline runs scrape --discovery for each Source:
//   A BootstrapDiscovery → PersistSeeds
//   B Warmup → ExpandDiscovery → PersistSeeds
//   C Warmup → ScrapePhase per DiscoveryScrapeTypes
```

**Общая фаза скрапа** — [`internal/pipeline/scrape_phase.go`](../../services/scraper/internal/pipeline/scrape_phase.go):

```go
// ScrapePhase loads tasks from DB, runs the worker pool, and persists snapshots.
func ScrapePhase(ctx, logger, m, taskRepo, snapshotRepo, sourceToScraper, cfg,
    sources, pageTypes []string, discoveryOnly bool, sqlPageType string) error
```

Внутри применяются job-фильтры из TOML (`filters.ScrapeTasks`).

### Как добавить новый маркетплейс

1. Скопировать [`internal/scrapers/example`](../../services/scraper/internal/scrapers/example) → `scrapers/<name>`
2. Скопировать [`internal/parsers/example`](../../services/scraper/internal/parsers/example) → `parsers/<name>`
3. Реализовать `sources.Source` (см. [`dns.go`](../../services/scraper/internal/sources/dns.go), [`wildberries.go`](../../services/scraper/internal/sources/wildberries.go))
4. Зарегистрировать в [`registry.go`](../../services/scraper/internal/sources/registry.go)
5. Документация по TOML-секциям — [`parsers/example/doc.go`](../../services/scraper/internal/parsers/example/doc.go)

---

## 3. DNS: новый источник

### Поток discovery (DNS)

```
discovery_seeds (config)
    → BootstrapDiscovery
    → RunDiscoveryBFS (max_bfs_fetches)
    → CreateTask(category) в tracked_pages
    → scrape --discovery (category snapshots)
    → parse --discovery
    → CreateTask(listing)
    → scrape --page-types listing
    → parse --page-types listing
```

| Слой | Файлы |
|------|-------|
| Source hooks | [`internal/sources/dns.go`](../../services/scraper/internal/sources/dns.go) |
| BFS обход каталога | [`internal/parsers/dns/discover_bfs.go`](../../services/scraper/internal/parsers/dns/discover_bfs.go) |
| HTTP + Chrome scraper | [`internal/scrapers/dns/`](../../services/scraper/internal/scrapers/dns/) |
| Парсинг category/listing | [`internal/parsers/dns/browse.go`](../../services/scraper/internal/parsers/dns/browse.go), [`listing_parser.go`](../../services/scraper/internal/parsers/dns/listing_parser.go) |

Пример bootstrap + BFS в source:

```go
// BootstrapDiscovery — discovery_seeds + search_queries from config.
func (DNS) BootstrapDiscovery(cfg config.Config) []TaskSeed { … }

// ExpandDiscovery — parsers/dns.RunDiscoveryBFS over catalog seeds.
func (s DNS) ExpandDiscovery(ctx context.Context, cfg config.Config) ([]TaskSeed, error) {
    stats, err := dnsParser.RunDiscoveryBFS(ctx, s.log, cfg.Scraping, cfg.Dns, cfg.Dns.DiscoverySeeds, &repo, s.Scraper())
    …
}
```

Конфиг для локального / pipeline прогона: [`config.dns-pipeline.toml`](../../services/pipeline-worker/config/jobs/scraper/config.dns-pipeline.toml)

```toml
[dns]
discovery_seeds = [
  "https://www.dns-shop.ru/catalog/9bbe8fe270e7c3ae/umnaa-tehnika/",
]
max_bfs_fetches = 50
browser_user_mode = true   # Chrome + Xvfb в Docker-образе

[jobs.scrape_discovery.dns]
discovery_bootstrap = ["seed"]   # только seed, без старых задач из БД
```

**Особенности DNS:**

- Qrator/antibot — warmup через HTTP, при 401 переключение на headless Chrome
- Изолированный Chrome profile в pipeline (`DNS_BROWSER_PROFILE` задаёт runner)
- `smart_home_device_markers` — фильтр на этапе parse listing (не путать с `limit` в jobs)

---

## 4. Job-фильтры в TOML

Новая секция конфига — [`internal/config/job_filter.go`](../../services/scraper/internal/config/job_filter.go):

```go
// JobsConfig holds per-job filters keyed by source name (e.g. jobs.scrape_discovery.dns).
type JobsConfig struct {
    ScrapeDiscovery map[string]SourceJobFilter `mapstructure:"scrape_discovery"`
    ParseDiscovery  map[string]SourceJobFilter `mapstructure:"parse_discovery"`
    Scrape          map[string]SourceJobFilter `mapstructure:"scrape"`
    Parse           map[string]SourceJobFilter `mapstructure:"parse"`
}

type SourceJobFilter struct {
    Limit              int       `mapstructure:"limit"`
    URLContains        []string  `mapstructure:"url_contains"`
    DiscoveryBootstrap []string  `mapstructure:"discovery_bootstrap"` // "seed", "db"
    CreatedAfter       time.Time `mapstructure:"created_after"`
    ScrapedAfter       time.Time `mapstructure:"scraped_after"`
    // …
}
```

Пример в TOML:

```toml
[jobs.scrape_discovery.dns]
discovery_bootstrap = ["seed"]   # или ["seed", "db"]
# limit = 10

[jobs.scrape.dns]
# created_after = 2026-07-09T00:00:00Z   # только свежие listing tasks
```

`discovery_bootstrap`:

| Значение | Поведение |
|----------|-----------|
| `seed` | Bootstrap + BFS из `discovery_seeds` / `search_queries` |
| `db` | Дополнительно подтягивать discovery-задачи из `tracked_pages` |
| `["seed"]` | Чистый прогон от seed (рекомендуется после очистки БД) |

---

## 5. Pipeline-worker: 4 job'а scraper

### Было (`main`)

Типичный workflow: `scraper scrape` → `scraper parse` → extractor → …

### Стало (renovate)

[`services/pipeline-worker/config/pipeline.toml`](../../services/pipeline-worker/config/pipeline.toml):

```toml
[pipeline]
job_names = [
  "scraper-scrape-discovery",
  "scraper-parse-discovery",
  "scraper-scrape",
  "scraper-parse",
]

[[jobs]]
name = "scraper-scrape-discovery"
image = "scraper:latest"
config_path = "scraper/config.dns-pipeline.toml"
command = ["/scraper", "scrape", "--config", "{{config_path}}", "--discovery", "--sources", "dns"]
shm_size = "2g"

[[jobs]]
name = "scraper-parse-discovery"
command = ["/scraper", "parse", "--config", "{{config_path}}", "--discovery", "--sources", "dns"]

[[jobs]]
name = "scraper-scrape"
command = ["/scraper", "scrape", "--config", "{{config_path}}", "--page-types", "listing", "--sources", "dns"]

[[jobs]]
name = "scraper-parse"
command = ["/scraper", "parse", "--config", "{{config_path}}", "--page-types", "listing", "--sources", "dns"]
```

Temporal workflow последовательно запускает job'ы — [`workflows/catalog_pipeline.go`](../../services/pipeline-worker/workflows/catalog_pipeline.go):

```go
for _, job := range input.Jobs {
    params := pipeline.RunContainerParams{
        Name: job.Name, Image: job.Image, Command: job.Command, …
    }
    // activity RunContainer
}
```

Runner пробрасывает env и изолирует Chrome profile — [`internal/docker/runner.go`](../../services/pipeline-worker/internal/docker/runner.go).

---

## 6. Observability

- Scraper и pipeline-worker шлют логи/метрики через OTLP (`shared/telemetry/go`)
- Pipeline-worker логирует stdout контейнеров job'ов в structured JSON
- Grafana / Jaeger / Loki — через `make monitoring-up`

Метрики scraper: `tasks_before_filter`, `tasks_matched`, `category_snapshots_matched`, `skipped_markers` и др. — [`internal/metrics/metrics.go`](../../services/scraper/internal/metrics/metrics.go)

---

## Полезные команды

### Первый запуск pipeline (DNS)

Перед `make pipeline-trigger` при необходимости задайте proxy в [`config.dns-pipeline.toml`](../../services/pipeline-worker/config/jobs/scraper/config.dns-pipeline.toml) (`[scraping].proxy`).

```bash
make help                  # все команды из корня

make monitoring-up         # OTEL + Grafana + Jaeger (нужен до pipeline)
make pipeline-build        # собрать scraper, extractor, worker, trigger
make pipeline-up           # Temporal + DB + pipeline-worker
make pipeline-migrate      # миграции catalog DB
make pipeline-seed         # начальные tracked_pages (опционально)
make pipeline-trigger      # запустить CatalogPipelineWorkflow
make pipeline-logs         # логи worker + temporal + postgresql
make pipeline-ps           # статус + URL Temporal UI
```

Полный стек на одной машине:

```bash
make pipeline-stack-up     # monitoring + build + pipeline + migrate
make pipeline-trigger
```

### Локальный scraper (без Temporal)

```bash
cd services/scraper

# Discovery: seed → BFS → scrape categories
go run ./cmd/scraper scrape --discovery --sources dns \
  --config ./cmd/scraper/config.local.toml

# Parse categories → создать listing tasks
go run ./cmd/scraper parse --discovery --sources dns \
  --config ./cmd/scraper/config.local.toml

# Scrape + parse listings
go run ./cmd/scraper scrape --page-types listing --sources dns \
  --config ./cmd/scraper/config.local.toml
go run ./cmd/scraper parse --page-types listing --sources dns \
  --config ./cmd/scraper/config.local.toml
```

### Тесты

```bash
cd services/scraper
go test ./internal/sources/... ./internal/parsers/dns/... ./internal/pipeline/...
```

E2E/smoke (требует сеть, опционально proxy):

```bash
cd services/scraper
go test -run TestDNS -v ./internal/sources/...
```

### БД (pipeline compose)

```bash
docker compose -p catalog-pipeline -f docker-compose.pipeline.yaml \
  exec catalog-postgresql psql -U catalog -d smart_home

# Примеры:
# SELECT page_type, COUNT(*) FROM tracked_pages WHERE source_name='dns' GROUP BY page_type;
# SELECT COUNT(*) FROM parsed_listing_snapshots;
```

### Остановка

```bash
make pipeline-down
make monitoring-down
# или
make pipeline-stack-down
```

---

## Карта ключевых файлов

```
.
├── Makefile                              # корневые команды (pipeline-*, monitoring-*, app-*)
├── docker-compose.pipeline.yaml          # Temporal + catalog DB + pipeline-worker
├── docs/renovate/scraper-pipeline.md     # этот документ
│
├── services/pipeline-worker/
│   ├── config/pipeline.toml              # workflow: список job'ов
│   ├── config/jobs/scraper/
│   │   ├── config.dns-pipeline.toml      # DNS pipeline (make pipeline-trigger)
│   │   └── config.dns-full.toml          # полный конфиг с фильтрами по датам
│   ├── workflows/catalog_pipeline.go
│   └── internal/docker/runner.go         # запуск job-контейнеров
│
└── services/scraper/
    ├── cmd/scraper/main.go
    ├── internal/
    │   ├── cli/scrape.go, parse.go       # точки входа CLI
    │   ├── config/job_filter.go          # [jobs.*] фильтры
    │   ├── sources/                      # контракт + registry + dns/wb/…
    │   ├── pipeline/                     # DiscoveryPipeline, ListingPipeline, ScrapePhase
    │   ├── scrapers/dns/                 # HTTP + Chrome
    │   ├── parsers/dns/                  # BFS, category, listing parsers
    │   └── parsers/example/doc.go        # шаблон + документация TOML
    └── Dockerfile                        # Chromium + Xvfb для DNS/WB browser mode
```

---

## Что читать в первую очередь (порядок для онбординга)

1. [`Makefile`](../../Makefile) — `make help`
2. [`docs/renovate/scraper-pipeline.md`](scraper-pipeline.md) — этот файл
3. [`internal/sources/source.go`](../../services/scraper/internal/sources/source.go) — контракт
4. [`internal/pipeline/discovery.go`](../../services/scraper/internal/pipeline/discovery.go) — flow discovery
5. [`internal/sources/dns.go`](../../services/scraper/internal/sources/dns.go) — пример реализации
6. [`config/pipeline.toml`](../../services/pipeline-worker/config/pipeline.toml) — как job'ы связаны в Temporal
7. [`parsers/example/doc.go`](../../services/scraper/internal/parsers/example/doc.go) — как устроен TOML

---

## Связанные ветки

| Ветка | Назначение |
|-------|------------|
| `main` | стабильная база, знакомая новому разработчику |
| `feature/scraper-pipeline-refactor` | активная разработка рефакторинга |
| `renovate/scraper-pipeline-on-main` | merge для онбординга и ревью «что изменилось» |

После ревью renovate-ветку мержим в `main` через PR с ссылкой на этот документ.
