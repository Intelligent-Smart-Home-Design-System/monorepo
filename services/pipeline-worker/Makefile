COMPOSE := docker compose -f docker-compose.yaml

.PHONY: build build-pipeline-images up run down logs ps trigger migrate

build:
	docker build -t scraper:latest ../scraper
	docker build -t extractor:latest ../extractor
	docker build -t catalog-builder:latest ../catalog-builder
	docker build -t quality-calculator:latest ../quality-calculator
	docker build --target worker -t pipeline-worker:latest .
	docker build --target trigger -t pipeline-trigger:latest .

build-pipeline-images:
	docker build --target worker -t pipeline-worker:latest .
	docker build --target trigger -t pipeline-trigger:latest .

up:
	$(COMPOSE) up -d

run: build
	$(COMPOSE) up -d

down:
	$(COMPOSE) down -v

logs:
	$(COMPOSE) logs -f pipeline-worker temporal temporal-ui catalog-postgresql

ps:
	$(COMPOSE) ps

trigger:
	$(COMPOSE) --profile tools run --rm pipeline-trigger

migrate:
	$(COMPOSE) run --rm catalog-db-migrate
