# Outbox Publisher Failure Plan

## Problem Statement
- **Symptom**: `TestOutboxPublisher_EnqueueFailure_DoesNotMarkPublished` fails with unmet metrics-query expectation and Redis connection errors during test teardown.
- **Related Production Risk**: Jobs observed stuck in `created` state even after auto-restart fix. Need to confirm whether outbox processing is functionally broken or test-only.

## Objectives
- **Restore** the failing unit/integration test to a stable, deterministic state.
- **Validate** that the outbox publisher processes real entries end-to-end (DB → Redis → runner).
- **Document** any functional bugs uncovered during investigation.

## Current Signals
- Test cancels context before `updateOutboxMetrics()` runs, so metrics query expectation is never met.
- Real runtime logs show repeated Redis connection failures and `failed to get outbox stats` errors after context cancellation.
- Historical bug reports indicate jobs remain in `created` status.

## Diagnostics
1. **Reproduce deterministically**
   - Run `go test ./internal/worker -run TestOutboxPublisher_EnqueueFailure_DoesNotMarkPublished -v` with `-count=1`.
   - Capture test log timestamps vs expectation registration.
   - Repeat with `-race` to surface hidden data races around cancellation.
2. **Instrument publisher loop**
   - Add temporary logging (or use `t.Logf`) inside `updateOutboxMetrics()` and retry branches to record when they fire relative to context cancellation.
   - Ensure instrumentation is removed once flake source confirmed.
3. **Check real outbox state**
   - Using Terminal C, run SQL: `SELECT id, topic, published_at FROM outbox ORDER BY id DESC LIMIT 20;`
   - Confirm whether new jobs insert outbox rows and whether `published_at` is NULL for stuck jobs.
4. **Verify Redis connectivity**
   - Ensure Terminal D `docker compose` Redis is running; connect via `redis-cli -p 6379 PING`.
   - For production equivalence, confirm Runner uses Upstash (Terminal E) or local and note any auth differences.
5. **Queue schema sanity check**
   - Inspect `internal/constants/queue.go` to ensure publisher targets `constants.JobsQueueName` and matches consumer expectations.

## Hypotheses
- **H1 (Test-only)**: Expectation should allow context cancellation before metrics query. Fix by removing or relaxing metrics expectation.
- **H2 (Functional)**: Publisher aborts following Redis failure; no retry/backoff ensures eventual success. Need to inspect `enqueueWithRetry()` behavior when Redis unavailable.
- **H3 (Upstream)**: Outbox rows missing due to job creation transaction rollback; publisher idle incorrectly.
- **H4 (Config drift)**: Queue name or Redis URL mismatch between test and production causing enqueue attempts to target wrong endpoint.

## Action Plan

### Phase 1 — Stabilize tests
- [ ] Re-run failing test with `-race` and capture logs.
- [ ] Adjust test harness to wait explicitly for either metrics call or enqueue failure (e.g., sync channel) before canceling context.
- [ ] If metrics query occurs only on idle loop, replace strict expectation with `ExpectQuery(...).Maybe()` or remove entirely and assert via fetch + mark only.
- [ ] Normalize cleanup so Redis server is closed after publisher stops to avoid mid-test disconnect noise.

### Phase 2 — Validate functional pipeline
- [ ] Instrument `enqueueWithRetry()` to log final error type (temporary) while verifying behavior against real Redis.
- [ ] Run integration scenario: insert fake outbox row via SQL (Terminal C), start publisher (Terminal C), confirm message appears in Redis queue (`LRANGE jobs 0 -1`).
- [ ] Submit real job via `/api/v1/jobs` (Terminal C) and confirm `executions` transition to running/completed.
- [ ] Review metrics emitted to ensure `OutboxPublishedTotal` increments and error counter stabilizes.

### Phase 3 — Hardening & follow-up
- [ ] If Redis outages still break flow, design fallback/alerting (e.g., `OutboxRetryQueue`, Sentry alert on sustained failure).
- [ ] Update runbook with mitigation steps and monitoring signals.

## Verification
- **Tests**: `go test ./internal/worker -run OutboxPublisher -count=1`
- **Integration**: Submit real job via `/api/v1/jobs`; confirm `executions` created.
- **Monitoring**: Validate metrics counters (`OutboxPublishedTotal`, `OutboxPublishErrorsTotal`).

## Decision Log & Rollback
- Document changes in `docs/runbooks/outbox.md` (create if missing).
- If regression found, disable publisher auto-start in main until fixed (config flag) and document toggle location in `cmd/runner/main.go`.
