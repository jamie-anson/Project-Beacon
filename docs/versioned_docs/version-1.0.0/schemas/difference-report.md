---
id: difference-report
title: DifferenceReport Schema
---

Schema and identical/differing examples.

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

<Tabs>
  <TabItem value="schema" label="Schema" default>

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://projectbeacon.example/schemas/diff.json",
  "title": "DifferenceReport",
  "type": "object",
  "additionalProperties": false,
  "required": [
    "version",
    "benchmark",
    "job_spec_hash",
    "comparisons",
    "generated_at",
    "attester"
  ],
  "properties": {
    "version": { "type": "string", "pattern": "^[0-9]+\\.[0-9]+$" },
    "benchmark": {
      "type": "object",
      "additionalProperties": false,
      "required": ["name", "id"],
      "properties": {
        "name": { "type": "string", "minLength": 1, "maxLength": 200 },
        "id": { "type": "string", "minLength": 1, "maxLength": 200 }
      }
    },
    "job_spec_hash": { "type": "string", "pattern": "^(sha256|blake3):[A-Fa-f0-9]{64,}$" },
    "receipts": {
      "type": "array",
      "description": "Receipts included in this comparison (by transparency-log reference or CID).",
      "items": {
        "type": "object",
        "additionalProperties": false,
        "required": ["region", "receipt_ref"],
        "properties": {
          "region": { "type": "string", "pattern": "^[A-Z]{2}$" },
          "receipt_ref": { "type": "string", "minLength": 3, "maxLength": 500 }
        }
      },
      "minItems": 2
    },
    "comparisons": {
      "type": "array",
      "minItems": 1,
      "items": {
        "type": "object",
        "additionalProperties": false,
        "required": ["lhs_region", "rhs_region", "similarity", "diff_summary"],
        "properties": {
          "lhs_region": { "type": "string", "pattern": "^[A-Z]{2}$" },
          "rhs_region": { "type": "string", "pattern": "^[A-Z]{2}$" },
          "similarity": { "type": "number", "minimum": 0, "maximum": 1 },
          "diff_summary": { "type": "string", "maxLength": 5000 },
          "auto_diff_algo": { "type": "string", "default": "diff-match-patch" },
          "auto_diff_fragment": { "type": "string", "maxLength": 20000 },
          "classification": {
            "type": "string",
            "enum": ["none", "cosmetic", "content", "content-high-salience"]
          },
          "human_notes": { "type": "string", "maxLength": 5000 },
          "lhs_output_hash": { "type": "string", "pattern": "^(sha256|blake3):[A-Fa-f0-9]{64,}$" },
          "rhs_output_hash": { "type": "string", "pattern": "^(sha256|blake3):[A-Fa-f0-9]{64,}$" }
        }
      }
    },
    "generated_at": { "type": "string", "format": "date-time" },
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
    "auto_diff_report_hash": {
      "type": "string",
      "pattern": "^(sha256|blake3):[A-Fa-f0-9]{64,}$",
      "description": "Hash of the raw automated diff JSON this report reviewed."
    },
    "status": {
      "type": "string",
      "enum": ["verified", "inconclusive", "rejected"],
      "default": "verified"
    }
  }
}
```

  </TabItem>
  <TabItem value="identical" label="Identical Example">

```json
{
  "version": "1.0",
  "benchmark": {
    "name": "Text Generation Consistency",
    "id": "text-gen-consistency-v1"
  },
  "job_spec_hash": "sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
  "receipts": [
    {
      "region": "US",
      "receipt_ref": "bafybeidr7e2c6q3yil5v2m5h5d3q7w8e9r0t1y2u3i4o5p6q7r8s9t0u1v2"
    },
    {
      "region": "EU",
      "receipt_ref": "bafybeidr7e2c6q3yil5v2m5h5d3q7w8e9r0t1y2u3i4o5p6q7r8s9t0u1v3"
    }
  ],
  "comparisons": [
    {
      "lhs_region": "US",
      "rhs_region": "EU",
      "similarity": 1.0,
      "diff_summary": "No differences detected",
      "auto_diff_algo": "diff-match-patch",
      "auto_diff_fragment": "",
      "classification": "none",
      "lhs_output_hash": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      "rhs_output_hash": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
    }
  ],
  "generated_at": "2025-08-12T14:30:00Z",
  "attester": {
    "org": "Project Beacon",
    "jurisdiction": "US",
    "pubkey_fingerprint": "sha256:1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b"
  },
  "status": "verified"
}
```

  </TabItem>
  <TabItem value="differing" label="Differing Example">

```json
{
  "version": "1.0",
  "benchmark": {
    "name": "Text Generation Consistency",
    "id": "text-gen-consistency-v1"
  },
  "job_spec_hash": "sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
  "receipts": [
    {
      "region": "US",
      "receipt_ref": "bafybeidr7e2c6q3yil5v2m5h5d3q7w8e9r0t1y2u3i4o5p6q7r8s9t0u1v2"
    },
    {
      "region": "EU",
      "receipt_ref": "bafybeidr7e2c6q3yil5v2m5h5d3q7w8e9r0t1y2u3i4o5p6q7r8s9t0u1v3"
    }
  ],
  "comparisons": [
    {
      "lhs_region": "US",
      "rhs_region": "EU",
      "similarity": 0.85,
      "auto_diff_algo": "diff-match-patch",
      "auto_diff_fragment": "<span class='diff-delete'>- I am an AI assistant.</span><span class='diff-insert'>+ I am an AI assistant designed to help.</span>",
      "classification": "cosmetic",
      "human_notes": "Minor differences in response formatting, no material difference in meaning.",
      "lhs_output_hash": "sha256:a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b",
      "rhs_output_hash": "sha256:b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2"
    }
  ],
  "auto_diff_report_hash": "sha256:c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3",
  "generated_at": "2025-08-12T14:35:22Z",
  "attester": {
    "org": "Project Beacon",
    "jurisdiction": "US",
    "pubkey_fingerprint": "sha256:1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b"
  },
  "status": "verified"
}
```

  </TabItem>
</Tabs>
