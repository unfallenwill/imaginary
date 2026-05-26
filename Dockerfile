# syntax=docker/dockerfile:1

# ---- Builder stage ----
FROM golang:1.26.3-bookworm AS builder

ARG IMAGINARY_VERSION=dev

WORKDIR /app

RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    DEBIAN_FRONTEND=noninteractive \
    apt-get update && \
    apt-get install --no-install-recommends -y \
    libvips-dev \
    libavcodec-dev \
    libavdevice-dev \
    libavfilter-dev \
    libavformat-dev \
    libavutil-dev \
    libswresample-dev \
    libswscale-dev

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
FROM debian:bookworm-slim

ARG IMAGINARY_VERSION

LABEL org.opencontainers.image.title="imaginary" \
      org.opencontainers.image.description="Fast HTTP microservice for high-level image processing" \
      org.opencontainers.image.url="https://github.com/h2non/imaginary" \
      org.opencontainers.image.source="https://github.com/h2non/imaginary" \
      org.opencontainers.image.version="${IMAGINARY_VERSION}" \
      org.opencontainers.image.authors="tomas@aparicio.me"

# FFmpeg 5.1.x runtime libraries (sonames pinned to Debian bookworm libav* package versions).
# If the base image is upgraded to a newer Debian release, bump these sonames accordingly.
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    DEBIAN_FRONTEND=noninteractive \
    apt-get update && \
    apt-get install --no-install-recommends -y \
    ca-certificates \
    libavcodec59 \
    libavdevice59 \
    libavfilter8 \
    libavformat59 \
    libavutil57 \
    libswresample4 \
    libswscale6 \
    libvips42

COPY --from=builder /out/imaginary /usr/local/bin/imaginary

ENV PORT=9000

USER nobody

STOPSIGNAL SIGTERM
EXPOSE 9000

ENTRYPOINT ["/usr/local/bin/imaginary"]
