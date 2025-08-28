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

## Current Blockers

1. **API endpoints returning empty** - need to debug why `/health` is blank
2. **Job processing untested** - need to verify complete flow works
3. **Admin functionality unclear** - `/admin/config` needs investigation
4. **Missing LLM benchmark containers** - no actual containers with LLM runtimes built
   - Need Dockerfile with Ollama/transformers + Llama 3.2 model
   - Need benchmark script that runs "Who are you?" inference
   - Need to build and push to container registry
   - Current container references are placeholder strings only

5. **Provider region constraints incomplete** — implement selection and verification for US/EU/ASIA providers
   - See `constraints-plan.md` for approach and milestones
   - Demand builder accepts `region` param and resource caps
   - Offer filtering by region tag/property; fallback preflight GeoIP verification
   - Persist region metadata (claimed/observed/verified) and expose via API

### Provider Region Constraints — Checklist

- [ ] M1: Demand builder accepts `region` param (US/EU/ASIA) and resource caps
- [ ] M2: Offer filtering by explicit region tag/property when present
- [ ] M3: Preflight GeoIP verification for offers without region; reject mismatches
- [ ] M4: Persist region metadata (claimed/observed/verified) on executions and expose via API
- [ ] M5: Frontend shows per-region progress on Bias Detection page
- [ ] M6: World View switches from synthetic counts to backend-provided region data

See `constraints-plan.md` for details and owners.

## MVP Success Definition

✅ **Ready for MVP when:**
- External users can submit jobs
- Jobs get processed and return receipts
- Service doesn't crash under normal use
- Basic security (signature verification) works

---

*Focus: Get it working, then make it better*
