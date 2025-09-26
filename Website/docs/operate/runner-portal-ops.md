---
id: runner-portal-ops
title: Runner Portal — Operations Guide
slug: /operate/portal
description: Backend contracts expected by the Portal UI, ports and terminals, and WebSocket event schema.
---

This page documents what the Portal expects from the backend.

## Ports & Terminals

- Terminal A: Yagna daemon
- Terminal B: Go API server on http://localhost:8090
- Terminal C: Actions (curl, tests)
- Terminal D: Postgres + Redis (docker compose)

## HTTP Endpoints (Terminal B)

- GET `/api/v1/transparency/root`
  - Returns `{ merkle_root | root, sequence, updated_at }`
- GET `/api/v1/transparency/proof?execution_id=...&ipfs_cid=...`
  - Returns proof object with `merkle_root/root`, `sequence`, proof nodes
- Optional gateway proxy:
  - GET `/api/v1/transparency/bundles/:cid` — serves CID content

## WebSocket (Terminal B)

- Path: `/ws`
- Emits newline-delimited JSON frames
- Events used by UI (any may be present):
  - `execution_id` (string)
  - `ipfs_cid` (string)
  - `merkle_root` (string)
  - `timestamp` or `time` (ISO string)

UI behavior:
- Header badge indicates Online/Offline/Error
- Tooltip includes retries and next reconnect delay
- Activity Feed renders available fields and persists them in `sessionStorage` (`beacon:activity`)

## Developer Pointers

- WebSocket/backoff logic: `portal/src/state/useWs.js`
- API helpers and gateway resolution:
  - `portal/src/lib/api/ipfs.js` → `getIpfsGateway()`, `bundleUrl(cid)`
  - `portal/src/lib/api/http.js` → shared HTTP wrappers emitting structured `ApiError`
  - `portal/src/lib/errorUtils.js` → `parseApiError()` for toast-friendly messaging
- Settings: `portal/src/pages/Settings.jsx`
- Dashboard & persistence: `portal/src/pages/Dashboard.jsx`
- Activity Feed: `portal/src/components/ActivityFeed.jsx`
- Modal & Proof viewer:
  - `portal/src/components/Modal.jsx`
  - `portal/src/components/ProofViewer.jsx`

## Example Commands (Terminal C)

Fetch transparency root:
```bash
curl -s http://localhost:8090/api/v1/transparency/root | jq
```

Fetch proof by execution id:
```bash
curl -s "http://localhost:8090/api/v1/transparency/proof?execution_id=<EXEC_ID>" | jq
```

Fetch proof by CID:
```bash
curl -s "http://localhost:8090/api/v1/transparency/proof?ipfs_cid=<CID>" | jq
```

## Observability Quickstart

Use these terminal-labeled steps to validate health, submit a job, and inspect metrics.

- __Terminal D: Postgres + Redis__
  - Ensure databases are up (compose or your preferred setup).

- __Terminal A: Yagna daemon__
  - Start your Golem/yagna stack as required for provider access.

- __Terminal B: Go API server__
  ```bash
  HTTP_PORT=:8090 \
  DATABASE_URL=postgres://postgres:password@localhost:5433/beacon_runner?sslmode=disable \
  REDIS_URL=redis://localhost:6379 \
  GOLEM_API_KEY=your-key \
  GOLEM_NETWORK=testnet \
  go run ./cmd/runner
  ```

- __Terminal C: Actions__
  - Health and metrics:
    ```bash
    curl -s http://localhost:8090/health | jq .
    curl -s http://localhost:8090/metrics | head
    ```
  - Submit a signed JobSpec and check status:
    ```bash
    go run ./scripts/sign_jobspec.go jobspec.json /tmp/signed_jobspec.json
    curl -sS -H "Content-Type: application/json" --data-binary @/tmp/signed_jobspec.json \
      http://localhost:8090/api/v1/jobs | jq .
    JOB_ID=job-001
    curl -s "http://localhost:8090/api/v1/jobs/$JOB_ID" | jq .
    curl -s "http://localhost:8090/api/v1/jobs/$JOB_ID?include=latest" | jq .
    ```
  - Key Prometheus metrics to watch:
    ```bash
    # HTTP
    curl -s http://localhost:8090/metrics | grep -E '^http_requests_total|^http_request_duration_seconds'

    # Jobs counters
    curl -s http://localhost:8090/metrics | grep -E '^jobs_enqueued_total|^jobs_processed_total|^jobs_failed_total'

    # Queue/Execution histograms
    curl -s http://localhost:8090/metrics | grep -E '^runner_queue_latency_seconds|^runner_execution_duration_seconds'

    # WebSocket
    curl -s http://localhost:8090/metrics | grep -E '^websocket_connections|^websocket_messages_'
    ```

### Grafana panels (included)

The runner dashboard at `runner-app/observability/grafana/dashboard-runner.json` includes panels for:
- Execution success rate, execution duration p95 by region
- Queue latency p95
- Jobs enqueued vs processed
- WebSocket connections
- HTTP rates and p95 latency

Import this JSON into Grafana (or use provisioning in `observability/grafana/provisioning/`).

## Responding to Outbox Alerts

Two alerts help detect a stuck outbox publisher:

- `RunnerOutboxBacklogHigh` — `outbox_unpublished_count > 10` for 10m
- `RunnerOutboxOldestTooHigh` — `outbox_oldest_unpublished_age_seconds > 600` for 5m

When these fire, follow these steps:

- __Terminal C: Inspect metrics quickly__
  ```bash
  # Outbox gauges
  curl -s http://localhost:8090/metrics | grep -E '^outbox_unpublished_count|^outbox_oldest_unpublished_age_seconds'

  # Publish counters
  curl -s http://localhost:8090/metrics | grep -E '^outbox_published_total|^outbox_publish_errors_total'
  ```

- __Terminal B: Check runner logs__
  - Look for Redis errors and outbox publisher messages (`outbox publisher started`, `outbox enqueue error`, `outbox mark published error`).

- __Terminal D: Verify dependencies__
  - Redis reachable? Use `redis-cli -u redis://localhost:6379 PING` or check compose logs.
  - Postgres healthy? Confirm connections and locks if needed.

- __Terminal B: Remediate__
  - Transient issues: restart the runner process/service.
  - Persistent `outbox_publish_errors_total`: investigate Redis connectivity/credentials or DB lock contention.

Notes:
- Gauges are updated by the outbox publisher loop; they may briefly lag when the system is idle.
- Counters help distinguish successful publishes vs errors over time.
