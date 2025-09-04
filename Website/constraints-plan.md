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

## Decisions (MVP)
- [x] GeoIP source: Primary MaxMind GeoLite2 City (local DB); fallback ipinfo/ip-api.
- [x] Provider tag standard: prefer `beacon.region` in {US, EU, ASIA}; accept `region`, `geo.region`, or tags array.
- [x] Budget policy: uniform caps for all regions in MVP; revisit per-region later.
- [x] Timeout/relax policy: 60s strict window, then allow offers without explicit region tag but require preflight probe; optional second window +30s.
- [x] Hard constraints: runtime=docker, vCPU>=2, RAM>=4GiB, outbound network required, price caps enforced.

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

## Backend Tasks (Checklists)
- [x] DemandBuilder accepts `region` param and resource/price caps.
- [x] OfferFilter classifies offers: explicit match vs `needs_probe`.
- [x] PreflightProbe: fetch egress IP, local GeoIP lookup, map to region.
- [x] Negotiator: strict 60s window → relax policy (+30s) → partial success.
- [x] Persistence: save claimed/observed/verified region, evidence reference.
- [x] Telemetry: offers_seen/matched/probed, probes_passed/failed, timings.
- [x] Unit tests: DemandBuilder, OfferFilter, GeoIP mapping.
- [x] Integration tests: negotiation → probe → execution → persistence.

## Frontend Tasks (Checklists)
- [x] Bias Detection submit: fixed models + selected questions + regions.
- [x] Per-region rows (US/EU/ASIA): status, provider id (short), retries, ETA, verification badge.
- [x] Partial success UI: 2/3 complete, “Retry missing region”.
- [x] Link to Job Detail and Executions with region metadata.

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

## Milestones (Checkable)
- [x] M1: Spec + DemandBuilder accepts region param (US/EU/ASIA) and caps.
- [x] M2: Offer filtering by tag/property; fallback to preflight `needs_probe` path.
- [x] M3: Preflight GeoIP verification wired; metadata persisted and exposed.
- [x] M4: Frontend per-region progress; Executions include region data.
- [x] M5: World View switches from synthetic to real region counts.

## Risks & Mitigations
- No explicit region in offers → probe path adds latency. Mitigate with caching and parallel search.
- Sparse regional coverage → timeout and partial success; clear UX and optional relax policy.
- GeoIP inaccuracies → store both claimed and observed, show verification status.

## Open Questions
- Per-region pricing caps (enable when we have market data).
- Architecture/OS constraints (do we need to strictly require x86_64?).
- Evidence permanence: whether to pin preflight evidence to IPFS by default.

## Tracking
- Owner: Backend runner
- Reviewers: Portal frontend, Transparency
- Status: Completed
- Last Updated: 2025-08-29
