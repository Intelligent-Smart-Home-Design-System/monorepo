COMPOSE_MONITORING := docker compose -f docker-compose.monitoring.yaml
COMPOSE_PIPELINE := docker compose -f services/pipeline-worker/docker-compose.yaml
COMPOSE_APP := docker compose -f services/main-pipeline/docker-compose.yml

.PHONY: help \
        monitoring-up monitoring-down monitoring-logs monitoring-ps \
        pipeline-build pipeline-up pipeline-run pipeline-down pipeline-logs pipeline-ps pipeline-trigger pipeline-migrate \
        app-build app-up app-run app-down app-logs app-ps \
        build-all up-all down-all

help: ## Показать все команды
	@echo "Использование: make <target>"
	@echo ""
	@echo "Мониторинг (общий стек):"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; /^monitoring-/ {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Каталог-пайплайн (Part 1):"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; /^pipeline-/ {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Фронтенд + бэкенд (Part 2):"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; /^app-/ {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Общие:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; /^(build-all|up-all|down-all)/ {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ─── Monitoring (shared stack) ───────────────────────────────────────

monitoring-up: ## Запустить централизованный стек мониторинга (OTEL Collector, Jaeger, Loki, Prometheus, Grafana)
	$(COMPOSE_MONITORING) up -d

monitoring-down: ## Остановить стек мониторинга
	$(COMPOSE_MONITORING) down

monitoring-logs: ## Логи стека мониторинга
	$(COMPOSE_MONITORING) logs -f

monitoring-ps: ## Статус контейнеров мониторинга
	$(COMPOSE_MONITORING) ps

# ─── Pipeline (Part 1) ──────────────────────────────────────────────

pipeline-build: ## Собрать все Docker-образы пайплайна
	$(MAKE) -C services/pipeline-worker build

pipeline-up: ## Запустить стек пайплайна
	$(COMPOSE_PIPELINE) up -d

pipeline-run: ## Собрать образы и запустить стек
	$(MAKE) -C services/pipeline-worker run

pipeline-down: ## Остановить стек пайплайна
	$(COMPOSE_PIPELINE) down

pipeline-logs: ## Логи пайплайна
	$(COMPOSE_PIPELINE) logs -f pipeline-worker temporal catalog-postgresql

pipeline-ps: ## Статус контейнеров пайплайна
	$(COMPOSE_PIPELINE) ps

pipeline-trigger: ## Ручной запуск пайплайна
	$(MAKE) -C services/pipeline-worker trigger

pipeline-migrate: ## Запустить миграции БД каталога
	$(MAKE) -C services/pipeline-worker migrate

# ─── App (Part 2) ───────────────────────────────────────────────────

app-build: ## Собрать Docker-образы фронтенда и бэкенда
	$(COMPOSE_APP) build

app-up: ## Запустить стек фронтенда и бэкенда
	$(COMPOSE_APP) up -d

app-run: ## Собрать образы и запустить стек
	$(COMPOSE_APP) up -d --build

app-down: ## Остановить стек фронтенда и бэкенда
	$(COMPOSE_APP) down

app-logs: ## Логи фронтенда и бэкенда
	$(COMPOSE_APP) logs -f

app-ps: ## Статус контейнеров фронтенда и бэкенда
	$(COMPOSE_APP) ps

# ─── Общие ──────────────────────────────────────────────────────────

build-all: pipeline-build app-build ## Собрать все Docker-образы

up-all: monitoring-up pipeline-up app-up ## Запустить все стеки (мониторинг → пайплайн → приложение)

down-all: app-down pipeline-down monitoring-down ## Остановить все стеки
