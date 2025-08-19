# Project Beacon Runner – QA Test Plan

Last updated: 2025-08-20
Owner: QA Agent (you + me)
Scope: `runner-app/`
Status: ✅ **COMPLETED** - All test files implemented and QA plan objectives achieved

## Goals
- Verify core flows end-to-end: API -> Queue -> Worker -> Golem -> Store -> IPFS -> Transparency.
- Catch regressions quickly via layered tests (unit, integration, e2e/smoke).
- Align with MVP success criteria (multi-region execution, receipts, diffs, permanence).

## Conventions
- Port: 8090 for API server (http://localhost:8090).
- Terminal mapping:
  - Terminal A: Yagna daemon
  - Terminal B: Go API server
  - Terminal C: Actions (curl, tests)
  - Terminal D: Postgres + Redis (docker compose)

## Test Levels
- Unit: fast, isolated with fakes/mocks.
- Integration: DB/Redis/sqlmock, in-memory queue, mock IPFS/Golem.
- E2E: real Postgres/Redis (docker), API server, background workers, mock Golem adapter, optional real IPFS pin disabled.

## Current Tests (inventory)
- Models and signing: `pkg/models/jobspec_test.go` (validation, signing, JSON), receipts.
- Integration sanity: `internal/integration/*.go` (JobSpec/Receipt/Diff JSON, config load/validate).
- Golem module (mocked): `internal/golem/*_test.go` (service, executor, multi-region, provider filters, cost, receipts, concur exec, error paths).
- Circuit breaker: `internal/circuitbreaker/circuit_breaker_test.go`.
- IPFS repo: `internal/store/ipfs_repo_test.go` (UpdateExecutionCID, GetExecutionsByJobSpecID).

## Gaps & New Test Design
- API layer (missing):
  - POST /api/jobs (submit jobspec)
  - GET /api/jobs/:id (status)
  - GET /api/jobs/:id/executions (history)
  - GET /healthz, /readyz
  - Error paths: invalid JSON, validation errors, 404s
  - Middleware: auth (if any), rate-limit, recovery, logging
- Queue/Worker integration:
  - Enqueue -> JobRunner consumes -> ExecuteSingleRegion -> persist execution & receipt -> outbox row -> OutboxPublisher publishes
  - Retry/backoff & DLQ semantics; stale job recovery
- Persistence:
  - JobsRepo, ExecutionsRepo, OutboxRepo (sqlmock + migration fixtures)
  - Idempotency (same jobspec twice), unique constraints, pagination
- IPFS integration:
  - Mock client for add/pin -> CID stored via `IPFSRepo.UpdateExecutionCID`
  - Failure paths: pin failure, network timeout
- Cross-region diff engine:
  - Deterministic diffs for known inputs; thresholds & classification; JSON roundtrip
- Config & middleware:
  - `internal/api/middleware/*`: auth required header, rate limits, circuit breaker integration
  - Config override precedence (env > file > defaults)
- Observability:
  - Metrics exposed (Prometheus handlers) and critical counters increment on success/error
  - Structured request/trace IDs propagate
- External client protections:
  - `internal/external/protected_client` timeouts/retries

## Coverage Matrix (high level)
- API handlers: happy-path + error-paths + auth + rate-limit
- Queue semantics: LPUSH/BRPOP, retries, DLQ, visibility timeout (if applicable)
- Worker: single-region exec path, persist receipt, outbox publish
- Golem: provider discovery filters, timeouts, partial failure, cost estimate
- Store: jobs/executions CRUD, joins, pagination, indexing
- IPFS: add/pin, CID persistence, failure handling
- Diff: text/struct diffs, thresholding, classification
- Config/middleware: load/validate, recovery, circuit breaker
- Observability: metrics/labels, health/ready endpoints

## Proposed Test Files (stubs to add)
- [x] internal/api/handlers_jobs_test.go
- [x] internal/api/handlers_health_test.go
- [x] internal/api/middleware/auth_test.go (security/validation tests exist)
- [x] internal/queue/redis_queue_integration_test.go
- [x] internal/queue/integration_test.go (comprehensive Redis integration tests)
- [x] internal/worker/job_runner_integration_test.go
- [x] internal/worker/outbox_publisher_test.go
- [x] internal/store/jobs_repo_test.go
- [x] internal/store/executions_repo_test.go
- [x] internal/ipfs/client_test.go (mock)
- [x] internal/diff/engine_test.go
- [x] internal/config/config_test.go (override precedence)
- [x] internal/external/protected_client_test.go (already exists – extend failures)

## Targeted Coverage Improvements (Backlog)
- WebSocket hub (`internal/websocket/hub.go`): cover `NewHub`, `Run`, `Broadcast*`, `ServeWS`, `readPump`/`writePump` using `httptest.NewServer` and a Gorilla WS client.
- Worker & Outbox (`internal/worker/*`): cover `NewJobRunner*`, `Start`, `handleEnvelope`, outbox publisher helpers; use fake queue + stub repos; assert metrics and state transitions.
- Transparency proof generator (`internal/transparency/proof_generator.go`): unit test `GenerateProof` against a tiny tree and verify consistency with `pkg/merkle`.
- Crypto helpers (`pkg/crypto/ed25519.go`): round-trip sign/verify on small structs; base64 encode/decode helpers.
- Model validators (`pkg/models/validator.go`): table-driven tests for `Validate*`, hash/compute helpers, and sanitize paths.

## Optional CI Quality Gates
- Coverage threshold (example):
  - Terminal C:
    ```bash
    go test ./... -coverprofile=cover.out && go tool cover -func=cover.out
    ```
  - CI: fail if total coverage < desired threshold (e.g., 35–40% initially).
- Race detector:
  - Terminal C:
    ```bash
    go test ./... -race -count=1
    ```
- Flake sweep (smoke):
  - Terminal C:
    ```bash
    for i in {1..5}; do go test ./... -count=1 || exit 1; done
    ```

## E2E Smoke Scenarios
1) Submit-and-complete single-region job
- Pre: Postgres+Redis up (Terminal D), API on :8090 (B), yagna mock/disabled
- Act: POST /api/jobs with valid jobspec
- Assert: 202 + job_id; poll GET /api/jobs/:id until completed; GET executions returns >=1; receipts signed; metrics increment

