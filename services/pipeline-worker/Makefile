ENV_FILE ?=
COMPOSE_ENV := $(if $(ENV_FILE),--env-file $(ENV_FILE),)
COMPOSE := docker compose $(COMPOSE_ENV) -f docker-compose.yaml

.PHONY: build build-pipeline-images up run down logs ps trigger migrate

build:
	docker build -f Dockerfile -t scraper:latest ../scraper
	docker build -f Dockerfile -t extractor:latest ../extractor
	docker build -f Dockerfile -t catalog-builder:latest ../catalog-builder
	docker build -f Dockerfile -t quality-calculator:latest ../quality-calculator
	docker build -f services/pipeline-worker/Dockerfile -t pipeline-worker:latest --target worker ../..
	docker build -f services/pipeline-worker/Dockerfile -t pipeline-trigger:latest --target trigger ../..

build-pipeline-images:
	docker build -f services/pipeline-worker/Dockerfile -t pipeline-worker:latest --target worker ../..
	docker build -f services/pipeline-worker/Dockerfile -t pipeline-trigger:latest --target trigger ../..

up:
	$(COMPOSE) up -d

run: build
	$(COMPOSE) up -d

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f pipeline-worker temporal temporal-ui catalog-postgresql

ps:
	$(COMPOSE) ps

trigger:
	$(COMPOSE) --profile tools run --rm pipeline-trigger

migrate:
	$(COMPOSE) run --rm catalog-db-migrate

deploy:
	git pull
	make build
	make migrate
	make up
