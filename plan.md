# Project Beacon — High-Level Build Plan

## Principles
- Integrity by design: benchmark-agnostic, open participation.
- Transparency and verifiability: public, tamper-evident logs.
- Permanence: benchmark results, methods, and key metadata are cryptographically signed and anchored to a public ledger; bundles are stored on decentralized storage (e.g., Golem + IPFS) for long-term availability.
- Neutrality: platform ensures integrity and safety review; does not select or editorialize content.

## Timeline Overview
- Phase 0 — Prep & Partner Seeding (2–4 weeks)
- Phase 1 — MVP Execution + Cross-Region Diff (6–8 weeks)
- Phase 2 — Attestation & Difference Verification (6–8 weeks)
- Phase 3 — Open Benchmark Registry (6–10 weeks)
- Phase 4 — Assurance Tiers & Stronger Geo Proof (8–12 weeks)
- Phase 5 — Federation & Political Safety Net (Ongoing)

---

## Phase 0 — Prep & Partner Seeding (2–4 weeks)
Goal: Secure early allies and scope clarity.

Scope
- One-pager: benchmark-agnostic platform, open participation, Golem-backed permanence.
- Identify 3–5 early attesters (universities, civic tech, open AI labs).
- Pick tech stack for:
  - JobSpec & Receipt signing (Ed25519 + JSON).
  - Storage (Golem + IPFS).
  - Transparency log (Merkle-based, Sigstore/Rekor-style).
- Define MVP example benchmark ("Who are you?" prompt).
- NEW: Define cross-region diff output format (JSON schema + visual presentation rules).

Deliverables
- Vision one-pager.
- Signed JobSpec/Receipt draft schema.
- Draft transparency-log data model and anchoring plan.
- MVP benchmark definition + harness stub.
- Cross-region diff JSON schema v0 and UI presentation rules.

Exit Criteria
- Consensus on tech stack and schemas.
- 3–5 attesters verbally committed.

---

## Phase 1 — MVP Execution + Cross-Region Diff (6–8 weeks)
Goal: Prove benchmark → Golem execution → permanent storage → difference detection works.

Scope
- Runner app: pulls signed JobSpec, runs harness, produces Receipt.
- Store bundle in IPFS/Golem; get CID.
- Append CID to transparency log.
- Run MVP benchmark in ≥3 countries via Golem node constraints.
- NEW: Automatic diff module:
  - Compare outputs from all regions for the same JobSpec.
  - Output JSON: {country, output_hash, similarity_score, diff_snippet}.
  - Basic web UI to highlight changed words/sentences.

Deliverables
- End-to-end run across multiple countries.
- CID-based permanent storage of bundles and results.
- Transparency log with verifiable entries.
- Public dashboard showing cross-region diffs.

Exit Criteria
- Repeatable end-to-end flow with reproducible logs and CIDs.
- Diff JSON and UI pass basic reliability checks.

---

## Phase 2 — Attestation & Difference Verification (6–8 weeks)
Goal: Independent verification of both integrity and differences.

Scope
- Attesters pull results from transparency log, verify:
  - Signatures valid.
  - Inputs identical across all countries.
  - Model weights identical.
- Attesters sign an AttestationReport including:
  - Confirmation of diff validity.
  - Optional deterministic re-run.
- Dashboard labels:
  - “Confirmed Difference” vs “Suspected Harness Artefact”.

Deliverables
- Attester client tooling + report schema.
- Dashboard integration of attestation states.

Exit Criteria
- ≥2 independent attesters confirm at least one diff.

---

## Phase 3 — Open Benchmark Registry (6–10 weeks)
Goal: Anyone can submit benchmarks (including politically sensitive ones).

Scope
- Submission: container + dataset/prompt hash + scoring function.
- Coordinator schedules runs in multiple regions.
- Cross-region diff runs automatically on all multi-country jobs.
- Governance:
  - No editorial control over content, only format/safety review.
  - Public methodology statement.

Deliverables
- Registry service + submission review workflow.
- Coordinator for multi-region scheduling.

Exit Criteria
- First community-submitted benchmark runs end-to-end with diffs and logs.

---

## Phase 4 — Assurance Tiers & Stronger Geo Proof (8–12 weeks)
Goal: Make location claims highly credible.

Scope
- Optional TEE mode to bind job spec, weights, and location claim in hardware quote.
- Multi-signal geo proof: IP/ASN, RTT triangulation, optional GPS attest inside TEE.
- Assurance tiers:
  - Bronze: Signed receipt only.
  - Silver: Replication in ≥3 nodes per country.
  - Gold: TEE + replication.
  - Platinum: TEE + replication + local human witness attester.

