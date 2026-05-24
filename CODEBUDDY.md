# CODEBUDDY.md

This file provides guidance to CodeBuddy Code when working with code in this repository.

## Project Overview

imaginary is a fast HTTP microservice for image processing, written in Go and backed by [bimg](https://github.com/h2non/bimg) (Go bindings) and [libvips](https://github.com/libvips/libvips) (C library). It exposes image operations (resize, crop, rotate, watermark, etc.) as HTTP endpoints and supports multiple image sources (POST body, local filesystem, remote HTTP URL, S3-compatible object storage).

## Build & Development Commands

Requires: Go 1.26+, libvips 8.8+, C compiler (gcc/clang).

```bash
make build              # Compile binary to bin/imaginary
make fmt                # Format all .go files with gofmt
make fmt-check          # Check formatting (CI gate)
make tidy-check         # Verify go.mod/go.sum are tidy (CI gate)
make vet                # Run go vet
make lint               # Run golangci-lint
make arch               # Run go-arch-lint (architecture boundary check)
make test               # Run tests
make race               # Run tests with race detector (CI gate)
make cover              # Generate coverage report
make cover-check        # Coverage gate (threshold: 50%, configurable via COVER_THRESHOLD)
make vuln               # Run govulncheck for known vulnerabilities
make quality            # Full gate: fmt-check + tidy-check + vet + lint + arch + race + build
```

Run a single test:
```bash
go test ./internal/image/... -run TestCrop -v
go test -race -count=1 ./internal/server/...
```

Run locally (requires libvips):
```bash
# If libvips installed via Homebrew (Linux/macOS):
LD_LIBRARY_PATH="$(brew --prefix vips)/lib:$LD_LIBRARY_PATH" go run ./cmd/imaginary -p 8088
# If libvips installed via apt (default library path, no extra config needed):
go run ./cmd/imaginary -p 8088
```

## Architecture

```
cmd/imaginary/main.go              # Entry point: config → dependency wiring → start server
internal/
  config/                           # CLI flags, env vars, YAML config parsing
  image/                            # Pure domain layer — image operations and types
  server/                           # HTTP transport — routing, controllers, middleware
  source/                           # Image source abstraction
    source.go                       # ImageSource interface + Resolver (strategy pattern)
    body/                           # POST multipart upload source
    fs/                             # Local filesystem source
    http/                           # Remote HTTP URL source (with origin allowlist)
    object/                         # S3-compatible object storage source
  storage/                          # Storage backend provider (S3)
  version/                          # Version constants
```

### Dependency direction (enforced by go-arch-lint + depguard)

```
cmd → config → server → source → image (leaf, no internal deps)
                        ↘ version (leaf, no internal deps)
storage → config + source
```

Key constraints:
- `internal/image` is a **pure domain package** — it must never import `net/http`, `server`, `source`, or `storage`. All I/O happens in `server` or `source`.
- `internal/source` cannot depend on `server`.
- `internal/storage` cannot depend on `server`.
- `internal/config` may depend on `server` (it builds `server.Config`).

### Key patterns

- **Operation pattern**: `img.Operation` is a `func([]byte, ImageOptions) (Image, error)` — each endpoint maps to one. Defined in `internal/image/image.go`.
- **Source resolver**: `source.Resolver` tries registered `ImageSource` implementations in order (body → object → fs → http) until one matches the request.
- **Error model**: `image.Error` uses semantic `Kind` constants (not HTTP status codes). Server layer maps kinds to HTTP statuses via `server/error.go`.
- **Config priority**: code defaults < config file < CLI flags < environment variables. Config is parsed in `internal/config/config.go`. Only 3 env vars override: `PORT`, `URL_SIGNATURE_KEY`, `GOLANG_LOG`. YAML file supports `${ENV_VAR}` interpolation.
- **ObjectStorage interface**: defined in `internal/source/source.go` at the consumer side (Go convention), implemented by `internal/storage/s3.go`.
- **Middleware chain**: `server/middleware.go` wraps handlers with endpoint filtering, throttling, CORS, API key auth, cache headers, and URL signature validation.

### Global state

- `controllers.remoteClient` — shared HTTP client for fetching watermark images
- `health.start` — server start time for uptime calculation

## Testing

Tests use `net/http/httptest` for HTTP-level integration tests against real libvips image processing. Test data lives in `testdata/`. No external mock frameworks — interfaces are mocked manually (e.g., `fakeObjectStorage`).

## Adding a New Image Operation

1. Add the operation function in `internal/image/image.go` (or a new file): `func MyOp(buf []byte, opts ImageOptions) (Image, error)`
2. If the operation has query parameters, add parsing in `internal/image/params.go` (`applyQueryParam`)
3. Register in `OperationsMap` in `internal/image/image.go` (for pipeline support)
4. Add route in `internal/server/server.go`: `mux.Handle(join(prefix, "/myop"), image(img.MyOp))`
5. Add test in `internal/image/` (unit) and/or `internal/server/` (HTTP integration)

## Quality Gates (CI)

CI runs on push/PR to master. Gates in order:
1. `make fmt-check` — formatting
2. `make tidy-check` — dependency tidiness
3. `make vet` — static analysis
4. `golangci-lint run` — lint (errcheck, govet, staticcheck, unused, gosec, ineffassign, unconvert, misspell, depguard, gocyclo, gocognit, dupl)
5. `govulncheck ./...` — vulnerability scan
6. `go-arch-lint check` — architecture boundary enforcement
7. `make race` — tests with race detector
8. `make cover-check` — coverage threshold (≥50%)
9. `make build` — compilation check

## Gotchas

- `go-arch-lint`: empty `mayDependOn: []` is invalid — omit the key entirely. `internal/source/**` glob does not match `internal/source` root package; use `[internal/source, internal/source/**]` array syntax.
- `make cover-check` requires `bc` for float comparison. If `go test` fails, `coverage.out` may not be generated (the script checks for this).
- golangci-lint v2 config uses `linters.settings` (not `linters-settings`) and `exclusions.rules` (not `exclude-rules`).
- `go-arch-lint` and `govulncheck` must be installed separately: `go install github.com/fe3dback/go-arch-lint@latest` and `go install golang.org/x/vuln/cmd/govulncheck@latest`.
