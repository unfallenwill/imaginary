# ---- Builder stage ----
ARG GOLANG_VERSION=1.26.3
FROM golang:${GOLANG_VERSION}-bookworm AS builder

ARG IMAGINARY_VERSION=dev

# Install libvips from Debian packages — no source compilation needed.
# This layer is cached until the base image changes.
RUN DEBIAN_FRONTEND=noninteractive \
  apt-get update && \
  apt-get install --no-install-recommends -y \
  libvips-dev && \
  rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Cache go modules — only invalidated when go.mod or go.sum changes.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build.
COPY . .

RUN CGO_ENABLED=1 go build -trimpath \
    -o /out/imaginary \
    -ldflags="-s -w -X github.com/h2non/imaginary/internal/version.Version=${IMAGINARY_VERSION}" \
    ./cmd/imaginary

# ---- Runtime stage ----
FROM debian:bookworm-slim

ARG IMAGINARY_VERSION

LABEL maintainer="tomas@aparicio.me" \
      org.label-schema.description="Fast, simple, scalable HTTP microservice for high-level image processing with first-class Docker support" \
      org.label-schema.schema-version="1.0" \
      org.label-schema.url="https://github.com/h2non/imaginary" \
      org.label-schema.vcs-url="https://github.com/h2non/imaginary" \
      org.label-schema.version="${IMAGINARY_VERSION}"

# Install runtime libraries. apt pulls transitive dependencies automatically,
# so only direct libvips42 + image format libraries are needed.
RUN DEBIAN_FRONTEND=noninteractive \
  apt-get update && \
  apt-get install --no-install-recommends -y \
  ca-certificates \
  libvips42 \
  && apt-get clean \
  && rm -rf /var/lib/apt/lists/*

COPY --from=builder /out/imaginary /usr/local/bin/imaginary

ENV PORT=9000

USER nobody

ENTRYPOINT ["/usr/local/bin/imaginary"]

EXPOSE ${PORT}
