FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/frontend-api .

FROM alpine:3.21
WORKDIR /app
COPY --from=build /out/frontend-api /usr/local/bin/frontend-api
EXPOSE 8080
ENTRYPOINT ["frontend-api"]
