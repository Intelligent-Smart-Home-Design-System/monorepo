# syntax=docker/dockerfile:1
# Build from monorepo root:
#   docker build -f services/scraper/Dockerfile -t scraper:latest .

FROM golang:1.25 AS build
WORKDIR /src

COPY shared/telemetry/go/go.mod shared/telemetry/go/go.sum ./shared/telemetry/go/
COPY services/scraper/go.mod services/scraper/go.sum ./services/scraper/

WORKDIR /src/services/scraper
RUN go mod download

WORKDIR /src
COPY shared/ ./shared/
COPY services/scraper/ ./services/scraper/

WORKDIR /src/services/scraper
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/scraper ./cmd/scraper/

FROM golang:1.25
WORKDIR /app

COPY --from=build /out/scraper /scraper

RUN apt-get update && \
    apt-get install --no-install-recommends -y \
    libnss3 \
    libxss1 \
    libasound2t64 \
    libxtst6 \
    libgtk-3-0 \
    libgbm1 \
    ca-certificates \
    dumb-init \
    && rm -rf /var/lib/apt/lists/*

EXPOSE 8080 9090
