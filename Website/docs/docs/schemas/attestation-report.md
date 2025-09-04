---
id: attestation-report
title: AttestationReport Schema
---

Schema and example attester confirmation.

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

<Tabs>
  <TabItem value="schema" label="Schema" default>

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://projectbeacon.example/schemas/attestation.json",
  "title": "AttestationReport",
  "type": "object",
  "additionalProperties": false,
  "required": [
    "version",
    "subject",
    "checks",
    "result",
    "attester",
    "generated_at"
  ],
  "properties": {
    "version": { "type": "string", "pattern": "^[0-9]+\\.[0-9]+$" },
    "subject": {
      "type": "object",
      "additionalProperties": false,
      "required": ["type", "references"],
      "properties": {
        "type": { "type": "string", "enum": ["receipt", "job", "diff", "bundle"] },
        "references": {
          "type": "array",
          "minItems": 1,
          "items": {
            "type": "object",
            "additionalProperties": false,
            "properties": {
              "tl_ref": { "type": "string", "minLength": 3, "maxLength": 500 },
              "cid": { "type": "string", "minLength": 10, "maxLength": 200 }
            }
          }
        }
      }
    },
    "checks": {
      "type": "object",
      "additionalProperties": false,
      "required": ["signatures", "content", "geo", "runtime"],
      "properties": {
        "signatures": {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "dsse_verified": { "type": "boolean" },
            "rekor_inclusion_proof": { "type": "boolean" },
            "tree_consistent": { "type": "boolean" }
          }
        },
        "content": {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "job_spec_hash_match": { "type": "boolean" },
            "image_digest_match": { "type": "boolean" },
            "sbom_verified": { "type": "boolean" },
            "weights_hash_match": { "type": "boolean" },
            "inputs_hash_match": { "type": "boolean" },
            "outputs_hash_match": { "type": "boolean" }
          }
        },
        "geo": {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "asn_ok": { "type": "boolean" },
            "rtt_consistent": { "type": "boolean" },
            "gps_claim_present": { "type": "boolean" }
          }
        },
        "runtime": {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "tee_quote_verified": { "type": "boolean" },
            "tpm_quote_verified": { "type": "boolean" },
            "perf_within_expected_band": { "type": "boolean" },
            "deterministic_rerun_attempted": { "type": "boolean" },
            "deterministic_rerun_match": { "type": "boolean" }
          }
        }
      }
    },
    "result": { "type": "string", "enum": ["confirm", "contest", "inconclusive"] },
    "notes": { "type": "string", "maxLength": 5000 },
    "attester": {
      "type": "object",
      "additionalProperties": false,
      "required": ["org", "jurisdiction", "pubkey_fingerprint"],
      "properties": {
        "org": { "type": "string", "minLength": 1, "maxLength": 200 },
        "jurisdiction": { "type": "string", "pattern": "^[A-Z]{2}$" },
        "pubkey_fingerprint": { "type": "string", "pattern": "^(sha256|blake3):[A-Fa-f0-9]{64,}$" }
      }
    },
    "generated_at": { "type": "string", "format": "date-time" }
  }
}
```

  </TabItem>
  <TabItem value="example" label="Example">

```json
{
  "version": "1.0",
  "subject": {
    "type": "diff",
    "references": [
      {
        "tl_ref": "rekor://abc123",
        "cid": "bafybeidr7e2c6q3yil5v2m5h5d3q7w8e9r0t1y2u3i4o5p6q7r8s9t0u1v2"
      }
    ]
  },
  "checks": {
    "signatures": {
      "dsse_verified": true,
      "rekor_inclusion_proof": true,
      "tree_consistent": true
    },
    "content": {
      "job_spec_hash_match": true,
      "image_digest_match": true,
      "sbom_verified": true,
      "weights_hash_match": true,
      "inputs_hash_match": true,
      "outputs_hash_match": true
    },
    "geo": {
      "asn_ok": true,
      "rtt_consistent": true,
      "gps_claim_present": true
    },
    "runtime": {
      "tee_quote_verified": true,
      "tpm_quote_verified": true,
      "perf_within_expected_band": true,
      "deterministic_rerun_attempted": true,
      "deterministic_rerun_match": true
    }
  },
  "result": "confirm",
  "notes": "All verification checks passed. The difference report shows only expected variations within tolerance limits.",
  "attester": {
    "org": "Project Beacon",
    "jurisdiction": "US",
    "pubkey_fingerprint": "sha256:1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b"
  },
  "generated_at": "2025-08-12T15:00:00Z"
}
```

  </TabItem>
</Tabs>
