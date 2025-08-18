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

## Week 1–2: Foundation & Single-Region ✅ COMPLETE

Checklist
- [x] Project scaffolding
  - [x] Initialize Go module, Gin skeleton: `cmd/runner`, `internal/`, `pkg/`
  - [x] Config loader (env + file), structured logging
  - [x] Docker dev setup; Makefile targets (dev, test, lint)
- [x] Persistence & queue
  - [x] PostgreSQL schema: `jobs`, `executions`, `receipts`
  - [x] Redis connection + basic enqueue/dequeue
  - [x] Migrations (e.g., golang-migrate)
- [x] JobSpec
  - [x] Define JSON schema (v0) and protobuf/Go structs
  - [x] Ed25519 signature verification (libsodium/go stdlib)
  - [x] Validation unit tests
- [x] Golem integration (single region)
  - [x] SDK wiring; provider discovery filtered by region
  - [x] Minimal benchmark container ("Who are you?"), harness stub
  - [x] Execute single run; capture stdout/stderr/exit
- [x] Receipt v0
  - [x] Receipt schema (job_id, region, output_hash, timestamps, provider_meta)
  - [x] Sign receipt; store in Postgres
  - [x] Expose "submit job" and "get result" endpoints
- [x] Observability
  - [x] Healthcheck endpoint, basic metrics (latency, errors, queue depth)

✅ **Deliverable ACHIEVED**: Single-region execution producing a signed Receipt, persisted and retrievable via API.

**BONUS COMPLETED**: Multi-region execution, cross-region diff analysis, Prometheus metrics, Grafana dashboards, React frontend with WebSocket real-time updates.

---

## Week 3–4: Multi-Region ✅ COMPLETE

Checklist
- [x] Multi-region execution engine
  - [x] Fan-out to ≥3 regions (US, EU, APAC)
  - [x] Concurrent execution with context timeouts
  - [x] Provider discovery filtered by region
  - [x] Error handling and graceful degradation
- [x] Receipt aggregation
  - [x] Collect receipts from all successful regions
  - [x] Cross-region execution summary
  - [x] Cryptographic signing of aggregated results
- [x] Job lifecycle management
  - [x] Job status tracking (pending, running, completed, failed)
  - [x] Execution persistence in PostgreSQL
  - [x] Real-time status updates via WebSocket
- [x] Queue processing
  - [x] Redis-based job queue with worker processes
  - [x] Background job execution with outbox pattern
  - [x] Retry logic and dead letter queue handling

✅ **Deliverable ACHIEVED**: Multi-region execution with aggregated receipts and job lifecycle management.

**BONUS COMPLETED**: Real-time WebSocket updates, comprehensive observability, production-ready error handling.

## Week 5–6: Diff Engine ✅ COMPLETE

Checklist
- [x] Cross-region diff analysis
  - [x] Output comparison across all execution regions
  - [x] Automated difference detection and flagging
  - [x] Structured diff result storage in PostgreSQL
- [x] Diff JSON schema and API
  - [x] Diff result schema with metadata (regions, timestamps, summary)
  - [x] REST endpoints: `GET /api/v1/diffs`, `POST /api/v1/diffs/analyze`
  - [x] Diff details with region-specific output breakdown
- [x] Frontend visualization
  - [x] Diff viewer component with syntax highlighting
  - [x] Visual indicators for identical vs different outputs
  - [x] Detailed diff breakdown by region
  - [x] Export functionality for diff analysis reports
- [x] Testing and validation
  - [x] Unit tests for diff detection logic
  - [x] Integration tests for multi-region comparison
  - [x] End-to-end testing with real execution outputs

✅ **Deliverable ACHIEVED**: Cross-region diff analysis with JSON API and visualization.

**BONUS COMPLETED**: Real-time diff notifications, export functionality, comprehensive UI for diff exploration.

## Week 7–8: Storage, Transparency, Dashboard 🚧 PARTIALLY COMPLETE

Checklist
- [x] Dashboard MVP
  - [x] React frontend with modern UI/UX
  - [x] Job management interface (create, execute, monitor)
  - [x] Real-time execution monitoring with WebSocket
  - [x] Cross-region diff visualization
  - [x] System health and metrics dashboard
  - [x] Responsive design with TailwindCSS
- [ ] IPFS integration
  - [x] Bundle receipts + outputs + metadata into IPFS objects
  - [x] Pin bundles to IPFS → generate Content IDs (CIDs)
  - [x] IPFS gateway integration for retrieval
  - [ ] CID storage in PostgreSQL for permanent reference (schema + repo added; API wiring pending)
- [ ] Transparency log
  - [ ] Append execution records to immutable transparency log
  - [ ] Merkle tree structure for tamper-evident history
  - [ ] Public verification endpoints
  - [ ] Log anchoring strategy (blockchain/timestamping)
- [ ] Advanced observability
  - [ ] Enhanced Grafana dashboards with IPFS metrics
  - [ ] Transparency log monitoring and alerting
  - [ ] Long-term storage analytics

🚧 **Deliverable IN PROGRESS**: Dashboard complete, IPFS and transparency log pending.

**COMPLETED**: Full-featured React dashboard with real-time updates, comprehensive job management, and diff visualization.

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
