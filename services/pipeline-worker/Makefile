# Thin wrapper — canonical targets live in monorepo root Makefile.
# Run from repo root: make pipeline-up, make pipeline-build, …

ROOT := ../..

.PHONY: build build-pipeline-images up run down logs ps trigger migrate seed deploy

build:
	$(MAKE) -C $(ROOT) pipeline-build

build-pipeline-images:
	docker build -f services/pipeline-worker/Dockerfile -t pipeline-worker:latest --target worker $(ROOT)
	docker build -f services/pipeline-worker/Dockerfile -t pipeline-trigger:latest --target trigger $(ROOT)

up:
	$(MAKE) -C $(ROOT) pipeline-up

run: build up

down:
	$(MAKE) -C $(ROOT) pipeline-down

logs:
	$(MAKE) -C $(ROOT) pipeline-logs

ps:
	docker compose -f $(ROOT)/docker-compose.pipeline.yaml ps

trigger:
	$(MAKE) -C $(ROOT) pipeline-trigger

migrate:
	$(MAKE) -C $(ROOT) pipeline-migrate

seed:
	$(MAKE) -C $(ROOT) pipeline-seed

deploy:
	$(MAKE) -C $(ROOT) deploy
