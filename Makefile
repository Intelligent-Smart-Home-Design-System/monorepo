PIPELINE_DIR := services/pipeline-worker
PIPELINE_ENV_SHIFT := services/pipeline-worker/.env.shift

COMPOSE_MONITORING := docker compose -f docker-compose.monitoring.yaml
COMPOSE_PIPELINE  := docker compose -f docker-compose.pipeline.yaml
COMPOSE_APP       := docker compose -f docker-compose.apps.yaml
COMPOSE_APP_PROD  := docker compose -f docker-compose.apps.prod.yaml

.PHONY: help \
        monitoring-up monitoring-down \
        pipeline-build pipeline-migrate pipeline-seed pipeline-up pipeline-up-shifted pipeline-down \
        pipeline-stack-up pipeline-stack-down pipeline-trigger pipeline-logs \
        app-up app-down \
        up down \
        up-test down-test \
        seed-catalog \
        deploy

help: ## Показать команды
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'

# ─── Мониторинг ─────────────────────────────────────────────────────

monitoring-up: ## Поднять мониторинг (OTEL, Jaeger, Loki, Prometheus, Grafana)
	$(COMPOSE_MONITORING) up -d --build

monitoring-down: ## Остановить мониторинг
	$(COMPOSE_MONITORING) down

# ─── Pipeline (docker-compose.pipeline.yaml в корне) ────────────────

pipeline-build: ## Собрать образы pipeline (scraper, extractor, worker, …)
	docker build -t scraper:latest services/scraper
	docker build -t extractor:latest services/extractor
	docker build -t catalog-builder:latest services/catalog-builder
	docker build -t quality-calculator:latest services/quality-calculator
	docker build -f services/pipeline-worker/Dockerfile -t pipeline-worker:latest --target worker .
	docker build -f services/pipeline-worker/Dockerfile -t pipeline-trigger:latest --target trigger .

pipeline-migrate: ## Прогнать миграции catalog DB
	$(COMPOSE_PIPELINE) run --rm catalog-db-migrate

pipeline-seed: ## Заполнить tracked_pages начальными задачами scraper (ON CONFLICT DO NOTHING)
	$(COMPOSE_PIPELINE) run --rm catalog-db-seed-tracked-pages

pipeline-up: ## Поднять pipeline-worker (дефолтные порты)
	$(COMPOSE_PIPELINE) up -d --build

pipeline-up-shifted: ## Поднять pipeline-worker со сдвигом портов (.env.shift)
	$(COMPOSE_PIPELINE) --env-file $(PIPELINE_ENV_SHIFT) up -d --build

pipeline-down: ## Остановить pipeline-worker (дефолтные порты)
	$(COMPOSE_PIPELINE) down

pipeline-down-shifted: ## Остановить pipeline-worker (порты из .env.shift)
	$(COMPOSE_PIPELINE) --env-file $(PIPELINE_ENV_SHIFT) down

pipeline-stack-up: monitoring-up pipeline-build pipeline-up ## Мониторинг + pipeline без app
	$(MAKE) pipeline-migrate

pipeline-stack-down: pipeline-down monitoring-down ## Остановить мониторинг + pipeline

pipeline-trigger: ## Запустить catalog pipeline workflow вручную (Temporal trigger)
	$(COMPOSE_PIPELINE) --profile tools run --rm pipeline-trigger

pipeline-logs: ## Логи pipeline-worker, temporal, catalog-postgresql
	$(COMPOSE_PIPELINE) logs -f pipeline-worker temporal temporal-ui catalog-postgresql

# ─── App (часть 2) ──────────────────────────────────────────────────

app-up: ## Поднять main-pipeline (без тестового профиля)
	$(COMPOSE_APP_PROD) up -d --build

app-down: ## Остановить main-pipeline
	$(COMPOSE_APP_PROD) down

# ─── Полный стек (3 части, pipeline со сдвигом портов) ──────────────

up: monitoring-up pipeline-build pipeline-up-shifted app-up ## Всё: мониторинг + pipeline (.env.shift) + app
	$(COMPOSE_PIPELINE) --env-file $(PIPELINE_ENV_SHIFT) run --rm catalog-db-migrate

down: ## Остановить всё: app + pipeline (.env.shift) + мониторинг
	$(COMPOSE_APP_PROD) down
	$(COMPOSE_PIPELINE) --env-file $(PIPELINE_ENV_SHIFT) down
	$(COMPOSE_MONITORING) down

# ─── Тест (мониторинг + app --profile test) ─────────────────────────

up-test: monitoring-up ## Поднять мониторинг + main-pipeline (--profile test)
	$(COMPOSE_APP) --profile test up -d --build

down-test: ## Остановить main-pipeline (test) + мониторинг
	$(COMPOSE_APP) --profile test down
	$(COMPOSE_MONITORING) down

seed-catalog: ## Пересобрать seed_catalog.sql из catalog.json
	python3 services/main-pipeline/config/generate_seed_catalog.py

# ─── Деплой ─────────────────────────────────────────────────────────

deploy: ## git pull + пересобрать и перезапустить (prod)
	git pull
	$(MAKE) up
