# Provider Region Constraints Plan

This document tracks the design and implementation plan for selecting providers by geographic region (US, EU, Asia) for LLM Bias Detection jobs on Golem.

## Objectives
- Ensure each bias-detection job runs across three regions: US, EU, Asia.
- Provide auditable provenance for chosen region per execution.
- Fail gracefully with clear user messaging when no providers match.

## Definitions
- Execution: one concrete run of a job on a specific provider.
- Claimed region: the region provided by the node (offer metadata/tags).
- Observed region: region inferred from provider endpoint’s GeoIP.

## Approach Overview
1. Discovery: publish demands; receive matching offers.
2. Negotiation: filter by constraints (CPU/mem/runtime) + region.
3. Verification: cross-check observed region; reject if mismatch.
4. Provenance: store both claimed and observed region in results and transparency logs.

## Constraints Model (MVP)
- Resources: docker runtime, network on, >=2 vCPU, >=4GB RAM.
- Region parameter: one of [US, EU, ASIA].
- Region matching priority:
  1) Offer property `region` or `tags` contains target region.
  2) If absent, accept tentatively, run preflight probe to get GeoIP; cancel if not in region.
- Timeouts: 45–60s to acquire a provider per region; after that, relax or mark partial.
- Budget caps: per-execution max price per minute and total cap.

## Data to Persist per Execution
- provider_id
- offer_id
- price terms
- region_claimed (from offer)
- region_observed (from GeoIP)
- region_verified (boolean and method)
- timestamps (queued, started, completed)
- image/version, model, question subset

## Backend Tasks
- Demand builder accepts `region` param (US/EU/ASIA) and resource caps.
- Offer filter:
  - Hard filter on explicit region tag/property when present.
  - Otherwise mark `needs_probe=true` and proceed.
- Preflight probe:
  - Tiny task to fetch provider’s public endpoint IP.
  - GeoIP lookup (offline DB or service) to infer country/region.
  - If mismatch: terminate agreement and continue searching (within timeout window).
- Execution metadata:
  - Attach claimed/observed/verified region to execution record and transparency append.

## Frontend Tasks
- Bias Detection submit uses fixed models/providers and selected questions.
- Show per-region rows (US/EU/ASIA): status, provider id (short), retries, ETA.
- Surface partial success (e.g., 2/3 regions complete) and retry failed.
- Link to Job Detail and Executions.

## API Endpoints (expected)
- POST `/api/v1/jobs` with spec:
  - `regions: ["US","EU","ASIA"]`
  - `questions: string[]` (from Questions page)
  - `models: string[]` (fixed for MVP)
  - `runs: 1`
- GET `/api/v1/jobs/:id` includes per-region executions and status.
- GET `/api/v1/executions?job_id=...` returns executions with region metadata.

## UI/UX Copy
- Questions page: "All questions selected by default. Unchecking reduces time and cost."
- Bias Detection page: show selected regions and note on regional verification (claimed + observed via GeoIP).
- World View: counts are synthetic until backend region metadata is fully enabled.

## Milestones
- M1: Spec + demand builder accepts region param (US/EU/ASIA).
- M2: Offer filtering by tag/property; fall back to preflight probe path.
- M3: Preflight GeoIP verification + metadata persisted.
- M4: Frontend progress per region; Executions include region data.
- M5: World View switches from synthetic to real region counts.

## Risks & Mitigations
- No explicit region in offers → probe path adds latency. Mitigate with caching and parallel search.
- Sparse regional coverage → timeout and partial success; clear UX and optional relax policy.
- GeoIP inaccuracies → store both claimed and observed, show verification status.

## Open Questions
- Which GeoIP source? (local DB vs. external API)
- Standardizing provider region tags to reduce probe dependence.
- Budget policy per region (uniform vs. region-specific caps).

## Tracking
- Owner: Backend runner
- Reviewers: Portal frontend, Transparency
- Status: Draft
