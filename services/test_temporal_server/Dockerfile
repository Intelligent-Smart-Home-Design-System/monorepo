FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY main.go ./

RUN go build -o test_temporal_server main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/test_temporal_server ./

EXPOSE 8080

CMD ["./test_temporal_server"]