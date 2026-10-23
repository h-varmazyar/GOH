ARG GO_IMAGE=golang:1.26-bookworm
ARG TDLIB_REF=master

FROM ${GO_IMAGE}
ARG TDLIB_REF

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        build-essential \
        ca-certificates \
        cmake \
        git \
        gperf \
        libssl-dev \
        php-cli \
        zlib1g-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /tmp

RUN git clone --depth 1 --branch "${TDLIB_REF}" https://github.com/tdlib/td.git \
    && cd td \
    && php SplitSource.php \
    && rm -rf build \
    && mkdir build \
    && cd build \
    && cmake -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX:PATH=/usr/local .. \
    && cmake --build . --target install --parallel "$(nproc)" \
    && cd .. \
    && php SplitSource.php --undo \
    && cd / \
    && rm -rf /tmp/td