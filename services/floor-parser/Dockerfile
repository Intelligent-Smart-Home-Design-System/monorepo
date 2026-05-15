FROM python:3.12-slim AS base
WORKDIR /app

COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY cmd ./cmd
COPY internal ./internal
COPY data ./data

FROM base AS api
ENTRYPOINT ["python", "-m", "cmd.floor_parser.main"]

FROM base AS worker
ENTRYPOINT ["python", "-m", "cmd.floor_parser.worker_main"]
