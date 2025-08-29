# Project Beacon MVP Launch Readiness

Simple checklist to get Beacon Runner live for initial network participants.

## Infrastructure ✅ (Complete)
- [x] Fly.io deployment working
- [x] Database connected (Neon Postgres)
- [x] Queue connected (Upstash Redis)
- [x] TLS/HTTPS working
- [x] Environment secrets configured

## MVP Core Requirements

### Basic Functionality
- [x] Service responds to requests (health endpoint working)
- [x] Job submission endpoint works (/api/v1/jobs responds with validation)
- [x] Job processing completes successfully (job enqueued and retrievable)
- [x] Receipts generated correctly (job status API working)

### Essential Security
- [x] Signature verification working (accepts valid signed jobs)
- [x] Trusted keys loaded from environment (dev-2025-q3 key active)
- [x] Basic input validation (helpful error messages)

### Minimal Monitoring
- [x] Service stays running (auto-scaling working)
- [x] Basic health check works (200 OK externally)
- [x] Can see if jobs are processing (job status API)

## MVP Launch Criteria

**Must Have:**
- [x] End-to-end job flow works (submit → enqueue → retrieve)
- [x] Service accessible from external network
- [x] Basic error handling (doesn't crash on bad input)

**Nice to Have (post-MVP):**
- Grafana Cloud metrics
- Load testing
- Comprehensive monitoring
 - Advanced security hardening
 - Performance optimization
 - GPU optimisation

## Recently Resolved Blockers 

- API endpoints returning empty — fixed. `/health` responds 200 OK with payload.
- Job processing untested — verified end-to-end (submit → enqueue → retrieve → receipt).
- Missing LLM benchmark containers — delivered (Llama 3.2-1B, Qwen 2.5-1.5B, Mistral 7B) with Ollama integration and local validation.
- Provider region constraints — completed (M1–M6 implemented: region param, offer filtering, GeoIP preflight, metadata persistence, frontend progress, World View data).
- Admin functionality defined and RBAC implemented — `/auth/whoami` and `/admin/config` (GET/PUT) secured via Authorization: Bearer tokens from `ADMIN_TOKENS`/`OPERATOR_TOKENS`; `/admin/port` and `/admin/hints` public only in debug mode.

## Current Blockers

- None at this time.

### Provider Region Constraints — Checklist

- [x] M1: Demand builder accepts `region` param (US/EU/ASIA) and resource caps
- [x] M2: Offer filtering by explicit region tag/property when present
- [x] M3: Preflight GeoIP verification for offers without region; reject mismatches
- [x] M4: Persist region metadata (claimed/observed/verified) on executions and expose via API
- [x] M5: Frontend shows per-region progress on Bias Detection page
- [x] M6: World View switches from synthetic counts to backend-provided region data

## MVP Success Definition

 Ready for MVP when: 
- External users can submit jobs
- Jobs get processed and return receipts
- Service doesn't crash under normal use
- Basic security (signature verification) works

---

*Focus: Get it working, then make it better*
