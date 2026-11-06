# grype-server

[![CI](https://github.com/brennoo/grype-server/actions/workflows/ci.yml/badge.svg)](https://github.com/brennoo/grype-server/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/brennoo/grype-server)](https://github.com/brennoo/grype-server/releases)
[![Docker Pulls](https://img.shields.io/docker/pulls/brennoo/grype-server)](https://hub.docker.com/r/brennoo/grype-server)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](/LICENSE)

**grype-server** wraps [Anchore Grype](https://github.com/anchore/grype) in a lightweight HTTP service for centralized SBOM vulnerability scanning.

Instead of every CI job, scanner worker, or cluster workload downloading and caching the Grype vulnerability database independently, `grype-server` maintains a single local SQLite database, keeps it synced in the background, and exposes a simple REST endpoint to scan SBOMs on demand.

## Highlights

- **Centralized Vulnerability DB** — Downloads the Grype v6 database on startup and checks for updates every 3 hours with thread-safe, zero-downtime hot-swapping.
- **Broad SBOM Compatibility** — Accepts CycloneDX (JSON & XML), SPDX, and Syft JSON formats out of the box.
- **First-Class VEX Support** — Supply [OpenVEX](https://openvex.dev/) or [CSAF](https://oasis-open.github.io/csaf-documentation/) documents alongside an SBOM to automatically filter out `not_affected` and `fixed` vulnerabilities.
- **Production & Kubernetes Ready** — Multi-arch container image (`linux/amd64`, `linux/arm64`), non-root execution (`1000:1000`), and dedicated readiness/liveness probes.

## Quick Start

Pull the image from Docker Hub and start the server:

```bash
docker pull brennoo/grype-server:v0.9.0

docker run -d \
  --name grype-server \
  -p 9991:9991 \
  -p 8080:8080 \
  -v grype-db:/data \
  brennoo/grype-server:v0.9.0 run --log-level info
```

> [!TIP]
> Mounting a volume at `/data` persists the Grype SQLite database across container restarts so cold starts don't require a fresh download.

## Usage

### Scan an SBOM

Send a base64-encoded SBOM to `POST /api/scanSBOM`:

```bash
curl -s -X POST http://localhost:9991/api/scanSBOM \
  -H "Content-Type: application/json" \
  --data-binary @- <<EOF
{
  "sbom": "$(base64 < sbom.json | tr -d '\n')"
}
EOF
```

The API returns a JSON payload containing the base64-encoded Grype scan document under `.vulnerabilities`. To decode and inspect the findings with `jq`:

```bash
curl -s -X POST http://localhost:9991/api/scanSBOM \
  -H "Content-Type: application/json" \
  -d "{\"sbom\": \"$(base64 < sbom.json | tr -d '\n')\"}" \
  | jq -r '.vulnerabilities' | base64 -d | jq .
```

> [!NOTE]
> Supported SBOM formats include **CycloneDX** (JSON and XML), **SPDX** (JSON, tag-value, RDF), and **Syft** JSON.

### Scan an SBOM with VEX Documents

Provide VEX documents in the `vexDocuments` array to suppress vulnerabilities that have already been triaged as `not_affected` or `fixed`.

Both [OpenVEX](https://openvex.dev/) and [CSAF](https://oasis-open.github.io/csaf-documentation/) formats are supported and auto-detected. All documents in a single request must share the same format, as detection is based on the first entry.

```bash
curl -s -X POST http://localhost:9991/api/scanSBOM \
  -H "Content-Type: application/json" \
  --data-binary @- <<EOF
{
  "sbom": "$(base64 < sbom.json | tr -d '\n')",
  "vexDocuments": [
    "$(base64 < vex.openvex.json | tr -d '\n')"
  ]
}
EOF
```

Matching VEX statements are evaluated before results are returned, so suppressed vulnerabilities are excluded from the response.

## Configuration

`grype-server` is configured via environment variables and CLI flags:

| Environment Variable | Default | Description |
| :--- | :--- | :--- |
| `REST_SERVER_PORT` | `9991` | TCP port for the REST API server. |
| `HEALTH_CHECK_ADDRESS` | `:8080` | Bind address for the health check server (becomes ready once the DB is loaded). |
| `DB_ROOT_DIR` | `./` (`/data` in Docker) | Directory where the Grype SQLite vulnerability database is stored. |
| `DB_UPDATE_URL` | `https://grype.anchore.io/databases` | Upstream URL used to fetch Grype database updates. |
| `LOG_LEVEL` | `warning` | Log verbosity (`debug`, `info`, `warning`, `error`, `fatal`, `panic`). Also configurable via `--log-level`. |
| `LOG_FORMAT` | `json` | Log output format. Set to `text` for human-readable timestamps and console output. |

## Development

The repository consists of two Go modules (`api` for OpenAPI codegen and `grype-server` for the service implementation).

```bash
make build             # Compile binary to grype-server/bin/grype-server
make test              # Run unit tests
make lint              # Run golangci-lint
make check             # Run unit tests + linters
make api               # Regenerate server and client code from api/openapi.yaml
make docker            # Build local single-arch Docker image
make docker-multiarch  # Build multi-arch (linux/amd64, linux/arm64) image via buildx
```

## Contributing

Contributions, bug reports, and feature requests are welcome! Please review our [Contributing Guide](/CONTRIBUTING.md) and [Code of Conduct](/CODE_OF_CONDUCT.md) before opening an issue or pull request.

## License

Licensed under the [Apache License, Version 2.0](/LICENSE).
