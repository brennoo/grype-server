# AGENTS.md

Project context and conventions for AI coding assistants.

## Project overview

**Grype Server** is a Go HTTP service that accepts a Software Bill of Materials (SBOM) and optional VEX documents, and returns vulnerability findings using [Grype](https://github.com/anchore/grype). It is deployed as a stateful K8s pod (one Grype DB per replica).

## Repository layout

```
.
├── api/                        # Go module: OpenAPI 3.0 spec + generated code
│   ├── openapi.yaml            # Source of truth — edit this, then regenerate
│   ├── server-gen.yaml         # oapi-codegen config → api/server/server.gen.go
│   ├── client-gen.yaml         # oapi-codegen config → api/client/client.gen.go
│   ├── generate.sh             # Regenerates server/ and client/ from openapi.yaml
│   ├── server/server.gen.go    # Generated — do not edit by hand
│   └── client/client.gen.go    # Generated — do not edit by hand
├── grype-server/               # Go module: scanner implementation + HTTP server
│   ├── cmd/grype-server/       # main.go entry point (binary: grype-server)
│   └── pkg/
│       ├── config/             # Viper-based config (port, DB_ROOT_DIR, log level)
│       ├── rest/               # HTTP server wiring (oapi-codegen strict handler)
│       └── scanner/            # Grype integration; interface/ holds the Scanner interface
├── Dockerfile
├── Makefile
├── .golangci.yml
└── .gitignore
```

## Go module setup

Two separate `go.mod` files connected by a `replace` directive in `grype-server/go.mod`:

```
replace github.com/openclarity/grype-server/api v0.0.0 => ./../api
```

When adding or removing dependencies, run `go mod tidy` inside the relevant module directory, not from the root.

## Common commands

```bash
make build             # compile → grype-server/bin/grype-server
make test              # go test ./pkg/... (unit tests only)
make lint              # golangci-lint run (requires golangci-lint v2 in PATH)
make check             # test + lint
make api               # regenerate api/server/ and api/client/ from openapi.yaml
make docker            # build local single-arch image
make docker-multiarch  # buildx multi-arch linux/amd64 + linux/arm64 image
make push-multiarch    # buildx push linux/amd64 + linux/arm64 to registry
make clean             # remove grype-server/bin/*
```

## Integration tests

Tagged `//go:build integration` and require a pre-downloaded Grype DB:

```bash
cd grype-server
GRYPE_DB_DIR=/path/to/db go test -tags integration ./pkg/scanner/... -v
```

## API codegen

The API is defined in `api/openapi.yaml` (OpenAPI 3.0) and generated with `oapi-codegen v2`. To change the API:

1. Edit `api/openapi.yaml`
2. Run `make api` — this removes and recreates `api/server/` and `api/client/`
3. Update the handler implementation in `grype-server/pkg/rest/server.go` to match the new `StrictServerInterface`

Never edit `*.gen.go` files directly.

## Key design decisions

- **Stateful per pod**: Grype downloads a SQLite vulnerability DB to `DB_ROOT_DIR` (default `/data` in Docker, `./` locally). Each replica maintains its own DB. HPA scales replicas independently; cold start includes a DB download.
- **Concurrent scan safety**: `scanner.go` uses a `sync.RWMutex` to protect DB access. Multiple simultaneous scans are safe; a DB update briefly holds a write lock and closes the old provider after unlocking.
- **Grype identity**: `clio.Identification{Name: "grype-server"}` controls the Grype DB cache directory name. Changing this causes a one-time re-download on next start.
- **`ReadHeaderTimeout`**: Set to 10 s on the HTTP server to mitigate Slowloris attacks (enforced by `gosec` G112).

## Linting

`golangci-lint v2` with config in `.golangci.yml`. Enabled linters: `errcheck`, `govet`, `staticcheck`, `ineffassign`, `misspell`, `gosec`, `unused`. Generated files (`*.gen.go`) are excluded.

Run before every commit: `make lint`

## Docker

- Multi-stage build: `golang:1.26.8-alpine` builder → `dhi.io/alpine-base:3.24` runtime
- Binary: `/app/grype-server`; DB volume: `/data`
- Multi-arch build/push (amd64 + arm64) via `make docker-multiarch` / `make push-multiarch`
