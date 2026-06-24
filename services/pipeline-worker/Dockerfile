FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY shared/telemetry/go/go.mod shared/telemetry/go/go.sum ./shared/telemetry/go/
COPY services/pipeline-worker/go.mod services/pipeline-worker/go.sum ./services/pipeline-worker/

WORKDIR /src/services/pipeline-worker
RUN go mod download

WORKDIR /src
COPY shared/ ./shared/
COPY services/pipeline-worker/ ./services/pipeline-worker/

WORKDIR /src/services/pipeline-worker
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/pipeline-worker ./cmd/pipeline-worker
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/pipeline-trigger ./cmd/pipeline-trigger

FROM alpine:3.20 AS base
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY services/pipeline-worker/config ./config

FROM base AS worker
COPY --from=builder /out/pipeline-worker /usr/local/bin/pipeline-worker
ENTRYPOINT ["/usr/local/bin/pipeline-worker"]

FROM base AS trigger
COPY --from=builder /out/pipeline-trigger /usr/local/bin/pipeline-trigger
ENTRYPOINT ["/usr/local/bin/pipeline-trigger"]
