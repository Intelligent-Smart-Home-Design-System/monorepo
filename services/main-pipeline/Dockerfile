FROM golang:1.25-alpine AS build
WORKDIR /src

COPY shared/telemetry/go/go.mod shared/telemetry/go/go.sum ./shared/telemetry/go/
COPY services/main-pipeline/go.mod services/main-pipeline/go.sum ./services/main-pipeline/

WORKDIR /src/services/main-pipeline
RUN go mod download

WORKDIR /src
COPY shared/ ./shared/
COPY services/main-pipeline/ ./services/main-pipeline/

WORKDIR /src/services/main-pipeline
RUN go build -o /out/main-pipeline ./cmd/main-pipeline
RUN go build -o /out/api-gateway ./cmd/api-gateway

FROM alpine:3.21
WORKDIR /app
COPY --from=build /out/main-pipeline /usr/local/bin/main-pipeline
COPY --from=build /out/api-gateway /usr/local/bin/api-gateway
EXPOSE 8080 2112 2116
ENTRYPOINT ["main-pipeline"]
