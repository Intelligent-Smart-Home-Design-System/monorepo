FROM golang:1.25-alpine AS build

WORKDIR /src
COPY services/simulation/go.mod services/simulation/go.sum ./services/simulation/
WORKDIR /src/services/simulation
RUN go mod download

WORKDIR /src
COPY services/simulation/ ./services/simulation/
WORKDIR /src/services/simulation
RUN go build -o /out/simulation ./cmd/simulation

FROM alpine:3.21

WORKDIR /app
COPY --from=build /out/simulation /usr/local/bin/simulation
EXPOSE 8080
ENTRYPOINT ["simulation"]
