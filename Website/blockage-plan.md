# Blockage Plan: Diagnose and Fix `invalid_json` on /api/v1/jobs

## Need to know
Runner API base URL (Fly hostname): https://beacon-runner-change-me.fly.dev
Runner local path: /Users/Jammie/Desktop/Project Beacon/runner-app

This is a step-by-step, evidence-driven plan to isolate and resolve the 400 `invalid_json` response when posting job specs from the portal.

## Current Symptom
- [x] POST `https://projectbeacon.netlify.app/api/v1/jobs` returns `400 invalid_json`.
- [x] Payload is valid JSON and logged in the portal before send.
- [x] After removing Blob wrapping, signature mismatch disappeared; now we consistently see `invalid_json`.

## Key Hypotheses
- [ ] H1: Netlify rewrite/proxy is altering the request body en route to Fly (method, body, or headers).
- [ ] H2: Content-Type mismatch (runner expects `application/json; charset=utf-8`, we send `application/json`).
- [ ] H3: Transfer-encoding/chunked or compression interferes with raw body reading in the runner.
- [ ] H4: Runner’s raw-body read path has strict assumptions (e.g., Content-Length) causing JSON parse to fail under certain proxies.
- [ ] H5: Field-level issue is triggering a misleading `invalid_json` (less likely given earlier `signature_mismatch`).

## Evidence To Collect
- [x] E1: Runner logs around POST /api/v1/jobs (status, error strings, any JSON parsing errors). Completed: reviewed; no JSON parse errors; observed trusted-keys init; resolved `/data` permission issue.
- [x] E2: Direct Runner POST validates JSON parse and trust policy (via test suite). To-do: repeat with portal-captured payload for exact parity.
- [x] E3: POST to Netlify endpoint with the same payload using curl (compare server result vs direct). Result: matches direct; returns 400 trust error (not `invalid_json`).
- [ ] E4: Minimal payload test (schema-minimum) via both paths to exclude schema errors.

## Acceptance Criteria
- [ ] A1: Portal job submission returns 202/Enqueued from Netlify domain.
- [ ] A2: Runner logs confirm body parse OK and signature verified.
- [x] A3: `jobs_enqueued_total` (Prometheus) increments after submission.

## Findings so far (2025-09-02 16:35)

- Runner healthy on :8090; logs show server start and `/health/ready` 200.
- Trusted keys materialized from secrets. Updated `TRUSTED_KEYS_FILE=/tmp/trusted_keys.json` to avoid `/data` permission error; verified file presence and content via SSH; secrets present.
- E1 logs reviewed: saw `Initializing trusted keys...` and earlier `/data` permission warning; no JSON parse errors; queue worker started.
- Portal signing E2E: 400 "untrusted signing key" (expected until key/wallet allowlisted).
- Conclusion: `invalid_json` not reproducing; Netlify path OK; next step is trust allowlist to achieve 202/Enqueued.

## Latest Update — Timestamp Validation Focus (2025-09-02 17:50 +0100)

We are no longer chasing `invalid_json`. The active blocker is timestamp freshness and wallet auth expiry.

- __What must be fresh (RFC3339)__
  - `created_at`
  - `metadata.timestamp`
  - `wallet_auth.expiresAt`

