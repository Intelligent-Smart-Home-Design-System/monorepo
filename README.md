# Smart Home Design System — монорепозиторий

Интеллектуальный планировщик умного дома: по плану квартиры и требованиям
подбирает наборы устройств из реального каталога, собранного парсингом маркетплейсов.

## Архитектура:

Система делится на две независимо разворачиваемые части.

- **Часть 1 — пайплайн построения каталога.** Скрейпинг, извлечение данных
  (LLM), построение каталога и расчёт качества, оркестрируемые через Temporal.
  Разворачивается через **Terraform + Docker**.
- **Часть 2 — фронтенд + бэкенд.** Next.js-фронтенд, `frontend-api` (отдаёт
  каталог и планы), `api-gateway` (auth + запуск Temporal-воркфлоу), воркеры и
  nginx как единая точка входа. Разворачивается через **Docker Compose**.
  Все сервисы части 2 ходят в **одну БД `smart_home`**.

Единая точка входа — **nginx на `:8090`**:

| Путь | Куда проксируется |
|------|-------------------|
| `/` и `/_next/*` | фронтенд (Next.js) |
| `/api/v1/auth/*` | `api-gateway` (`/auth/*`) — регистрация / вход / refresh |
| `/api/v1/*` | `frontend-api` — каталог, планы |
| `/start`, `/result`, `/auth/*`, `/healthz` | `api-gateway` |

## Быстрый старт

```bash
# TODO запуск через makefile
```

## Структура проекта

```text
.
├── Makefile                      # TODO
├── README.md
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
│   │   ├── docker-compose.yml
│   │   └── docker-compose_prod.yml
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
