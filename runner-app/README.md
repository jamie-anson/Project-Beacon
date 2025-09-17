# Project Beacon Runner - Signature Verification Fixed

Cross-region benchmark execution engine for detecting AI model differences across geographic regions.

## Overview

The Project Beacon Runner executes benchmarks across multiple geographic regions via the Golem Network, detects cross-region differences, and stores results with cryptographic provenance.

## Architecture

- **JobSpec Handler**: Validates and processes signed benchmark specifications
- **Golem Execution Engine**: Manages multi-region container execution
- **Cross-Region Diff Module**: Detects and analyzes output differences
- **Storage & Transparency**: IPFS storage with Merkle tree transparency log

## Quick Start

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- PostgreSQL 15+
- Redis 7+

### Setup

```bash
# Clone the repository
git clone <repository-url>
cd runner-app

# Install dependencies
make setup

# Start supporting services
make docker-up

# Run development server
make dev
```

The API will be available at `http://localhost:8090` (fallback may choose another port if 8090 is busy; see Ports & Discovery below).

### Terminal labels

- Terminal A: Yagna daemon
- Terminal B: Go API server (local)
- Terminal C: Actions (curl, tests)
- Terminal D: Local infra (docker compose: Postgres/Redis)
- Terminal E: Cloud ops (Fly/Neon/Upstash/Grafana Cloud)

### API Endpoints

- `GET /health` - Aggregate health
- `GET /health/live` - Liveness probe
- `GET /health/ready` - Readiness probe
- `POST /api/v1/jobs` - Create new benchmark job
- `GET /api/v1/jobs` - List all jobs
- `GET /api/v1/jobs/:id` - Get specific job
- `POST /api/v1/jobs/:id/execute` - Execute job across regions
- `GET /api/v1/jobs/:id/executions` - Get job executions
- `GET /api/v1/jobs/:id/diffs` - Get cross-region diffs
  
- Admin (token required unless running in Gin debug mode):
  - `GET /admin/port` → `{ addr, strategy }`
  - `GET /admin/hints` → `{ base_url, resolved_addr, strategy }`

## Observability

Health probes and Prometheus metrics are exposed. Examples target `http://localhost:8090`.

### Health

```bash
curl -s http://localhost:8090/health | jq .
curl -s http://localhost:8090/health/live | jq .
curl -s -i http://localhost:8090/health/ready | sed -n '1,10p'
```

### Metrics

Prometheus text exposition format is available at both paths:

- `/metrics`
- `/api/v1/metrics` (alias under API namespace)

```bash
curl -sI http://localhost:8090/metrics | sed -n '1,10p'
curl -s http://localhost:8090/api/v1/metrics | head -n 20
```

Tracing is enabled via Gin's OpenTelemetry middleware (`otelgin`). The OTLP exporter is used when `OTEL_EXPORTER_OTLP_ENDPOINT` is set.

## Signing and Security

Project Beacon requires JobSpecs to be cryptographically signed. Trusted-keys allowlist, timestamp freshness, nonce replay protection, and unified error codes are implemented.

- See detailed guide: `docs/signing.md`

Quick submit example to the local server on `http://localhost:8090`:

```bash
curl -X POST http://localhost:8090/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d @examples/jobspec-who-are-you.json.signed
```

Common failure checks (expect 400 errors with structured codes):

```bash
# Missing nonce
curl -sS -X POST http://localhost:8090/api/v1/jobs -H 'Content-Type: application/json' -d '{"id":"demo","version":"1.0","metadata":{"timestamp":"2025-08-22T14:50:32Z"},"signature":"","public_key":""}' | jq

# Malformed timestamp
curl -sS -X POST http://localhost:8090/api/v1/jobs -H 'Content-Type: application/json' -d '{"id":"demo","version":"1.0","metadata":{"timestamp":"2025/08/22 14:50:32","nonce":"n1"},"signature":"","public_key":""}' | jq
```

## Development

### Project Structure

```
├── cmd/runner/          # Main application entry point
├── internal/
│   ├── api/            # HTTP API handlers
│   ├── jobspec/        # JobSpec validation and processing
│   ├── golem/          # Golem Network integration
│   │   └── client/     # Extracted Yagna REST client (interfaces + impl)
│   ├── diff/           # Cross-region difference detection
│   ├── storage/        # IPFS and database storage
│   └── db/             # Database models and migrations
├── pkg/
│   ├── crypto/         # Cryptographic utilities (Ed25519)
│   └── models/         # Shared data models
├── web/                # Frontend dashboard (React)
├── docker/             # Docker configurations
└── scripts/            # Utility scripts
```

### Running Tests

```bash
make test
```

### Building

```bash
make build
```

### Docker Development

```bash
# Start all services including the runner
make docker-full

# Stop all services
make docker-down
```

## Configuration

Environment variables:

