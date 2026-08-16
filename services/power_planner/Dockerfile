FROM ubuntu:24.04 AS builder

RUN apt-get update && \
    apt-get install --no-install-recommends -y \
    ca-certificates \
    cmake \
    g++ \
    make \
    ninja-build \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY CMakeLists.txt ./
COPY include ./include
COPY src ./src
COPY tests ./tests

RUN cmake -S . -B build -G Ninja -DCMAKE_BUILD_TYPE=Release && \
    cmake --build build --target power_planner

FROM ubuntu:24.04

RUN apt-get update && \
    apt-get install --no-install-recommends -y \
    ca-certificates \
    libstdc++6 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/build/power_planner /usr/local/bin/power_planner
COPY data ./data

ENTRYPOINT ["power_planner"]
CMD ["--input", "data/sample_input.json"]
