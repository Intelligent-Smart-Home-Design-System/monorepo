FROM golang:1.25-alpine AS build
WORKDIR /src

COPY shared/telemetry/go/go.mod shared/telemetry/go/go.sum ./shared/telemetry/go/
COPY services/layout/go.mod services/layout/go.sum ./services/layout/

WORKDIR /src/services/layout
RUN go mod download

WORKDIR /src
COPY shared/ ./shared/
COPY services/layout/ ./services/layout/

WORKDIR /src/services/layout
RUN go build -o /out/layout-worker ./cmd/layout-worker

FROM alpine:3.21
WORKDIR /app
COPY --from=build /out/layout-worker /usr/local/bin/layout-worker
COPY services/layout/internal/configs /app/internal/configs
EXPOSE 2114
ENTRYPOINT ["layout-worker"]
