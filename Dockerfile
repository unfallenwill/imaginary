# syntax=docker/dockerfile:1

# ---- Builder stage ----
FROM ubuntu:26.04 AS builder

ARG IMAGINARY_VERSION=dev
ARG GOLANG_VERSION=1.26.3

# Install Go and FFmpeg build dependencies
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    DEBIAN_FRONTEND=noninteractive \
    apt-get update && \
    apt-get install --no-install-recommends -y \
    build-essential ca-certificates curl \
    libvips-dev \
    libavcodec-dev \
    libavdevice-dev \
    libavfilter-dev \
    libavformat-dev \
    libavutil-dev \
    libswresample-dev \
    libswscale-dev

RUN curl -fsSL https://go.dev/dl/go${GOLANG_VERSION}.linux-amd64.tar.gz | \
    tar -C /usr/local -xz && \
    ln -s /usr/local/go/bin/go /usr/local/bin/go

ENV GOPATH=/go
ENV PATH=/go/bin:/usr/local/go/bin:$PATH

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

ARG BUILD_TAGS="ffmpeg"

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 \
    go build -trimpath \
    -tags "${BUILD_TAGS}" \
    -o /out/imaginary \
    -ldflags="-s -w -X github.com/h2non/imaginary/internal/version.Version=${IMAGINARY_VERSION}" \
    ./cmd/imaginary

# ---- Runtime stage ----
FROM ubuntu:26.04

ARG IMAGINARY_VERSION

LABEL org.opencontainers.image.title="imaginary" \
      org.opencontainers.image.description="Fast HTTP microservice for high-level image processing" \
      org.opencontainers.image.url="https://github.com/h2non/imaginary" \
      org.opencontainers.image.source="https://github.com/h2non/imaginary" \
      org.opencontainers.image.version="${IMAGINARY_VERSION}" \
      org.opencontainers.image.authors="tomas@aparicio.me"

RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    DEBIAN_FRONTEND=noninteractive \
    apt-get update && \
    apt-get install --no-install-recommends -y \
    ca-certificates \
    libvips42 \
    libavcodec62 \
    libavdevice62 \
    libavfilter11 \
    libavformat62 \
    libavutil60 \
    libswresample6 \
    libswscale9

COPY --from=builder /out/imaginary /usr/local/bin/imaginary

ENV PORT=9000

USER nobody

STOPSIGNAL SIGTERM
EXPOSE 9000

ENTRYPOINT ["/usr/local/bin/imaginary"]
