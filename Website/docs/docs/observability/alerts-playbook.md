---
id: alerts-playbook
title: Common Alerts Playbook
sidebar_label: Alerts Playbook
---

Actionable guidance for the alerts defined in `runner-app/observability/prometheus/alerts.yaml`.

Conventions below use your terminal labels:
- Terminal A: Yagna daemon
- Terminal B: Go API server
- Terminal C: Actions (curl, tests)
- Terminal D: Postgres + Redis (docker compose)

## RunnerJobsDeadLetter (critical)
Jobs moved to dead-letter in the last 5m.

- Checks
  - Terminal D: `docker compose logs runner | tail -n 200` (look for worker errors)
  - Terminal B: validate JobSpec schema handling
  - Terminal D: Postgres healthy? `docker compose ps postgres`
- Likely causes
  - Persistent processing errors, invalid JobSpec, DB write issues
- Actions
  - Re-run a small test job; inspect worker stack traces
  - Verify DB constraints and repo methods

## RunnerJobsRetriesHigh (warning)
`jobs_retried_total` > 0.5/s over 5m.

- Checks
  - Terminal D: `docker compose logs runner`
  - Terminal C: API health `curl -i http://localhost:8090/healthz`
- Likely causes
  - Transient DB/Redis/IPFS issues; backpressure
- Actions
  - Inspect DB/Redis health; increase backoff if needed; fix root cause

## RunnerOutboxPublishErrors (warning)
Errors publishing outbox messages to Redis.

- Checks
  - Terminal D: `docker compose logs runner` (look for outbox publisher errors)
  - Redis reachable? `redis-cli -h localhost -p 6379 PING`
- Likely causes
  - Redis unavailable or auth mismatch
- Actions
  - Restore Redis; verify `REDIS_URL`; ensure network connectivity

## RunnerHTTP5xxHigh (warning)
5xx > 5% over 10m.

- Checks
  - Terminal C: `curl -i http://localhost:8090/healthz`
  - Terminal D: `docker compose logs runner`
- Likely causes
  - Handler panics, DB timeouts, dependency failures
- Actions
  - Inspect API logs/stack traces; add circuit breakers or tighten timeouts

## RunnerHTTPLatencyHigh (warning)
p95 > 500ms for 10m.

- Checks
  - Terminal C: `hey -z 30s http://localhost:8090/` (or similar load)
  - Terminal D: Postgres/Redis metrics (exporters recommended)
- Likely causes
  - DB/Redis slow, CPU saturation, large payloads
- Actions
  - Add indexing/caching; optimize hot paths; scale workers

## GolemExecutionFailureRateHigh (warning)
Golem failures >10% over 5m.

- Checks
  - Terminal A: Yagna status and provider availability
  - Terminal D: `docker compose logs runner` (golem execution errors)
- Likely causes
  - Provider issues, network instability, wallet problems
- Actions
  - Retry with different providers/regions; validate payload and budget

## GolemWalletBalanceLow (critical)
GLM balance below 0.1.

- Checks
  - Terminal A: Inspect wallet balance
- Actions
  - Fund wallet; set up alert routing for on-call

## GolemExecutionDurationHigh (warning)
p95 duration > 5m by region.

- Checks
  - Prometheus graph for `golem_execution_duration_seconds_bucket` by `region`
- Likely causes
  - Slow providers or large tasks
- Actions
  - Adjust provider filters; cap duration/budget; parallelize work

## GolemProvidersUnavailable (critical)
No providers available in region for 5m.

- Checks
  - Terminal A: Yagna network health; connectivity
- Actions
  - Fail over to another region; reduce requirements; investigate network

## Operations runbook

- Silence during maintenance
  - Terminal C: Alertmanager UI http://localhost:9093 → Silence
- Verify Prometheus targets
  - http://localhost:9090/targets (runner, yagna must be UP)
- Reload rules
  - `curl -X POST http://localhost:9090/-/reload`

## References
- Alerts config: `runner-app/observability/prometheus/alerts.yaml`
- Alertmanager: `runner-app/observability/alertmanager/alertmanager.yml`
- Prometheus: `runner-app/observability/prometheus/prometheus.yml`
