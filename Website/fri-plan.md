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

---

Owner: Jamie / Date: <today>
Status: In progress
