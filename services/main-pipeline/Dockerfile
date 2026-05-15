FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/main-pipeline-worker ./cmd/main-pipeline-worker
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/main-pipeline-trigger ./cmd/main-pipeline-trigger

FROM alpine:3.20 AS base
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY config ./config

FROM base AS worker
COPY --from=builder /out/main-pipeline-worker /usr/local/bin/main-pipeline-worker
ENTRYPOINT ["/usr/local/bin/main-pipeline-worker"]

FROM base AS trigger
COPY --from=builder /out/main-pipeline-trigger /usr/local/bin/main-pipeline-trigger
ENTRYPOINT ["/usr/local/bin/main-pipeline-trigger"]
