COMPOSE_MONITORING := docker compose -f docker-compose.monitoring.yaml
COMPOSE_PIPELINE  := docker compose -f services/pipeline-worker/docker-compose.yaml
COMPOSE_APP       := docker compose -f services/main-pipeline/docker-compose.yml
COMPOSE_APP_PROD      := docker compose -f services/main-pipeline/docker-compose_prod.yml

.PHONY: help \
        monitoring-up monitoring-down \
        pipeline-up pipeline-down \
        app-up app-down \
        up down \
        up-test down-test \
        up-prod down-prod \
        deploy

help: ## Показать команды
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

# ─── По отдельности ─────────────────────────────────────────────────

monitoring-up: ## Поднять мониторинг (OTEL, Jaeger, Loki, Prometheus, Grafana)
	$(COMPOSE_MONITORING) up -d --build

monitoring-down: ## Остановить мониторинг
	$(COMPOSE_MONITORING) down

pipeline-up: ## Поднять pipeline-worker
	$(COMPOSE_PIPELINE) up -d --build

pipeline-down: ## Остановить pipeline-worker
	$(COMPOSE_PIPELINE) down

app-up: ## Поднять main-pipeline (без тестового профиля)
	$(COMPOSE_APP_PROD) up -d --build

app-down: ## Остановить main-pipeline
	$(COMPOSE_APP_PROD) down

# ─── Полный стек ────────────────────────────────────────────────────

up: monitoring-up pipeline-up app-up ## Поднять всё: мониторинг + pipeline + app

down: ## Остановить всё: app + pipeline + мониторинг
	$(COMPOSE_APP_PROD) down
	$(COMPOSE_PIPELINE) down
	$(COMPOSE_MONITORING) down

# ─── Тест (мониторинг + app --profile test) ─────────────────────────

up-test: monitoring-up ## Поднять мониторинг + main-pipeline (--profile test)
	$(COMPOSE_APP) --profile test up -d --build

down-test: ## Остановить main-pipeline (test) + мониторинг
	$(COMPOSE_APP) --profile test down
	$(COMPOSE_MONITORING) down
# ─── Деплой ─────────────────────────────────────────────────────────

deploy: ## git pull + пересобрать и перезапустить (prod)
	git pull
	up