- `DATABASE_URL` - PostgreSQL connection string
- `REDIS_URL` - Redis connection string  
- `IPFS_API_URL` - IPFS API endpoint
- `GOLEM_API_KEY` - Golem Network API key
- `HTTP_PORT` - HTTP server port in ":<port>" form (default: ":8090")
- `PORT_STRATEGY` - one of `strict`, `fallback`, or `ephemeral` (default: `fallback` for dev)
- `PORT_RANGE` - fallback scan range, e.g. `8090-8099`
- `RUNNER_HTTP_ADDR_FILE` - path to write the resolved addr (default: `.runner-http.addr`)
- `ADMIN_TOKEN` - token required for admin endpoints (public only in Gin debug mode)

Security and signing:

- `TRUST_ENFORCE` - enable trusted-keys allowlist enforcement (bool)
- `TRUSTED_KEYS_FILE` - path to trusted keys JSON file
- `TRUSTED_KEYS_RELOAD_SECONDS` - optional hot-reload interval for trusted keys
- `TIMESTAMP_MAX_SKEW_MINUTES` - allowed clock skew for timestamps
- `TIMESTAMP_MAX_AGE_MINUTES` - maximum allowed age for timestamps
- `REPLAY_PROTECTION_ENABLED` - enable nonce-based replay protection (requires Redis)
- `RUNNER_SIG_BYPASS` - development-only signature verification bypass (never use in prod/CI)

### Local env precedence (.env)

- `make dev` now loads variables from `.env` and **overrides any existing shell environment variables** for the current make invocation. This makes local development reproducible regardless of your shell state.
- To change a value, edit `.env`. To temporarily override, run `VAR=value make dev` and place the override later in `.env` if you want it persisted.

### Debug logging gates

Verbose debug instrumentation around canonicalization comparison and shadow v1 verification is gated behind a debug flag:

- Set `DEBUG=true` (or `LOG_LEVEL=debug`) to enable these logs.
- When not in debug, these logs are suppressed. The gating occurs in `internal/api/handlers_simple.go` using `logging.DebugEnabled()`.

### Ports, Strategies, and Discovery

- Strategies (`PORT_STRATEGY`):
  - `strict` (prod/staging): bind exactly `HTTP_PORT` or fail.
  - `fallback` (default dev): try `:8090`, scan `PORT_RANGE` on conflict.
  - `ephemeral` (tests/CI): bind to `:0`.

- Addr file (`RUNNER_HTTP_ADDR_FILE`, default `.runner-http.addr`) is written in all modes with the resolved `host:port`.

- Make targets and helper script:
  - `make port` → prints port from `.runner-http.addr`
  - `make addr` → prints `host:port`
  - `make base` → prints `http://localhost:<port>`
  - `./scripts/runner-port.sh --port|--addr|--base [--file <path>]`

- Examples (Terminal labels):
  - Terminal B (server, dev default 8090):
    ```bash
    PORT_STRATEGY=fallback HTTP_PORT=:8090 PORT_RANGE=8090-8099 make dev
    ```
  - Terminal B (ephemeral + addr file):
    ```bash
    ADMIN_TOKEN=dev-token PORT_STRATEGY=ephemeral RUNNER_HTTP_ADDR_FILE=.runner-http.addr make dev
    ```
  - Terminal C (actions):
    ```bash
    BASE=$(make base)
    curl -sS "$BASE/health/ready"
    curl -sS -H "X-Admin-Token: dev-token" "$BASE/admin/hints"
    ```

## JobSpec Format

```json
{
  "id": "benchmark-001",
  "version": "1.0",
  "benchmark": {
    "name": "Who Are You?",
    "description": "Text generation benchmark",
    "container": {
      "image": "beacon/text-gen",
      "tag": "latest",
      "resources": {
        "cpu": "1000m",
        "memory": "512Mi"
      }
    },
    "input": {
      "type": "prompt",
      "data": {
        "prompt": "Who are you? Describe yourself in 2-3 sentences."
      }
    }
  },
  "constraints": {
    "regions": ["US", "EU", "APAC"],
    "min_regions": 3,
    "timeout": "10m"
  },
  "signature": "...",
  "public_key": "..."
}
```

## Phase 1 Development Status

### Week 1-2: Foundation ✅
- [x] Project setup with Go, Gin, PostgreSQL
- [x] Basic API structure and database schema
- [x] Docker development environment
- [x] JobSpec and Receipt data models
- [ ] Ed25519 signature validation
- [ ] Basic Golem integration

### Week 3-4: Multi-Region Execution
- [ ] Geographic provider selection
- [ ] Parallel execution engine
- [ ] Enhanced receipt system
- [ ] Job queue management

### Week 5-6: Cross-Region Diff Engine
- [ ] Diff algorithm implementation
- [ ] Diff classification and scoring
- [ ] Diff JSON schema and API
- [ ] Automated difference detection

### Week 7-8: Storage & Dashboard
- [ ] IPFS integration and pinning
- [ ] Transparency log implementation
- [ ] React dashboard for diff visualization
- [ ] Export and reporting features

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

[License details to be added]

## Support

For questions and support, please open an issue or contact the Project Beacon team.
# Force Railway redeploy
# Trigger deployment test
