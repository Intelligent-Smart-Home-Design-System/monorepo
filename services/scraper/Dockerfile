# syntax=docker/dockerfile:1

FROM golang:1.25

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/reference/dockerfile/#copy
COPY cmd/ ./cmd/
COPY internal/ ./internal/

RUN CGO_ENABLED=0 GOOS=linux go build -o /scraper ./cmd/scraper/

FROM ubuntu:noble

RUN apt-get update > /dev/null && \
    apt-get install --no-install-recommends -y \
    libnss3 \
    libxss1 \
    libasound2t64 \
    libxtst6 \
    libgtk-3-0 \
    libgbm1 \
    ca-certificates \
    dumb-init \
    > /dev/null && \
    rm -rf /var/lib/apt/lists/*

RUN touch /.dockerenv

COPY --from=builder /scraper /usr/bin/scraper

EXPOSE 8080 9090

ENTRYPOINT ["dumb-init", "--"]
CMD ["scraper"]
