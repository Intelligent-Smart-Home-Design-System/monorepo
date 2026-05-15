FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/layout-worker ./cmd/layout-worker

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /out/layout-worker /usr/local/bin/layout-worker
COPY internal/configs ./internal/configs
ENTRYPOINT ["/usr/local/bin/layout-worker"]