Deliverables
- Tiers definition + UI badges + policy.
- TEE-enabled runner prototype.

Exit Criteria
- At least Silver tier available in production; Gold piloted.

---

## Phase 5 — Federation & Political Safety Net (Ongoing)
Goal: Make the platform truly global and independent.

Scope
- Multiple independent coordinators.
- Federated transparency logs (mutually anchored).
- Governance doc and multi-sig keyholders in different jurisdictions.
- Optional “jurisdictional mirrors” for sensitive benchmarks.

Deliverables
- Federation protocol + shared anchoring cadence.
- Governance charter + multi-sig ops runbook.

Exit Criteria
- ≥2 independent coordinators interoperating; logs mutually anchored.

---

## Cross-Region Diff MVP Flow (Phase 1)
1. JobSpec says: “Run in: US, KE, BR”.
2. Runner nodes in each country execute → produce Receipt + output.
3. Bundles go to IPFS/Golem → get CIDs.
4. Diff module:
   - Pull outputs for same JobSpec hash.
   - Compare text → mark differences (word- and sentence-level).
   - Store diff JSON in transparency log.
5. Dashboard:
   - Side-by-side outputs.
   - Highlight changed/removed/added text.
   - Show similarity score (%) and number of differences.

---

## Diff JSON Schema v0 (draft)
```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "CrossRegionDiff",
  "type": "object",
  "properties": {
    "job_spec_hash": { "type": "string", "description": "SHA-256 of canonical JobSpec JSON" },
    "model_fingerprint": { "type": "string", "description": "Hash or version of model weights" },
    "benchmark_id": { "type": "string" },
    "items": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "country": { "type": "string" },
          "node_ids": { "type": "array", "items": { "type": "string" } },
          "output_cid": { "type": "string" },
          "output_hash": { "type": "string" },
          "similarity_score": { "type": "number", "minimum": 0, "maximum": 1 },
          "diff_snippet": { "type": "string", "description": "Compact unified-style or JSON Patch-like snippet" }
        },
        "required": ["country", "output_cid", "output_hash", "similarity_score"]
      }
    },
    "pairwise": {
      "type": "array",
      "description": "Optional matrix of pairwise comparisons",
      "items": {
        "type": "object",
        "properties": {
          "from_country": { "type": "string" },
          "to_country": { "type": "string" },
          "similarity_score": { "type": "number" },
          "diff_overview": { "type": "string" }
        },
        "required": ["from_country", "to_country", "similarity_score"]
      }
    },
    "created_at": { "type": "string", "format": "date-time" },
    "signatures": {
      "type": "array",
      "description": "Signatures over the canonicalized diff JSON (Ed25519)",
      "items": { "type": "string" }
    }
  },
  "required": ["job_spec_hash", "benchmark_id", "items", "created_at"]
}
```

Presentation Rules (v0)
- Compute similarity via token-level Jaccard or cosine on sentence embeddings (choose one, document in log entry).
- Diff snippet: compact unified format with +/- markers; limit to top N segments by impact.
- UI highlights:
  - Added (green underline), removed (red strikethrough), changed (yellow background).
  - Show per-country summary: similarity%, token delta, count of changed sentences.

---

## Tech Stack (initial decisions)
- Signatures: Ed25519 over canonical JSON (RFC 8785-like). Receipts and diffs are signed.
- Storage: Golem for compute bundle distribution; IPFS for content-addressed storage of outputs, receipts, and diffs (CIDs referenced in log entries).
- Transparency Log: Merkle-based, Rekor-style API; periodic anchoring to public ledger (e.g., Ethereum/OP Stack/L2). Include inclusion proofs.
- Runner: containerized harness; region selection via Golem node constraints (country/IP/ASN), later TEE option.
- UI: minimal dashboard for Phase 1; expandable to registry and attestation views.

## Risks & Mitigations
- Regional routing fidelity: use multiple signals (IP/ASN, provider geodata) and replication.
- Model drift: pin model fingerprints; enforce via JobSpec and receipts.
- Censorship/takedown: decentralized storage + federation + mirrors (Phase 5).

## Success Metrics (Phase 1)
- ≥3-country run produces consistent logs and signed diffs within 24h.
- Public dashboard loads within 2s and renders diffs deterministically.
- Third-party reproduces one run from transparency log + CIDs.
