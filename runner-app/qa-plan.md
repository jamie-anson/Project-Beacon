# Project Beacon Runner – QA & Coverage Plan

Last updated: 2025-08-21
Owner: QA Agent (you + me)
Scope: `runner-app/`
Status: IN PROGRESS

## Objectives
- Ensure core flows are correct and stable: API -> Queue -> Worker -> Golem -> Store -> IPFS -> Transparency.
- Prevent regressions through layered tests (unit, integration, e2e/smoke) with CI enforcement.
- Achieve and sustain coverage targets with per-package thresholds.

## Coverage Targets
- Overall repository: 85%+ within 2 sprints, 90% long-term.
- Core packages (correctness/security critical): 90–95%
  - `pkg/models/`, `pkg/crypto/`, `pkg/merkle/`, `internal/worker/`, `internal/service/`, `internal/jobspec/`
- Supporting libraries: 80–85%
  - `internal/diff/`, `internal/transparency/`, `internal/golem/`
- Adapters/glue: 70–75%
  - `internal/api/`, `internal/websocket/`, `internal/metrics/`

### Per-package explicit targets (overrides)
- `internal/metrics/`: ≥ 90% (stability-critical for observability and regression detection)

## Phased Plan
1) Baseline & CI gates (Week 1)
   - Generate coverage profile and HTML report:
     - `go test ./... -coverpkg=./... -coverprofile=coverage.out -count=1`
     - `go tool cover -html=coverage.out -o coverage.html`
   - CI coverage gates:
    - Current CI gates (as of 2025-08-21): Overall ≥ 51%, Core ≥ 86%, Non-core ≥ 53% (temporary to keep CI green; will ratchet up).
    - Next bump target (by 2025-08-28): Overall ≥ 55%, Core ≥ 90%, Non-core ≥ 60%.
    - Source of truth: thresholds live in `.github/workflows/test-and-coverage.yml` (env: `OVERALL_MIN`, `CORE_MIN`, `NONCORE_MIN`). Keep this plan and workflow in sync.
   - Always run race detector in CI: `go test -race ./...`.

   - Temporary CI thresholds (ramping this sprint):
    - Today (2025-08-21): Overall ≥ 51%, Core ≥ 86%, Non-core ≥ 53% [temporary]
    - Next week (by 2025-08-28): Overall ≥ 55%, Core ≥ 90%, Non-core ≥ 60%
    - Weeks 3–4: Overall ≥ 70–80%, Core ≥ 92–94%, Non-core ≥ 65–75%
    - Post-sprint steady state: Overall ≥ 85%, Core ≥ 95%, Non-core ≥ 80%

2) Targeted unit tests (Weeks 1–2)
   - `internal/worker/`:
     - `job_runner.go`: bad envelope, invalid jobspec, no regions, execute failure, persistence failure.
     - `outbox_publisher.go`: retry/backoff, DLQ path, JSON wrap errors.
   - `internal/service/`:
     - Validation errors, repository failures, idempotency/transactional boundaries.
   - `pkg/models/`: maintain >90% by covering any new branches.
   - `internal/jobspec/`: parsing/normalization and malformed inputs.

3) Integration & property tests (Weeks 2–3)
   - End-to-end without external deps: enqueue → JobRunner execute single region → persist receipt → outbox publish.
   - Property/Fuzz tests:
     - `pkg/crypto`: signature round-trips, malformed inputs.
     - `pkg/merkle`: tree building/verification for randomized inputs and edge cases.

4) Deflaking & sustainability (Weeks 3–4)
   - Stabilize flaky tests (prefer events over strict sleeps; deterministic fakes like in-memory queue or miniredis).
   - Enforce per-PR coverage on new/changed lines.
   - Publish coverage artifact (`coverage.html`) and PR comment; add badge.

## Current Test Inventory
- Models and signing: `pkg/models/jobspec_test.go` (validation, signing, JSON, receipts).
- Integration sanity: `internal/integration/*.go` (JobSpec/Receipt/Diff JSON, config load/validate).
- Golem module (mocked): `internal/golem/*_test.go` (service, executor, multi-region, provider filters, cost, receipts, concurrent exec, error paths).
- Circuit breaker: `internal/circuitbreaker/circuit_breaker_test.go`.
- IPFS repo: `internal/store/ipfs_repo_test.go`.

