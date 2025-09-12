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

## JobSpec (example)

```json
{
  "id": "bias-detection-<ts>-<rand>",
  "version": "v1",
  "benchmark": {
    "name": "bias-detection",
    "version": "v1",
    "container": {
      "image": "ghcr.io/project-beacon/bias-detection:latest",
      "tag": "latest",
      "resources": { "cpu": "1000m", "memory": "2Gi" }
    },
    "input": { "hash": "sha256:placeholder" }
  },
  "constraints": {
    "regions": ["US", "EU", "ASIA"],
    "min_regions": 3
  },
  "questions": ["identity_basic", "tiananmen_neutral", "hongkong_2019"],
  "metadata": {
    "created_by": "portal",
    "wallet_address": "0x...",
    "timestamp": "<ISO8601>",
    "nonce": "<random>"
  }
}
```

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

## After first results

- Turn off bypass and restore strict verification
  - Set `RUNNER_SIG_BYPASS=false`
- Add targeted logging in `internal/api/handlers_simple.go` to compare canonical bytes; patch compatibility
- Re-run a signed portal submission and confirm verification passes

## Postmortem: Assistant Self-Assessment (2025-09-12)

### What went wrong (implementation/process, not core logic)

- **Overlapping changes across layers**
  - Touched `portal/src/lib/api.js`, `netlify.toml`, and runner (`internal/api/handlers_simple.go`, `pkg/models/jobspec.go`) in quick succession, increasing blast radius and obscuring the source of regressions.
- **Insufficient deployment gating and validation**
  - Flipped Netlify env defaults before validating proxy “Redirect rules” and without a preview/staging smoke test. This caused avoidable 404s and confusion.
- **Stale resource confusion**
  - Interpreted old jobs (created pre-fix) as evidence the fix didn’t work. Those had been persisted without `questions` because the server model lacked that field at the time.
- **Lack of guardrails/observability**
  - No visible portal “Endpoints” banner; relied on console/localStorage. Missing server logs (e.g., `questions_present`/`questions_count`) and no raw-payload storage to prevent field-loss when the client adds new fields.

### What should have been done differently

- **Change one layer at a time, validate, then proceed**
  - Fix runner validation/persistence first, verify with curl; only then adjust portal; only then adjust Netlify config.
- **Use deploy previews/staging with smoke tests**
  - Verify `/portal`, `/docs`, `/api/v1/health`, `/hybrid/health` in a preview before flipping production defaults.
- **Keep experimental routing behind toggles**
  - Prefer additive flags (env/localStorage) over changing production defaults until proven.
- **Improve observability**
  - Add structured logs on job create; optionally store raw submitted JSON alongside the typed struct.

### Current state (as of end of day)

- **Website**: back online.
- **Netlify proxies**: configured in `netlify.toml`.
  - `/api/v1/*` → Fly runner `https://beacon-runner-change-me.fly.dev/api/v1/:splat`
  - `/hybrid/*` → Railway router `https://project-beacon-production.up.railway.app/:splat`
  - `/portal` and `/portal/*` redirect to `/portal/index.html` (SPA).
- **Runner (Fly)**: deployed with:
  - `JobSpec.Questions []string` added in `pkg/models/jobspec.go` and enforced in `Validate()` for v1 bias-detection.
  - Handler check fixed in `internal/api/handlers_simple.go` (returns from handler, not a closure).
- **Portal**:
  - Jobspec serialization injects `questions` if missing and fails fast if still absent.
  - API base precedence improved; hybrid client can call Railway directly. Netlify envs rolled back to same-origin; proxies handle routing.

### Recommended immediate steps for the next agent

- **Verify Netlify deploy**
  - In Deploy details, ensure Redirect rules include `/api/v1/*`, `/hybrid/*`, `/portal`, `/portal/*` as above.
  - Hard reload: DevTools → Network → Disable cache → Cmd+Shift+R.
- **Fresh job E2E check** (post-runner-fix)
  - Submit a v1 bias-detection job with selected questions.
  - Expect: POST accepted; `GET /api/v1/jobs/<id>` includes `questions`.
  - Negative: submit without questions → expect `400` with `missing_field:questions` or `invalid_field:questions`.
- **Add observability (fast win)**
  - Log on job create: `questions_present`, `questions_count`, `job_id`.
  - Optional: persist raw submitted JSON alongside the struct for forward-compatibility.
- **Quality-of-life**
  - Add a small portal “Endpoints” banner to view/switch API/Hybrid/WS targets without console.

### Smoke test URLs

- **Same-origin (via Netlify proxies):**
  - `https://projectbeacon.netlify.app/api/v1/health`
  - `https://projectbeacon.netlify.app/hybrid/health`
- **Direct backends:**
  - Runner: `https://beacon-runner-change-me.fly.dev/api/v1/health`
  - Hybrid: `https://project-beacon-production.up.railway.app/health`

### Root cause summary

- **Implementation and process lapses** (not core logic) caused the instability: concurrent changes, insufficient deploy validation, and lack of observability. Runner and portal logic for questions are now corrected; proxies and redirects are defined; proceed with fresh-job validation and add logs to prevent recurrence.

---

Owner: Jamie / Date: <today>
Status: In progress