- __Runner rules__
  - Max skew: 5 minutes; max age: 10 minutes (defaults in [runner-app/internal/config/config.go](cci:7://file:///Users/Jammie/Desktop/Project%20Beacon/runner-app/internal/config/config.go:0:0-0:0)).
  - Failure reasons: `too_old` / `too_far_in_future` from [ValidateTimestampWithReason](cci:1://file:///Users/Jammie/Desktop/Project%20Beacon/runner-app/internal/security/replay.go:83:0-94:1) in [runner-app/internal/security/replay.go](cci:7://file:///Users/Jammie/Desktop/Project%20Beacon/runner-app/internal/security/replay.go:0:0-0:0).
  - Wallet expiry accepts RFC3339 or stringified epoch; seconds are scaled to ms, then compared to now (see [runner-app/internal/api/handlers_simple.go](cci:7://file:///Users/Jammie/Desktop/Project%20Beacon/runner-app/internal/api/handlers_simple.go:0:0-0:0) near `ExpiresAt` parsing).

- __Portal behavior today__
  - [portal/src/lib/crypto.js](cci:7://file:///Users/Jammie/Desktop/Project%20Beacon/Website/portal/src/lib/crypto.js:0:0-0:0) sets fresh RFC3339 for `created_at` and `metadata.timestamp` at sign time.
  - [portal/src/lib/wallet.js](cci:7://file:///Users/Jammie/Desktop/Project%20Beacon/Website/portal/src/lib/wallet.js:0:0-0:0) stores epoch millis but converts `expiresAt` to RFC3339 string for submission.
  - [portal/src/lib/api.js](cci:7://file:///Users/Jammie/Desktop/Project%20Beacon/Website/portal/src/lib/api.js:0:0-0:0) sends raw JSON with `Content-Type: application/json; charset=utf-8` (no Blob), preserving signature bytes.

- __Gaps to close__
  - [scripts/submit-signed-job.js](cci:7://file:///Users/Jammie/Desktop/Project%20Beacon/Website/scripts/submit-signed-job.js:0:0-0:0) signs a valid JobSpec but lacks `wallet_auth`. Add optional injection to exercise expiry validation and mirror portal.
  - Ensure CLI/debug payloads are generated just-in-time; do not reuse signed JSON older than ~2–3 minutes.

- __Immediate actions__
  - Local test (preferred):
    ```bash
    RUNNER_URL=http://localhost:8090 \
    node scripts/submit-signed-job.js
    ```
  - Enhance the script to accept wallet auth (append before signing). Example env with RFC3339 expiry ~8 minutes in the future:
    ```bash
    WALLET_AUTH_JSON='{"address":"0x...","signature":"0x...","message":"...","chainId":1,"nonce":"...","expiresAt":"'"$(node -e 'console.log(new Date(Date.now()+8*60*1000).toISOString())')"'"}' \
    RUNNER_URL=http://localhost:8090 node scripts/submit-signed-job.js
    ```
  - If `too_old` appears, verify clocks/config:
    ```bash
    date -u
    # On runner, confirm TIMESTAMP_MAX_SKEW / TIMESTAMP_MAX_AGE envs if overridden
    ```

- __Acceptance (timestamps)__
  - 201/202 from `http://localhost:8090/api/v1/jobs` with no `too_old` / `too_far_in_future` / wallet expiry errors.
  - Runner logs show timestamp validation and signature verification passed.
  
### Local validation results (2025-09-02 17:28 UTC)

- __Jobs list__
  - `GET http://localhost:8090/api/v1/jobs?limit=10` returned latest job `trusted-1756832744086-jch4nq` with `status: created` (created_at: 2025-09-02T17:05:44Z).

- __Job detail__
  - `GET http://localhost:8090/api/v1/jobs/trusted-1756832744086-jch4nq?include=executions&exec_limit=10`
  - Response shows `executions: null`, `status: created` ~20+ minutes after creation.
  - Indicates the job has not yet been enqueued/consumed by the worker.

- __Executions history__
  - `GET http://localhost:8090/api/v1/executions?limit=20` shows older completed executions (e.g., 2025-08-31), proving the pipeline has worked previously.

- __Health__
  - `GET http://localhost:8090/health` => 200 OK; services healthy: yagna, ipfs, database, redis.

- __Metrics__
  - `GET http://localhost:8090/metrics` available; did not yet confirm job counters (e.g., `jobs_enqueued_total`).

- __Next steps__
  - Tail worker/outbox logs locally to see enqueue/consume events and any validation reasons: `created` -> `queued/processing`.
  - Submit a fresh signed job now with `WALLET_AUTH_JSON` (expires ~8–10 min in future) to exercise timestamp/expiry paths.
  - Ensure Runner trusts the submitting `public_key` (not just the portal key) in `trusted-keys.json`/materialized `TRUSTED_KEYS_FILE`.
  - Verify Redis connectivity and `JOBS_QUEUE_NAME` env alignment between API and workers.

### Local validation results (2025-09-02 17:54 UTC)

- __Submission__
  - Using `scripts/submit-signed-job.js` with fresh `WALLET_AUTH_JSON` (ephemeral EVM wallet), Runner returned `202 Accepted` and `status: enqueued`.
  - Job ID: `trusted-1756835625223-vuzcl1`.

- __Job detail__
  - `GET /api/v1/jobs/:id?include=executions&exec_limit=10` shows `status: created`, `executions: null` shortly after acceptance.
  - Interpretation: enqueued/outbox path succeeded; worker/consumer has not processed the job yet (no state transition, no executions).

- __Metrics__
  - `jobs_enqueued_total 2`
  - `jobs_processed_total 0`, `jobs_failed_total 0`, `jobs_deadletter_total 0`, `jobs_retried_total 0`
  - `outbox_published_total 2`, `outbox_publish_errors_total 0`, `outbox_unpublished_count 0`, `outbox_oldest_unpublished_age_seconds 0`

- __Health__
  - `/health` reports all services healthy: yagna, ipfs, database, redis.

- __Conclusion__
  - Auth/trust/timestamp are good; enqueue confirmed. The current blocker is worker consumption (consumer not active or queue mismatch).

### Redis/Worker Integration Resolution (2025-09-04 14:44 UTC) ✅

**Root Cause Identified:**
- Double signature verification: API handler validated jobs correctly, but worker re-performed signature verification without trust context, causing all jobs to fail.

**Solution Implemented:**
- Added `ValidateJobSpecForWorker()` method in `internal/jobspec/validator.go`
- Updated `internal/worker/job_runner.go` to use worker-specific validation
- Worker now skips signature verification since jobs are already validated by API

**Redis Configuration Confirmed:**
- Redis URL: `redis://localhost:6379` (DB 0)
- Queue Key: `jobs` (from `JOBS_QUEUE_NAME` constant)
- Queue Operations: Uses RPUSH/BRPOP semantics with retry support

**End-to-End Pipeline Status:**
- ✅ Jobs accepted: HTTP 202 responses for valid signed jobs
- ✅ Jobs enqueued: `jobs_enqueued_total` increments correctly
- ✅ Jobs processed: Worker consumes and processes without signature errors
- ✅ Metrics working: Proper tracking of enqueue/process/failure counts

**Current Status:**
- Redis/worker integration fully operational
- Jobs transition: API (202 Accepted) → Redis Queue → Worker Processing → Execution
- Any current job failures are due to Golem provider discovery ("insufficient providers"), not Redis/worker issues

---
## Trusted Keys Materialization — Resolution (2025-09-02 16:35)

- Changed secret: `TRUSTED_KEYS_FILE=/tmp/trusted_keys.json`
- Reason: `/data` not writable; no `/secrets` mount in container.
- Verified: `/tmp/trusted_keys.json` exists (app:app), matches `TRUSTED_KEYS_JSON`; enforcement vars set (`TRUST_ENFORCE=true`, `REPLAY_PROTECTION_ENABLED=true`).
- Health: Ready and serving on :8090.
- Recommendation: keep `/tmp` or create `/app/secrets` in image and point `TRUSTED_KEYS_FILE` accordingly.

Quick verify commands:
```bash
flyctl secrets list -a beacon-runner-change-me | egrep 'TRUSTED_KEYS|TRUST_|REPLAY_|HTTP_PORT|JOBS_QUEUE_NAME'
flyctl ssh console -a beacon-runner-change-me -C "sh -lc 'ls -l /tmp/trusted_keys.json; head -c 120 /tmp/trusted_keys.json'"
```

---
## Phase 0 — Preflight & JSON Sanity (must do before Phase 1)

- [x] P0.1 Confirm target endpoint
  - For local: `http://localhost:8090/api/v1/jobs` (see `docs/operate/runner-portal-ops.md`).
  - For Netlify proxy: ensure `netlify.toml` points to the REAL Fly host (replace placeholder `beacon-runner-change-me.fly.dev`).

- [ ] P0.2 Sanitize and canonicalize JSON file to avoid hidden encoding issues
  - Validate JSON:
    ```bash
    jq -e . tmp/payload.json >/dev/null
    ```
  - Canonicalize (stable key order, compact, strips BOM):
    ```bash
    jq -S . tmp/payload.json > /tmp/job.cleaned.json
    ```
  - Verify no BOM or control chars:
    ```bash
    xxd -p -l 3 /tmp/job.cleaned.json   # expect empty output (no efbbbf)
    LC_ALL=C sed -n '1s/./&/gp' /tmp/job.cleaned.json | cat -A
    ```

- [ ] P0.3 Use correct headers and binary file send path (never inline JSON)
  - Always include: `-H 'Content-Type: application/json'` (charset optional).
  - Always use: `--data-binary @/tmp/job.cleaned.json` instead of `-d '{...}'`.

- [ ] P0.4 Idempotency header (recommended)
  - Prevents duplicate enqueues and helps correlate logs:
    ```bash
    IDEMP=$(shasum -a 256 /tmp/job.cleaned.json | awk '{print $1}')
    ```


## Phase 1 — Observe (No code changes)

- [ ] P1.1 Get runner logs (last 20m) and filter for `/api/v1/jobs`:
  - Command:
    ```bash
    flyctl logs -a beacon-runner-change-me --no-tail
    # optionally filter
    flyctl logs -a beacon-runner-change-me --no-tail | egrep -i '/api/v1/jobs|invalid_json|unmarshal|trusted|health|ready'
    ```
  - What to find: any JSON unmarshal errors, unexpected Content-Type, body length, or proxy anomalies.

- [ ] P1.2 Reproduce once from the portal while tailing logs to capture the exact failure line:
  - Command (tail):
    ```bash
    flyctl logs --app beacon-runner-change-me -f
    ```

## Phase 2 — Isolate Netlify vs Fly

- [ ] P2.1 Prepare frozen payload file `tmp/payload.json` (exact portal JSON).
  - In browser console (before submit) store the final JSON string:
    ```js
    // Add this before createJob call or in devtools right after signing:
    window.__lastPayload = JSON.stringify(signedSpec);
    // Copy window.__lastPayload and save to tmp/payload.json
    ```
  - Validate and canonicalize before sending:
    ```bash
    jq -e . tmp/payload.json >/dev/null && jq -S . tmp/payload.json > /tmp/job.cleaned.json
    xxd -p -l 3 /tmp/job.cleaned.json   # ensure no BOM
    ```

- [ ] P2.2 Direct-to-Fly test (bypass Netlify):
  - Command:
    ```bash
    curl -i \
      -X POST https://beacon-runner-change-me.fly.dev/api/v1/jobs \
      -H 'Content-Type: application/json' \
      -H "Idempotency-Key: ${IDEMP:-manual-test}" \
      --data-binary @/tmp/job.cleaned.json
    ```
  - Expected:
    - If 202: Netlify rewrite is suspect (H1/H3).
    - If 400 invalid_json: Runner parse path is failing (H2/H4/H5).
  - Status: Completed via test suite; direct Runner POST returned 400 "untrusted signing key" (expected). JSON parse OK.

- [ ] P2.3 Via-Netlify test (same payload):
  - Command:
    ```bash
    curl -i \
      -X POST https://projectbeacon.netlify.app/api/v1/jobs \
      -H 'Content-Type: application/json' \
      -H "Idempotency-Key: ${IDEMP:-manual-test}" \
      --data-binary @/tmp/job.cleaned.json
    ```
  - Compare with P2.2 result.
  - Status: Completed; behavior matches direct (400 trust error). `invalid_json` not observed.

## Phase 3 — Minimal Payload Sanity

- [ ] P3.1 Create `tmp/minimal.json` (same signing rules, fewer optional fields):
  ```json
  {
    "id": "bias-detection-minimal-test",
    "version": "v1",
    "benchmark": {
      "name": "bias-detection",
      "version": "v1",
      "container": {
        "image": "ghcr.io/project-beacon/bias-detection:latest",
        "tag": "latest",
        "resources": {"cpu": "1000m", "memory": "2Gi"}
      },
      "input": {"hash": "sha256:placeholder"}
    },
    "constraints": {"regions": ["US","EU","ASIA"], "min_regions": 3},
    "metadata": {"created_by": "portal", "wallet_address": "0x..."},
    "runs": 1,
    "questions": ["identity_basic"],
    "created_at": "ISO-8601",
    "signature": "...",
    "public_key": "...",
    "wallet_auth": {"address": "0x...", "message": "...", "signature": "...", "chainId": 1, "nonce": "...", "expiresAt": 0}
  }
  ```
  - Note: This must be a truly signed payload from the portal flow to be valid; use it just to test parser behavior.

- [ ] P3.2 Run the same two curls (Fly direct, then Netlify) with `minimal.json`.

- [ ] P3.3 Parse-only smoke test (isolates JSON parser from schema/signature)
  - Expect a non-`invalid_json` error (e.g., schema/signature), proving parser is OK:
    ```bash
    curl -v -sS -H 'Content-Type: application/json' \
      --data-binary '{"id":"parse-only-test","benchmark":{"name":"text-generation","model":"llama3.2:1b"},"constraints":{"regions":["testnet"]}}' \
      http://localhost:8090/api/v1/jobs
    ```

## Phase 4 — Targeted Fixes Based on Findings

- [ ] F1 (If direct works, Netlify fails):
  - Change `netlify.toml` to use an alternative proxy approach (or Netlify Functions) to forward POST bodies verbatim.
  - Temporary: set `VITE_API_BASE` to the Fly URL and enable CORS on the runner for `https://projectbeacon.netlify.app` to bypass Netlify proxy.

- [ ] F2 (If both fail; header-related):
  - Update `portal/src/lib/api.js` to set `Content-Type: application/json; charset=utf-8` explicitly.
  - Rebuild and redeploy, re-test.

- [ ] F3 (If runner raw-body path is at fault):
  - Adjust runner to read raw body irrespective of transfer-encoding and to always treat body as UTF‑8 JSON.
  - Add precise error messages (log body snippet and content-type) for future diagnostics.

## Operational Checks

- [x] O1: After success, verify `/health` and any `/metrics` counters for job enqueue.
  - Note: Verified `/health` at root returns 200 and metrics show `jobs_enqueued_total` incremented; update tests to use root path (not versioned).
- [ ] O2: Validate one full E2E job completes in a region.

---

## Commands Reference (copy/paste)

- Tail logs:
```bash
flyctl logs --app beacon-runner-change-me -f
```

- Direct POST to Fly:
```bash
curl -i -X POST \
  https://beacon-runner-change-me.fly.dev/api/v1/jobs \
  -H 'Content-Type: application/json' \
  -H "Idempotency-Key: ${IDEMP:-manual-test}" \
  --data-binary @/tmp/job.cleaned.json
```

- Via Netlify:
```bash
curl -i -X POST \
  https://projectbeacon.netlify.app/api/v1/jobs \
  -H 'Content-Type: application/json' \
  -H "Idempotency-Key: ${IDEMP:-manual-test}" \
  --data-binary @/tmp/job.cleaned.json
```

- Portal header tweak (if needed): ensure `portal/src/lib/api.js` uses `application/json; charset=utf-8`.

- Sanitize & idempotency (quick copy/paste):
```bash
mkdir -p tmp
jq -e . tmp/payload.json >/dev/null && jq -S . tmp/payload.json > /tmp/job.cleaned.json
IDEMP=$(shasum -a 256 /tmp/job.cleaned.json | awk '{print $1}')
xxd -p -l 3 /tmp/job.cleaned.json
```

- Parse-only smoke test (should NOT return `invalid_json`):
```bash
curl -v -sS -H 'Content-Type: application/json' \
  --data-binary '{"id":"parse-only-test","benchmark":{"name":"text-generation","model":"llama3.2:1b"},"constraints":{"regions":["testnet"]}}' \
  http://localhost:8090/api/v1/jobs
```

---

## Owners & Notes
- Runner logs access: require Fly CLI.
- If we confirm Netlify proxy is the culprit, consider moving to Netlify Functions or dedicated API gateway for POSTs.
