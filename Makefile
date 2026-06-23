PIPELINE_DIR := services/pipeline-worker

COMPOSE_MONITORING := docker compose -f docker-compose.monitoring.yaml
COMPOSE_APP       := docker compose -f docker-compose.apps.yaml
COMPOSE_APP_PROD  := docker compose -f docker-compose.apps.prod.yaml

.PHONY: help \
        monitoring-up monitoring-down \
        pipeline-build pipeline-migrate pipeline-up pipeline-up-shifted pipeline-down \
        pipeline-stack-up pipeline-stack-down \
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

# ─── Pipeline-worker (через вложенный Makefile) ─────────────────────

pipeline-build: ## Собрать образы pipeline (scraper, extractor, worker, …)
	$(MAKE) -C $(PIPELINE_DIR) build

pipeline-migrate: ## Прогнать миграции catalog DB
	$(MAKE) -C $(PIPELINE_DIR) migrate

pipeline-up: ## Поднять pipeline-worker (дефолтные порты)
	$(MAKE) -C $(PIPELINE_DIR) up

pipeline-up-shifted: ## Поднять pipeline-worker со сдвигом портов (.env.shift)
	$(MAKE) -C $(PIPELINE_DIR) ENV_FILE=.env.shift up

pipeline-down: ## Остановить pipeline-worker (дефолтные порты)
	$(MAKE) -C $(PIPELINE_DIR) down

pipeline-down-shifted: ## Остановить pipeline-worker (порты из .env.shift)
	$(MAKE) -C $(PIPELINE_DIR) ENV_FILE=.env.shift down

pipeline-stack-up: monitoring-up pipeline-build pipeline-up ## Мониторинг + pipeline без app и без Terraform
	$(MAKE) -C $(PIPELINE_DIR) migrate

pipeline-stack-down: pipeline-down monitoring-down ## Остановить мониторинг + pipeline

# ─── App (часть 2) ──────────────────────────────────────────────────

app-up: ## Поднять main-pipeline (без тестового профиля)
	$(COMPOSE_APP_PROD) up -d --build

app-down: ## Остановить main-pipeline
	$(COMPOSE_APP_PROD) down

# ─── Полный стек (3 части, pipeline со сдвигом портов) ──────────────

up: monitoring-up pipeline-build pipeline-up-shifted app-up ## Всё: мониторинг + pipeline (.env.shift) + app
	$(MAKE) -C $(PIPELINE_DIR) ENV_FILE=.env.shift migrate

down: ## Остановить всё: app + pipeline (.env.shift) + мониторинг
	$(COMPOSE_APP_PROD) down
	$(MAKE) -C $(PIPELINE_DIR) ENV_FILE=.env.shift down
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
