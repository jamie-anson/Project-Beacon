---
id: field-reference
title: Schema Field Reference
---

This document provides detailed explanations for all fields in the Project Beacon schemas.

## Common Fields

### version
- **Type**: String (pattern: `^[0-9]+\.[0-9]+$`)
- **Description**: The version of the schema being used, in semantic versioning format (e.g., "1.0").

### generated_at
- **Type**: String (ISO 8601 date-time)
- **Description**: When this document was generated, in UTC.

### attester
- **Type**: Object
- **Description**: Information about the entity creating this document.
  - **org**: Organization name (string, 1-200 chars)
  - **jurisdiction**: ISO 3166-1 alpha-2 country code (e.g., "US", "GB")
  - **pubkey_fingerprint**: Fingerprint of the public key used for signing (format: `(sha256|blake3):[A-Fa-f0-9]{64,}`)

## JobSpec Schema

### benchmark
- **Type**: Object
- **Description**: Information about the benchmark being run.
  - **name**: Human-readable name of the benchmark (1-200 chars)
  - **id**: Unique identifier for the benchmark (1-200 chars)

### job_spec_hash
- **Type**: String (format: `(sha256|blake3):[A-Fa-f0-9]{64,}`)
- **Description**: Cryptographic hash of the job specification.

### image
- **Type**: String
- **Description**: Container image reference (e.g., "docker.io/library/ubuntu:20.04").

### command
- **Type**: Array of strings
- **Description**: Command and arguments to run in the container.

### env
- **Type**: Object (string key-value pairs)
- **Description**: Environment variables to set in the container.

## Receipt Schema

### runner_instance
- **Type**: Object
- **Description**: Information about the runner that executed the job.
  - **id**: Unique identifier for the runner instance
  - **version**: Version of the runner software

### hardware
- **Type**: Object
- **Description**: Hardware specifications of the runner.
  - **cpu**: CPU information (model, cores, etc.)
  - **memory**: Total available memory in bytes
  - **gpu**: GPU information if available

### environment
- **Type**: Object
- **Description**: Software and system environment details.
  - **os**: Operating system name and version
  - **container_runtime**: Container runtime and version
  - **libraries**: Key library versions

### geo
- **Type**: Object
- **Description**: Geographic information about where the job ran.
  - **region**: ISO 3166-1 alpha-2 country code
  - **coordinates**: [Optional] GPS coordinates
  - **timezone**: IANA timezone name

## DifferenceReport Schema

### subject
- **Type**: Object
- **Description**: The subject of this report.
  - **type**: Type of subject ("receipt", "job", "diff", "bundle")
  - **references**: Array of references to the subjects

### checks
- **Type**: Object
- **Description**: Results of various verification checks.
  - **signatures**: Digital signature validations
  - **content**: Content verification results
  - **geo**: Geographic verification results
  - **runtime**: Runtime environment validations

### result
- **Type**: String (enum: "confirm", "contest", "inconclusive")
- **Description**: Overall result of the attestation.

### notes
- **Type**: String (max 5000 chars)
- **Description**: Human-readable notes about the attestation.

## AttestationReport Schema

### receipts
- **Type**: Array of objects
- **Description**: Receipts included in this comparison.
  - **region**: ISO 3166-1 alpha-2 country code
  - **receipt_ref**: Reference to the receipt (3-500 chars)

### comparisons
- **Type**: Array of objects
- **Description**: Comparison results between regions.
  - **lhs_region**: Left-hand side region code
  - **rhs_region**: Right-hand side region code
  - **similarity**: Similarity score (0.0 to 1.0)
  - **classification**: Type of differences found ("none", "cosmetic", "content", "content-high-salience")
  - **human_notes**: Optional human-readable notes (max 5000 chars)

### status
- **Type**: String (enum: "verified", "inconclusive", "rejected")
- **Default**: "verified"
- **Description**: Current status of this report.

## Cryptographic Hashes

All hash values in the schemas follow this format:
- **Prefix**: Either "sha256:" or "blake3:"
- **Value**: Lowercase hex-encoded hash value (minimum 64 characters)
- **Example**: `sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08`

## Best Practices

1. **Versioning**: Always include the version field and increment it when making breaking changes.
2. **Timestamps**: Use UTC for all timestamps and include timezone information.
3. **Hashing**: Use consistent hashing algorithms and document any changes to them.
4. **Validation**: Always validate documents against their schemas before processing.
5. **Security**: Keep private keys secure and rotate them periodically.

## Common Patterns

### Region Codes
Use ISO 3166-1 alpha-2 country codes (e.g., "US", "GB", "JP").

### Date/Time Format
Use ISO 8601 format: `YYYY-MM-DDTHH:MM:SSZ` (e.g., "2025-08-12T14:30:00Z").

### References
- [JSON Schema Specification](https://json-schema.org/)
- [ISO 3166-1 Country Codes](https://www.iso.org/iso-3166-country-codes.html)
- [ISO 8601 Date/Time Format](https://www.iso.org/iso-8601-date-and-time-format.html)
