# Friday Plan: Minimal Q&A Results

Objective: get end-to-end results flowing today using the Hybrid Router + Runner, with a basic Q&A JobSpec, progress monitoring, and a temporary signature bypass. We’ll fix signature canonicalization after we capture first results.

## Tasks (Today)

- [ ] Portal: include `questions` array in JobSpec serialization
- [ ] Submit test job with 2–3 questions; region order set to preferred first (MVP executes first region only)
- [ ] Monitor job progress from portal and API until first execution completes
- [ ] Keep runner signature bypass enabled for today to unblock results
- [ ] Validate receipt: provider, response text, timings
- [ ] Document quick commands for monitoring and troubleshooting
- [ ] (Optional) Ensure worker stays active: keep 1 machine running to avoid autostop
- [ ] Fix Google Maps API key

## Portal work

- Ensure form serialization includes `questions`:
  - When submitting, gather selected question IDs and set `jobspec.questions = [ ... ]`.
  - Confirm in network payload (DevTools) that `questions` is present.

## Monitoring

- Portal UI
  - Jobs → select job → Live Progress → Refresh until first region completes
  - Click execution row to view result output and metadata

- Runner API
  - Status only:
    ```bash
    curl -sSfL "https://beacon-runner-change-me.fly.dev/api/v1/jobs/<JOB_ID>" | jq .
    ```
  - Include latest receipt (when available):
    ```bash
    curl -sSfL "https://beacon-runner-change-me.fly.dev/api/v1/jobs/<JOB_ID>?include=latest" | jq .
    ```
  - Executions list:
    ```bash
    curl -sSfL "https://beacon-runner-change-me.fly.dev/api/v1/jobs/<JOB_ID>?include=executions&exec_limit=5" | jq .
    ```
  - Watch shortcuts:
    ```bash
    watch -n 5 'curl -s "https://beacon-runner-change-me.fly.dev/api/v1/jobs/<JOB_ID>" | jq -r .status'
    watch -n 5 'curl -s "https://beacon-runner-change-me.fly.dev/api/v1/jobs/<JOB_ID>?include=latest" | jq -r ".executions[0] | {provider_used, success, created_at, completed_at}"'
    ```

## Runner settings (today)

- Signature bypass is enabled to unblock results:
  - `RUNNER_SIG_BYPASS=true`
- Trust allowlist remains enforced (portal key in allowlist).
- After results are captured, we will add canonicalization debug logs and then re-enable strict signature verification.

## Worker/Queue sanity

- Ensure env and active worker
  - `HYBRID_BASE` is set to the Railway router
  - `REDIS_URL` configured
  - `JOBS_QUEUE_NAME` (default `jobs`)
  - Keep at least one machine running to avoid autostop during processing

## Troubleshooting quick checks

- Recent jobs:
  ```bash
  curl -sSfL "https://beacon-runner-change-me.fly.dev/api/v1/jobs?limit=10" | jq .
  ```
- Logs (filter common lines):
  ```bash
  flyctl logs -a beacon-runner-change-me --no-tail | grep -E "job enqueued|idempotent|no latest receipt|CreateJob service error" | tail -n 100
  ```

## Progress Update (End of Day - 2025-09-12)

### ✅ Completed Today
- **Fixed production runner validation**: Added `JobSpecID` field mapping for `jobspec_id` → `id` compatibility
- **Updated validation middleware**: Now handles portal's `jobspec_id` field format correctly
- **Created receipt viewer**: Deployed comprehensive HTML viewer at `/receipt-viewer.html`
- **Added executions API**: Real database integration for viewing execution receipts
- **Local runner**: Fully functional with end-to-end job processing and receipt generation
- **Identified production issues**: Root cause analysis of stuck jobs and API errors

### ⚠️ Outstanding Issues (Critical for Monday)
1. **Production jobs stuck in "created" status**
   - Jobs created before the fix aren't published to outbox queue
   - Need manual republish script or database fix
   - Test job: `bias-detection-1757700139114-vzks0e` still stuck

2. **Production executions API returns 500 error**
   - Database connection or handler issue on production
   - Local version works correctly
   - Affects receipt viewing functionality

3. **Job ID field mapping inconsistency**
   - New jobs show empty `"id": ""` in API responses
   - Field mapping between `jobspec_id` and `id` needs verification

### 🎯 Next Steps (Monday Priority)
1. **Fix stuck jobs**: Create admin endpoint or script to republish old jobs to queue
2. **Debug production executions API**: Check database connection and error logs
3. **Verify end-to-end flow**: Submit fresh job and confirm processing to completion
4. **Test receipt viewer**: Ensure it works with production API once fixed

### Smoke test URLs

- **Same-origin (via Netlify proxies):**
  - `https://projectbeacon.netlify.app/api/v1/health`
  - `https://projectbeacon.netlify.app/hybrid/health`
  - `https://projectbeacon.netlify.app/receipt-viewer.html`
- **Direct backends:**
  - Runner: `https://beacon-runner-change-me.fly.dev/api/v1/health`
  - Hybrid: `https://project-beacon-production.up.railway.app/health`

### Test Commands for Monday
```bash
# Check job status
curl -s "https://beacon-runner-change-me.fly.dev/api/v1/jobs/bias-detection-1757700139114-vzks0e" | jq '.status'

# Test executions API (currently 500)
curl -s "https://beacon-runner-change-me.fly.dev/api/v1/executions" | jq .

# Submit new test job
curl -X POST "https://beacon-runner-change-me.fly.dev/api/v1/jobs" \
  -H "Content-Type: application/json" \
  -d '{"jobspec_id":"test-monday","version":"v1","benchmark":{"name":"bias-detection","container":{"image":"ghcr.io/project-beacon/bias-detection:latest","resources":{"cpu":"1000m","memory":"2Gi"}},"input":{"hash":"sha256:placeholder"}},"constraints":{"regions":["US"],"min_regions":1},"questions":["Who are you?"],"signature":"test","public_key":"test"}'
```

---

Owner: Jamie / Date: 2025-09-12
Status: Blocked on production issues - ready for Monday resolution
