ARG GO_IMAGE=golang:1.26-bookworm
ARG RUNTIME_IMAGE=debian:bookworm-slim
ARG TDLIB_IMAGE=gatec-tdlib:local

FROM ${TDLIB_IMAGE} AS tdlib

FROM ${GO_IMAGE} AS builder

WORKDIR /src
ENV CGO_ENABLED=1

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        build-essential \
        ca-certificates \
        libssl-dev \
        zlib1g-dev \
    && rm -rf /var/lib/apt/lists/*

COPY --from=tdlib /usr/local/include/ /usr/local/include/
COPY --from=tdlib /usr/local/lib/ /usr/local/lib/

WORKDIR /src
COPY go.mod go.sum ./
COPY vendor ./vendor
COPY . .

RUN go build -mod=vendor -trimpath -ldflags='-s -w' -o /out/gatec ./cmd \
	&& go build -mod=vendor -trimpath -ldflags='-s -w' -o /out/gatec-worker ./cmd/worker \
	&& go build -mod=vendor -trimpath -ldflags='-s -w' -o /out/gatec-signal-replay ./cmd/signal-replay \
	&& go build -mod=vendor -trimpath -ldflags='-s -w' -o /out/gatec-airdrop-catcher ./cmd/airdrop-catcher

FROM ${RUNTIME_IMAGE} AS runtime

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        libstdc++6 \
        libssl3 \
        tzdata \
        zlib1g \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /out/gatec /usr/local/bin/gatec
COPY --from=builder /out/gatec-worker /usr/local/bin/gatec-worker
COPY --from=builder /out/gatec-signal-replay /usr/local/bin/gatec-signal-replay
COPY --from=builder /out/gatec-airdrop-catcher /usr/local/bin/gatec-airdrop-catcher

RUN mkdir -p /app/logs /app/.tdlib2

ENTRYPOINT ["/usr/local/bin/gatec"]
