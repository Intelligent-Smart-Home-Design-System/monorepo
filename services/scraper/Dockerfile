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

RUN apt-get update && \
    apt-get install --no-install-recommends -y \
    libnss3 \
    libxss1 \
    libasound2t64 \
    libxtst6 \
    libgtk-3-0 \
    libgbm1 \
    ca-certificates \
    dumb-init \
    && rm -rf /var/lib/apt/lists/*

# metrics (8080) and healthcheck (9090)
EXPOSE 8080 9090
