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
- [x] Job submission endpoint works (/api/v1/jobs responds)
- [ ] Job processing completes successfully (need valid signed job)
- [ ] Receipts generated correctly (need valid signed job)

### Essential Security
- [x] Signature verification working (rejects invalid payloads)
- [ ] Trusted keys loaded from environment (need to verify)
- [x] Basic input validation (400 for invalid requests)

### Minimal Monitoring
- [ ] Service stays running
- [ ] Basic health check works
- [ ] Can see if jobs are processing

## MVP Launch Criteria

**Must Have:**
- [ ] End-to-end job flow works (submit → process → receipt)
- [ ] Service accessible from external network
- [ ] Basic error handling (doesn't crash on bad input)

**Nice to Have (post-MVP):**
- Grafana Cloud metrics
- Load testing
- Comprehensive monitoring
- Advanced security hardening
- Performance optimization

## Current Blockers

1. **API endpoints returning empty** - need to debug why `/health` is blank
2. **Job processing untested** - need to verify complete flow works
3. **Admin functionality unclear** - `/admin/config` needs investigation

## MVP Success Definition

✅ **Ready for MVP when:**
- External users can submit jobs
- Jobs get processed and return receipts
- Service doesn't crash under normal use
- Basic security (signature verification) works

---

*Focus: Get it working, then make it better*
