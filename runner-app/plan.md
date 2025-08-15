# Runner App — Phase 1 Execution Plan

Scope: Backend runner that ingests signed JobSpecs, executes benchmarks on Golem across multiple regions, generates signed Receipts, stores bundles to IPFS, appends transparency entries, and exposes APIs/UI for diffs.

Alignment: Mirrors Website plan (`Website/plan.md`) and detailed program doc (`Website/runner-app-plan.md`). Incorporates Permanence principle: cryptographic signing, on-chain anchoring, and decentralized storage.

---

## Architecture (Runner App)
- JobSpec Handler: schema + Ed25519 signature verification, queuing.
- Golem Execution Engine: provider selection, single → multi-region runs, retries/timeouts.
- Receipt System: signed execution receipts (outputs + metadata).
- Diff Module: cross-region comparison, similarity score, diff snippet.
- Storage & Transparency: bundle to IPFS (CID), append to transparency log, plan for anchoring.
- APIs: REST endpoints for job lifecycle, results, diffs, verification.

Tech stack: Go + Gin, PostgreSQL, Redis, IPFS, Docker; frontend later: Next.js for dashboard.

---

## Milestones (8 weeks)
1) Week 1–2: Foundation & single-region execution with signed receipts.
2) Week 3–4: Multi-region execution + aggregation.
3) Week 5–6: Diff engine + JSON API.
4) Week 7–8: IPFS + transparency log + dashboard MVP.

Success criteria (Phase 1):
- Execute benchmark across ≥3 regions; signed receipts; IPFS bundles with CID; transparency log entries; diff visualization.

---

## Week 1–2: Foundation & Single-Region (Actionable)

Checklist
- [ ] Project scaffolding
  - [ ] Initialize Go module, Gin skeleton: `cmd/runner`, `internal/`, `pkg/`
  - [ ] Config loader (env + file), structured logging
  - [ ] Docker dev setup; Makefile targets (dev, test, lint)
- [ ] Persistence & queue
  - [ ] PostgreSQL schema: `jobs`, `executions`, `receipts`
  - [ ] Redis connection + basic enqueue/dequeue
  - [ ] Migrations (e.g., golang-migrate)
- [ ] JobSpec
  - [ ] Define JSON schema (v0) and protobuf/Go structs
  - [ ] Ed25519 signature verification (libsodium/go stdlib)
  - [ ] Validation unit tests
- [ ] Golem integration (single region)
  - [ ] SDK wiring; provider discovery filtered by region
  - [ ] Minimal benchmark container ("Who are you?"), harness stub
  - [ ] Execute single run; capture stdout/stderr/exit
- [ ] Receipt v0
  - [ ] Receipt schema (job_id, region, output_hash, timestamps, provider_meta)
  - [ ] Sign receipt; store in Postgres
  - [ ] Expose "submit job" and "get result" endpoints
- [ ] Observability
  - [ ] Healthcheck endpoint, basic metrics (latency, errors, queue depth)

Deliverable: Single-region execution producing a signed Receipt, persisted and retrievable via API.

---

## Week 3–4: Multi-Region
- Fan-out to ≥3 regions; timeouts/retries; aggregate receipts; job status lifecycle; priority queue.

## Week 5–6: Diff Engine
- Similarity scoring, diff snippet extraction; Diff JSON schema; REST endpoints; tests.

## Week 7–8: Storage, Transparency, Dashboard
- Bundle receipts + outputs + metadata; IPFS pin → CID; transparency log append; anchoring plan; Next.js dashboard MVP.

---

## APIs (initial draft)
- POST `/v1/jobs` → create from signed JobSpec
- GET `/v1/jobs/{id}` → status
- GET `/v1/jobs/{id}/executions` → per-region receipts
- GET `/v1/jobs/{id}/result` → output + receipt
- GET `/v1/diffs?job_id=...` (Week 5–6)
- GET `/v1/transparency/{cid}` (Week 7–8)

---

## Data Schemas (draft v0)
- jobs: id, created_at, spec_hash, requester_pubkey
- executions: id, job_id, region, provider_id, output_hash, stdout_cid, stderr_cid, started_at, finished_at, status
- receipts: id, execution_id, payload_json, signature, signer_pubkey
- diffs (later): id, job_id, region, output_hash, similarity_score, diff_snippet_json

---

## Permanence & Transparency
- Sign: JobSpec and Receipt with Ed25519.
- Store: outputs + receipts bundled and pinned to IPFS; record CID.
- Anchor: transparency-log entry for each run; plan on-chain attestation (v1).

---

## Refactor Opportunities & Reliability Improvements

Purpose: Reduce technical debt and improve reliability of Postgres and Redis integrations.

Checklist (incremental, PR-sized):

- [ ] Repository/service layers
  - [ ] Move SQL to a repository `internal/store/jobs_repo.go` (UpsertJob, GetJob, ListJobs)
  - [ ] Add service `internal/service/jobs.go` for `CreateJob(ctx, spec)` (validate → repo → queue/outbox)
  - [ ] Pass `context.Context` with timeouts to all DB calls

- [ ] Postgres improvements
  - [ ] Switch to `pgx/pgxpool` with pool config
  - [ ] Adopt `golang-migrate` and versioned migrations (replace ad-hoc creation)
  - [ ] Add indices: `status`, `created_at`; consider GIN on `jobspec_data`
  - [ ] Constrain `status` (enum or check constraint)

- [ ] Atomic DB + queue (Outbox pattern)
  - [ ] Add `outbox` table and transactionally write along with job row
  - [ ] Background publisher reads outbox, enqueues to Redis, marks sent
  - [ ] Metrics and alerts for stuck outbox items

- [ ] Redis queue reliability
  - [ ] Standardize list semantics (LPUSH/BRPOP or RPOPLPUSH processing list + ack)
  - [ ] Retry with backoff and move to dead-letter `jobs:dead` after N attempts
  - [ ] Keep payload as small envelope `{id, enqueued_at, attempt}` (fetch spec from Postgres)
  - [ ] Extract queue names/constants (e.g., `queue.JobsQueue`)

- [ ] Observability & health
  - [ ] Structured logging (job_id, attempt, region)
  - [ ] Prometheus metrics: enqueued, processed, failed, retries, queue depth
  - [ ] `/api/v1/health` includes DB ping and Redis ping

- [ ] Tooling & config
  - [ ] Update Dockerfile Go version to match `go.mod` (e.g., `golang:1.24-alpine`)
  - [ ] Centralize config in `internal/config` (DB, Redis, timeouts, queue names)
  - [ ] Ensure `.env` parity with `docker-compose.yml` via `${...}` interpolation

---

## Today’s Starter Tasks
- [ ] Create repo scaffolding with Gin service and Makefile
- [ ] Add migrations for `jobs`, `executions`, `receipts`
- [ ] Implement JobSpec schema + signature verify
- [ ] Wire Redis queue; stub worker to call Golem SDK
- [ ] Add POST `/v1/jobs` and GET `/v1/jobs/{id}`

If you want, I’ll start scaffolding the service now (dirs, main.go, config, migrations, Makefile).