2) Multi-region job with partial failure
- Constraints require 3 regions; simulate 1 region timeout
- Assert: job completes if >= MinRegions success; diff artifact created; classification set

3) IPFS publish on completion
- On completed execution, mock IPFS returns CID; verify CID persisted and visible in GET executions

4) Retry & DLQ
- Force transient error -> retry with backoff; after N attempts, move to DLQ; assert DLQ entry

5) Protection & limits
- Missing auth -> 401; exceeded rate -> 429; healthz readyz 200

## Environment Setup (E2E)
- Terminal D: `docker compose -f docker-compose.yml up -d postgres redis`
- Terminal B: `HTTP_PORT=8090 DATABASE_URL=... REDIS_URL=... go run ./cmd/runner`
- Terminal C: `go test ./...` or curl calls for smoke tests

## Acceptance Criteria per Flow
- Submit job: 2xx on valid, 4xx on invalid; persisted Job row
- Execute region: execution row + receipt JSON with signature verified
- Outbox: row emitted then published to Redis queue; ack on success
- IPFS: CID stored or error escalated; no silent drops
- Diff: scores and classification deterministic for controlled inputs

## Work Plan & Tracking
- ✅ High priority (COMPLETED)
  - ✅ API handler tests (submit/status/executions/health)
  - ✅ Queue->Worker integration (retry/DLQ)
  - ✅ E2E smoke test 1 and 3 (submit-and-complete, IPFS persist)
- ✅ Medium (COMPLETED)
  - ✅ Golem failure modes; persistence repos; protected client
- ✅ Low (COMPLETED)
  - ✅ Diff engine scenarios; full observability assertions

## Milestones Checklist
- [x] API: handlers basic happy-path
- [x] API: error-paths + middleware
- [x] Queue/Worker: happy-path + retry + DLQ
- [x] Store: repos CRUD + pagination
- [x] IPFS: mock client + CID persistence
- [x] Golem: filters, partial failure, cost + timeouts
- [x] E2E: submit-and-complete job
- [x] E2E: IPFS publish on completion
- [x] E2E: retry & DLQ
- [x] Observability: metrics + healthz/readyz
- [x] Transparency: root/proof/bundle endpoints smoke-tested
- [x] Protections: rate limiting 429 behavior verified

## Notes
- Keep tests hermetic by default; gate network-dependent tests with build tags or env flags.
- Use sqlmock for DB unit tests; for e2e prefer real Postgres/Redis with docker compose.
- For port 8090, ensure tests and docs reference `http://localhost:8090` consistently.
