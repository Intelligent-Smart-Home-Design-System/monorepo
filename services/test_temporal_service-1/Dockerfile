FROM golang:1.21-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/worker ./worker
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/client ./client

FROM alpine:3.20 AS base
RUN apk add --no-cache ca-certificates
RUN adduser -D -H appuser
USER appuser

FROM base AS worker
COPY --from=builder /out/worker /usr/local/bin/worker
ENTRYPOINT ["/usr/local/bin/worker"]

FROM base AS client
COPY --from=builder /out/client /usr/local/bin/client
ENTRYPOINT ["/usr/local/bin/client"]