## Coverage Matrix (What to cover)
- API handlers: happy-path + error-paths + auth + rate-limit.
- Queue semantics: LPUSH/BRPOP, retries, DLQ, stale recovery.
- Worker: single-region exec path, persist receipt, outbox publish.
- Golem: provider filters, timeouts, partial failure, cost estimate.
- Store: jobs/executions CRUD, joins, pagination, indexing.
- IPFS: add/pin, CID persistence, failure handling.
- Diff: text/struct diffs, thresholding, classification.
- Config/middleware: load/validate, recovery, circuit breaker.
- Observability: metrics/labels, health/ready endpoints.

## Metrics Testing Checklist
- __Reset registry in tests__: Call `resetProm()` from `internal/metrics/collector_more_test.go` at the start of any test that resets or relies on the Prometheus default registry. This helper sets a fresh registry and calls `RegisterAll()` to re-register all collectors.
- __Centralized registration__: Ensure `internal/metrics/metrics.go` `init()` calls `RegisterAll()`. Tests must never duplicate registration lists; always prefer `RegisterAll()`.
- __Gin middleware tests__: Call `resetProm()` before constructing a Gin router that uses `GinMiddleware()` so metrics like `HTTPRequestsTotal` and `HTTPRequestDuration` exist in the default registry.
- __Custom registries__: If a test needs a custom `prometheus.Registry`, set both `prometheus.DefaultRegisterer` and `prometheus.DefaultGatherer` to it, then call `RegisterAll()`.
- __Assertions__: Validate presence of `http_requests_total` and label cardinality for dynamic/static paths, methods, and status codes; ensure histograms observe durations.

## Prioritized Backlog (Tests to add)
- Worker & Outbox (`internal/worker/*`): `NewJobRunner*`, `Start`, `handleEnvelope`, enqueueWithRetry; fake queue + stub repos; assert metrics/state.
- Service (`internal/service/jobs.go`): invalid inputs, repo failures, idempotency.
- Transparency (`internal/transparency/proof_generator.go`): `GenerateProof` vs `pkg/merkle` on tiny tree.
- Crypto (`pkg/crypto/ed25519.go`): sign/verify round-trips; base64 decode errors; fuzz inputs.
- Merkle (`pkg/merkle`): property tests for order, duplicates, empty leaves.

## CI Quality Gates
- Coverage gate:
  - `go test -race -coverpkg=./... -coverprofile=coverage.out ./...`
  - Parse `go tool cover -func=coverage.out` to enforce thresholds (overall + per-package).
  - Upload `coverage.html` as artifact; comment summary on PR; badge on README.
- Flake sweep (optional):
  ```bash
  for i in {1..5}; do go test ./... -count=1 || exit 1; done
  ```

## E2E Smoke Scenarios
1) Submit-and-complete single-region job
   - Pre: Postgres+Redis up, API on :8090, yagna mock/disabled
   - Act: POST /api/jobs with valid jobspec
   - Assert: 202 + job_id; poll GET /api/jobs/:id until completed; executions >=1; receipts signed; metrics increment

2) Multi-region with partial failure
   - Constraints require 3 regions; simulate 1 region timeout
   - Assert: completes if >= MinRegions success; diff artifact; classification set

3) IPFS publish on completion
   - Mock IPFS returns CID; verify persisted CID in GET executions

4) Retry & DLQ
   - Force transient error -> retry with backoff; after N attempts, DLQ entry asserted

5) Protection & limits
   - Missing auth -> 401; exceeded rate -> 429; healthz/readyz 200

## Environment Setup (E2E)
- `docker compose -f docker-compose.yml up -d postgres redis`
- `HTTP_PORT=8090 DATABASE_URL=... REDIS_URL=... go run ./cmd/runner`
- `go test ./...` or curl for smoke tests

## Acceptance Criteria per Flow
- Submit job: 2xx on valid, 4xx on invalid; persisted Job row
- Execute region: execution row + signed receipt; signature verifies
- Outbox: row emitted then published; ack on success
- IPFS: CID stored or error escalated; no silent drops
- Diff: scores/classification deterministic for controlled inputs

## Milestones
- Week 1: CI gates at 50% overall; core 85%; add critical worker/service unit tests.
- Week 2: Raise to 60% overall; core 90%; add E2E job flow tests.
- Weeks 3–4: Property/fuzz tests; raise to 75–80% overall; core 92–95%; deflake and finalize.

## Notes
- Prefer hermetic tests; gate network-dependent tests via build tags/env.
- Use `sqlmock` for DB unit tests; use real Postgres/Redis for e2e.
- Standardize on `http://localhost:8090` for docs/tests.
