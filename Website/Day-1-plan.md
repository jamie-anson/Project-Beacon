# Day-1 Golem Live Test Plan (MVP)

Simple, high-signal plan for our first live testing session on the Golem network.

## Objectives
- Validate end-to-end flow in production: submit → enqueue → execute → receipt → transparency.
- Exercise admin RBAC and runtime config via `/auth/whoami` and `/admin/config`.
- Prove basic reliability: app stays healthy, queue drains, and persistence is stable.
- Capture timings, errors, and receipts for a short report.

## Prerequisites
- Runner deployed and reachable over HTTPS.
- Environment configured on the runner:
  - `ADMIN_TOKENS`, `OPERATOR_TOKENS`, `VIEWER_TOKENS` (comma-separated tokens)
  - `TRUSTED_KEYS` (allows verifying signed JobSpecs)
  - `DATABASE_URL` (Neon Postgres)
  - `REDIS_URL` (Upstash Redis)
  - `IPFS_URL` (gateway), `YAGNA_URL` (Golem)
  - `GIN_MODE=release`
- Have one admin token and one operator token available locally:
  - `export ADMIN_TOKEN='<one-admin-token>'`
  - `export OPERATOR_TOKEN='<one-operator-token>'`
- Credits/funds and a working yagna daemon for providers on the target region(s).
- Optional: jobspec signer built from runner repo (`cmd/jobspec-signer`).

## Schedule (UTC)
- T-60m: Owners meet, verify deployment health.
- T-30m: RBAC sanity checks, admin config read.
- T-15m: Submit 1 validation (expected-400) request to confirm error paths.
- T-0m: Submit 1 signed, minimal job; observe enqueue and execution.
- T+15m: Submit 2–3 jobs (US/EU/ASIA) if capacity available.
- T+45m: Verify receipts and transparency endpoints; capture artifacts.
- T+60m: Review, summarize, and decide next steps.

## Smoke Tests (Runner)
- Health:
  - `curl -sSf https://<runner>/health | jq`
  - `curl -sSf https://<runner>/health/ready | jq`
- RBAC role discovery:
  - `curl -sSf -H "Authorization: Bearer $ADMIN_TOKEN" https://<runner>/auth/whoami | jq`
- Admin config (read):
  - `curl -sSf -H "Authorization: Bearer $OPERATOR_TOKEN" https://<runner>/admin/config | jq`
- Admin config (update; admin only):
  - `curl -sSf -X PUT https://<runner>/admin/config \
      -H "Authorization: Bearer $ADMIN_TOKEN" \
      -H 'Content-Type: application/json' \
      -d '{
            "ipfs_gateway": "https://w3s.link",
            "features": { "ws_live_updates": false },
            "constraints": { "default_region": "EU", "max_cost": 2.5 }
          }' | jq`

## Job Submission
- Expect-400 validation (unsigned):
  - `curl -sS -X POST https://<runner>/api/v1/jobs \
      -H 'Content-Type: application/json' \
      -d '{"id":"job-invalid-unsigned","version":"1","benchmark":"noop","container":{"image":"noop"}}' | jq`
- Sign a JobSpec (recommended):
  1) Generate or load keypair (runner repo binary):
  - `./jobspec-signer generate-keypair -o dev-key.json`
  2) Sign JobSpec (example path below):
  - `./jobspec-signer sign -k dev-key.json -i ./jobspec.json -o ./jobspec.signed.json`
  3) Ensure the public key is in `TRUSTED_KEYS` on the runner.
- Submit signed JobSpec (202 expected):
  - `curl -sS -X POST https://<runner>/api/v1/jobs \
      -H 'Content-Type: application/json' \
      -H 'Idempotency-Key: day1-001' \
      -d @./jobspec.signed.json | tee /tmp/day1-submit.json | jq`
- Poll list/status:
  - `curl -sS 'https://<runner>/api/v1/jobs?limit=5' | jq`
  - `JOB_ID='<from submit response>'`
  - `curl -sS "https://<runner>/api/v1/jobs/$JOB_ID?include=executions" | jq`

## Optional: Use Provided JobSpecs
- See repo: `Website/llm-benchmark/jobspecs/`
  - `llama-bias-detection-unified.json`
  - `mistral-bias-detection-unified.json`
  - Sign before submit to avoid 400 signature errors.

## Transparency Checks
- `curl -sS https://<runner>/api/v1/transparency/root | jq`
- `curl -sS 'https://<runner>/api/v1/transparency/proof?index=0' | jq`
- If a bundle CID is returned in receipts, fetch:
  - `curl -sS https://<runner>/api/v1/transparency/bundles/<cid> | jq`

## Monitoring & Logs
- Watch runner logs for errors and latency spikes.
- Observe Redis queue depth and dead-letter queue (if any).
- Confirm Postgres writes for executions and receipts.
- Health endpoints should remain 200; rate limits at `/api/v1/jobs` should behave as expected under low volume.

## Success Criteria
- At least one signed job enqueued and executed per target region (as capacity allows).
- Receipts retrievable via API; transparency endpoints respond with valid structures.
- No 5xx due to app faults; predictable 4xx on invalid input.

## Risk & Mitigation
- Provider scarcity → reduce concurrency; retry later windows.
- Signature or trust misconfig → verify `TRUSTED_KEYS` and re-sign with known key.
- Queue/persistence errors → pause submissions; inspect Redis/Postgres connectivity.

## Rollback Plan
- Halt submissions; set rate limits to minimal.
- Revert `/admin/config` to safe defaults.
- Roll back deployment to previous stable image.

## Owners & Roles
- **Run lead:** coordinates timeline and decisions.
- **Infra:** monitors runner, DB, Redis, networking.
- **Ops:** manages tokens, admin config, and emergency changes.

---

Capture all commands, responses, latencies, and job IDs into a Day-1 log for retrospective.
