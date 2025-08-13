---
id: receipt
title: Receipt Schema
---

Receipt JSON Schema and example from MVP.

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

<Tabs>
  <TabItem value="schema" label="Schema" default>

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://projectbeacon.example/schemas/receipt.json",
  "title": "Receipt",
  "type": "object",
  "additionalProperties": false,
  "required": [
    "version",
    "job_id",
    "job_spec_hash",
    "runner_instance",
    "hardware",
    "environment",
    "geo",
    "run",
    "artefacts",
    "created_at"
  ],
  "properties": {
    "version": { "type": "string", "pattern": "^[0-9]+\\.[0-9]+$" },
    "job_id": { "type": "string", "pattern": "^[a-z0-9-]{6,64}$" },
    "job_spec_hash": { "type": "string", "pattern": "^(sha256|blake3):[A-Fa-f0-9]{64,}$" },
    "runner_instance": {
      "type": "object",
      "additionalProperties": false,
      "required": ["provider_id", "region", "instance_id"],
      "properties": {
        "provider_id": { "type": "string", "minLength": 3, "maxLength": 120 },
        "region": { "type": "string", "pattern": "^[A-Z]{2}$" },
        "instance_id": { "type": "string", "minLength": 3, "maxLength": 120 }
      }
    },
    "hardware": {
      "type": "object",
      "additionalProperties": false,
      "required": ["cpu", "ram_gb"],
      "properties": {
        "cpu": { "type": "string", "minLength": 1, "maxLength": 200 },
        "gpu": { "type": "string", "maxLength": 200 },
        "gpu_vram_gb": { "type": "number", "minimum": 0 },
        "ram_gb": { "type": "number", "minimum": 0 },
        "storage_gb": { "type": "number", "minimum": 0 },
        "driver": { "type": "string", "maxLength": 200 },
        "microcode": { "type": "string", "maxLength": 200 },
        "power_cap_w": { "type": "number", "minimum": 0 }
      }
    },
    "environment": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "kernel": { "type": "string", "maxLength": 200 },
        "os": { "type": "string", "maxLength": 200 },
        "container_runtime": { "type": "string", "maxLength": 200 },
        "runner_commit": { "type": "string", "maxLength": 80 }
      }
    },
    "geo": {
      "type": "object",
      "additionalProperties": false,
      "required": ["ip", "asn"],
      "properties": {
        "ip": { "type": "string", "format": "ipv4" },
        "asn": { "type": "integer", "minimum": 1 },
        "rtts_ms": {
          "type": "array",
          "items": { "type": "number", "minimum": 0 },
          "minItems": 1
        },
        "gps_claim": {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "lat": { "type": "number", "minimum": -90, "maximum": 90 },
            "lon": { "type": "number", "minimum": -180, "maximum": 180 },
            "accuracy_m": { "type": "number", "minimum": 0 }
          }
        }
      }
    },
    "run": {
      "type": "object",
      "additionalProperties": false,
      "required": ["started_at", "ended_at", "outputs_hash"],
      "properties": {
        "started_at": { "type": "string", "format": "date-time" },
        "ended_at": { "type": "string", "format": "date-time" },
        "tok_per_s": { "type": "number", "minimum": 0 },
        "p50_latency_ms": { "type": "number", "minimum": 0 },
        "p95_latency_ms": { "type": "number", "minimum": 0 },
        "cost_estimate": { "type": "number", "minimum": 0 },
        "throttling_events": { "type": "integer", "minimum": 0 },
        "outputs_hash": { "type": "string", "pattern": "^(sha256|blake3):[A-Fa-f0-9]{64,}$" }
      }
    },
    "artefacts": {
      "type": "object",
      "additionalProperties": false,
      "required": ["bundle_cid"],
      "properties": {
        "bundle_cid": { "type": "string", "minLength": 10, "maxLength": 200 },
        "stdout_hash": { "type": "string", "pattern": "^(sha256|blake3):[A-Fa-f0-9]{64,}$" },
        "logs_hash": { "type": "string", "pattern": "^(sha256|blake3):[A-Fa-f0-9]{64,}$" }
      }
    },
    "attestations": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "tpm_quote": { "type": "string", "maxLength": 50000 },
        "tee_quote": { "type": "string", "maxLength": 50000 },
        "witness_sigs": {
          "type": "array",
          "items": { "type": "string", "maxLength": 2000 },
          "uniqueItems": true
        }
      }
    },
    "created_at": { "type": "string", "format": "date-time" },
    "notes": { "type": "string", "maxLength": 2000 }
  }
}
```

  </TabItem>
  <TabItem value="example" label="MVP Example">

```json
{
  "version": "1.0",
  "job_id": "job-1234-5678-90ab",
  "job_spec_hash": "sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
  "runner_instance": {
    "provider_id": "aws-ec2",
    "region": "US",
    "instance_id": "i-0123456789abcdef0"
  },
  "hardware": {
    "cpu": "Intel Xeon Platinum 8375C",
    "gpu": "NVIDIA A10G",
    "gpu_vram_gb": 24,
    "ram_gb": 64
  },
  "environment": {
    "kernel": "5.15.0-1050-aws",
    "os": "Ubuntu 22.04.3 LTS",
    "container_runtime": "containerd://1.6.21"
  },
  "geo": {
    "ip": "203.0.113.42",
    "asn": 14618,
    "rtts_ms": [28.4, 27.9, 28.1]
  },
  "run": {
    "started_at": "2025-08-12T12:00:00Z",
    "ended_at": "2025-08-12T12:05:30Z",
    "outputs_hash": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
  },
  "artefacts": {
    "bundle_cid": "bafybeidr7e2c6q3yil5v2m5h5d3q7w8e9r0t1y2u3i4o5p6q7r8s9t0u1v2"
  },
  "created_at": "2025-08-12T12:06:00Z"
}
```

  </TabItem>
</Tabs>
