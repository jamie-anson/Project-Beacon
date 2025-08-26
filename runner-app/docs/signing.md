# Project Beacon Signature System

This document covers the complete signature system for Project Beacon JobSpecs, including key management, signing/verification, server configuration, and troubleshooting.

## Overview

Project Beacon uses Ed25519 signatures to ensure JobSpec authenticity and prevent tampering. All JobSpecs must be cryptographically signed before submission to the runner API.

### Security Features

- **Replay Protection**: Prevents reuse of signed JobSpecs via nonce tracking
- **Timestamp Freshness**: Validates JobSpec timestamps within configurable skew limits
- **Rate Limiting**: Limits signature verification failures to prevent abuse
- **Trusted Keys**: Optional allowlist enforcement for authorized signers
- **Backward Compatibility**: Supports legacy v0 canonicalization with deprecation warnings

## Key Management

### Generating Keys

Use the provided script to generate a new Ed25519 key pair:

```bash
go run ./scripts/generate-keypair.go
```

This outputs:
- **Private Key**: Base64-encoded, keep secure
- **Public Key**: Base64-encoded, share with runner operators
- **Key ID (KID)**: SHA256 hash of public key for identification

### Storing Keys Securely

**Private Keys:**
```bash
# Store in environment variable
export BEACON_PRIVATE_KEY="your-base64-private-key"

# Or in secure file with restricted permissions
echo "your-base64-private-key" > ~/.beacon/private.key
chmod 600 ~/.beacon/private.key
```

**Public Keys:**
```bash
# Public keys can be shared openly
echo "your-base64-public-key" > ~/.beacon/public.key
```

### Key ID (KID) Generation

The Key ID is the SHA256 hash of the public key:

```bash
# Calculate KID from public key
echo -n "your-base64-public-key" | sha256sum
```

## Signing JobSpecs

### Required Metadata Fields

All signed JobSpecs should include security metadata. These fields are
required when trust enforcement is enabled (`TRUST_ENFORCE=true`) and
strongly recommended in all environments:

```json
{
  "id": "my-job-v1",
  "version": "1.0",
  "metadata": {
    "timestamp": "2025-08-22T14:50:32Z",
    "nonce": "unique-nonce-12345"
  },
  "signature": "base64-signature",
  "public_key": "base64-public-key"
}
```

### Using the Regenerate Script

Generate signed examples with current timestamps and nonces:

```bash
# Regenerate default examples in ./examples directory
bash scripts/regenerate_signed_examples.sh

# Or specify a list
EXAMPLES="jobspec-who-are-you.json" bash scripts/regenerate_signed_examples.sh

# Output (example):
# Building sigtool...
# Regenerating signed examples...
# Signed: examples/jobspec-who-are-you.json.signed
# Verified: examples/jobspec-who-are-you.json.signed
# Signer public key: <base64-public-key>
# Done.
```

### Manual Signing Process

1. **Create JobSpec** with required fields
2. **Add security metadata**:
   - `timestamp`: Current UTC time in RFC3339 format
   - `nonce`: Unique identifier (UUID or random string)
3. **Sign the JobSpec**:
   ```go
   keyPair, _ := crypto.GenerateKeyPair()
   jobSpec.Sign(keyPair.PrivateKey)
   ```
4. **Submit to API**:
   ```bash
   curl -X POST http://localhost:8090/api/v1/jobs \
     -H "Content-Type: application/json" \
     -d @signed-jobspec.json
   ```

### Canonicalization Behavior (v1)

- The canonicalized, signable copy retains the `signature` and `public_key` keys but zeroes their values when computing the message to sign/verify.
- Deterministic JSON encoding is used for both signing and verification to ensure parity.
- Server verification accepts the current method and performs a compatibility fallback:
  - v1 canonicalization is equivalent to the current method (keys retained, values zeroed).
  - Legacy v0 canonicalization (keys removed) may be accepted temporarily with a deprecation warning; re-sign using the current method.

## Local Verification with sigtool

### Installation

```bash
go build -o sigtool ./cmd/sigtool
```

### Usage

**Sign a JobSpec:**
```bash
./sigtool sign --private-key="$BEACON_PRIVATE_KEY" --input=jobspec.json --output=signed.json
```

**Verify a signature:**
```bash
./sigtool verify --public-key="$BEACON_PUBLIC_KEY" --input=signed.json
```

**Generate key pair:**
```bash
./sigtool keygen --output-dir=~/.beacon
```

**Extract public key from JobSpec:**
```bash
./sigtool extract-pubkey --input=signed.json
```

## Server Configuration

### Environment Variables

**Security Settings:**
```bash
# Enable replay protection (recommended)
REPLAY_PROTECTION_ENABLED=true

# Timestamp validation (minutes)
TIMESTAMP_MAX_SKEW_MINUTES=5    # ±5 minutes from current time
TIMESTAMP_MAX_AGE_MINUTES=10    # Maximum age of timestamps

# Trusted keys enforcement
TRUST_ENFORCE=true              # Require keys to be in allowlist
TRUSTED_KEYS_FILE=trusted-keys.json
TRUSTED_KEYS_RELOAD_SECONDS=30  # Optional: hot-reload trusted keys file

# Development-only bypass (never enable in prod/CI)
RUNNER_SIG_BYPASS=false
```

**Redis Configuration:**
```bash
REDIS_URL=redis://localhost:6379
```

### Trusted Keys Configuration

Create `trusted-keys.json` with authorized public keys. The file is a JSON array of entries:

```json
[
  {
    "kid": "sha256-hash-of-public-key",
    "public_key": "base64-encoded-public-key",
    "status": "active",
    "not_before": "2025-08-22T14:50:32Z",
    "not_after": "2026-08-22T14:50:32Z"
  }
]
```

Validation rules enforced on load (`internal/config/trust.go`):
- `kid` required and unique
- `public_key` required, base64-decodable, and unique across entries
- `status` must be `active`, `revoked`, or empty
- `not_before`/`not_after` optional RFC3339; if both present, `not_before` ≤ `not_after`

### Server Startup

```bash
# Terminal D (Local infra: docker compose for Postgres/Redis)
docker compose up postgres redis

# Terminal B: Start API server
go run ./cmd/runner/main.go
```

## API Integration

### JobSpec Schema Requirements

**Required Fields (v1):**
- `signature`: Base64-encoded Ed25519 signature
- `public_key`: Base64-encoded Ed25519 public key
- `metadata.timestamp`: RFC3339 timestamp (required when `TRUST_ENFORCE=true`)
- `metadata.nonce`: Unique nonce string (required when `TRUST_ENFORCE=true`)

**Example Request:**
```bash
curl -X POST http://localhost:8090/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "id": "benchmark-test",
    "version": "1.0",
    "metadata": {
      "timestamp": "2025-08-22T14:50:32Z",
      "nonce": "unique-12345"
    },
    "signature": "base64-signature",
    "public_key": "base64-public-key"
  }'
```

**Success Response:**
```json
{
  "id": "benchmark-test",
  "status": "enqueued"
}
```

## Troubleshooting Guide

### Common Errors and Solutions

#### `signature_mismatch`
**Error:** `{"error": "signature verification failed", "error_code": "signature_mismatch"}`

**Causes & Fixes:**
- **Invalid signature**: Re-sign the JobSpec with correct private key
- **Canonicalization mismatch**: Use current signing method, not legacy v0
- **Modified JobSpec**: Don't modify JobSpec after signing
- **Wrong public key**: Ensure public key matches the private key used for signing

**Debug:**
```bash
# Verify signature locally
go run ./scripts/verify-examples.go ./examples
```

## End-to-End (E2E) Sign → Submit Test

Follow this to validate the full flow on your machine.

- __Terminal D (Local infra: Postgres + Redis via docker compose)__
  ```bash
  docker compose up -d postgres redis
  ```

- __Terminal B (Go API server on :8090)__
  ```bash
  go run ./cmd/runner/main.go
  ```

- __Terminal C (Actions)__
  ```bash
  make e2e
  ```

Expected output:
```text
HTTP status: 202
Body: {"id":"who-are-you-benchmark-v1","status":"enqueued"}
E2E PASS: received 202 and expected response body.
```

## Trust Policy (Allowlist)

You can enforce that only pre-approved public keys may submit jobs.

- __Config file__: `config/trusted_keys.example.json` (copy to your own path and edit)
- __Enable enforcement__ (dev example):
  ```bash
  export TRUSTED_KEYS_FILE="$(pwd)/config/trusted_keys.example.json"
  export TRUST_ENFORCE=true
  # Optional: hot-reload trusted keys every 30s
  export TRUSTED_KEYS_RELOAD_SECONDS=30
  go run ./cmd/runner/main.go
  ```

Each entry:
```json
[
  {
    "kid": "dev-2025-q3",
    "public_key": "<base64-ed25519>",
    "status": "active",
    "not_before": "2025-08-01T00:00:00Z",
    "not_after": "2026-08-01T00:00:00Z"
  }
]
```

Behavior when `TRUST_ENFORCE=true`:
- __trusted__: accepted
- __revoked__: rejected (400 trust_violation:revoked)
- __not_yet_valid__: rejected (400 trust_violation:not_yet_valid)
- __expired__: rejected (400 trust_violation:expired)
- __unknown__: rejected (400 trust_violation:unknown)

Logs include `trusted-keys: evaluation` with `kid` and status for observability.

### Dev Bypass (local only)

For quick local iteration, you can bypass signature verification entirely. This should never be used in CI or production.

```bash
export RUNNER_SIG_BYPASS=true
go run ./cmd/runner/main.go
```

Notes:
- The server will log a warning per request: `RUNNER_SIG_BYPASS enabled: skipping signature verification (dev only)`.
- Trust policy checks may still run (and be enforced) unless you also disable them.

### Maintainer Notes: Examples

Keep example signatures up to date when signing logic changes.

Commands:
- `make regen-examples` — regenerate `examples/*.signed` using `.dev/keys/private.key` and current `sigtool`.
- `make validate-examples` — validate JSON of `examples/*.signed`.
- `make install-pre-commit` — install a pre-commit hook to enforce freshness locally.

CI:
- Workflow: `.github/workflows/validate-examples.yml` regenerates examples and fails if diffs are detected.

### Legacy v0 Signature Compatibility

- The server may accept legacy v0 signatures (where `signature` and `public_key` fields were removed before signing) temporarily for backward compatibility.
- Submissions verified via v0 will emit a deprecation log like:
  ```
  DEPRECATED: JobSpec <id> signed with v0 canonicalization - please re-sign with current method
  ```
- Action: Re-sign all JobSpecs using the current canonicalization method (keys retained, values zeroed) to avoid future rejection when v0 support is removed.

#### `replay_detected`
**Error:** `{"error": "replay protection failed: replay detected", "error_code": "replay_detected"}`

**Causes & Fixes:**
- **Reused nonce**: Generate a new unique nonce for each JobSpec
- **Duplicate submission**: Each JobSpec submission must have a unique nonce

**Debug:**
```bash
# Check Redis for used nonces
redis-cli KEYS "nonce:*"
```

#### `timestamp_invalid`
Timestamp failures are unified under `timestamp_invalid` with a structured reason:

Examples:
```json
{"error":"timestamp validation failed","error_code":"timestamp_invalid","details":{"reason":"too_old"}}
{"error":"timestamp validation failed","error_code":"timestamp_invalid","details":{"reason":"too_far_in_future"}}
{"error":"timestamp validation failed","error_code":"timestamp_invalid","details":{"reason":"format_invalid"}}
```

Causes & Fixes:
- __too_old__: Use current time when signing; ensure timestamp within `TIMESTAMP_MAX_AGE_MINUTES`
- __too_far_in_future__: Do not use future timestamps; respect `TIMESTAMP_MAX_SKEW_MINUTES`
- __format_invalid__: Ensure RFC3339 format (e.g., `2025-08-22T14:50:32Z`)

Helper:
```bash
# Generate fresh timestamp
date -u +"%Y-%m-%dT%H:%M:%SZ"
```

#### `trust_violation:*`
Errors are namespaced to indicate the cause. Examples:

```json
{"error":"untrusted signing key: unknown key","error_code":"trust_violation:unknown"}
{"error":"trusted key revoked","error_code":"trust_violation:revoked"}
{"error":"trusted key expired","error_code":"trust_violation:expired"}
{"error":"trusted key not yet valid","error_code":"trust_violation:not_yet_valid"}
```

Causes & Fixes:
- **unknown**: Add public key to your trusted keys file and reload
- **revoked**: Use a different active key
- **expired / not_yet_valid**: Adjust validity window or rotate keys

#### `rate_limit_exceeded`
**Error:** `{"error": "rate limit exceeded", "error_code": "rate_limit_exceeded"}`

**Causes & Fixes:**
- **Multiple failed attempts**: Wait before retrying
- **Fix signature issues**: Resolve underlying signature problems first

### Debugging Commands

**Check JobSpec structure:**
```bash
jq . signed-jobspec.json
```

**Validate timestamp format:**
```bash
date -d "2025-08-22T14:50:32Z" 2>/dev/null && echo "Valid" || echo "Invalid"
```

**Test signature locally:**
```bash
go test ./pkg/models -run TestJobSpecSigningAndVerification -v
```

#### `protection_unavailable:replay`
**Error:** `{"error":"replay protection unavailable","error_code":"protection_unavailable:replay"}`

**Cause:** Replay protection is enabled, but Redis is unavailable.

**Fix:** Ensure Redis is running and configured correctly, or disable replay protection in dev.

---

## Manual Verification (curl)

Use labeled terminals and port 8090.

- __Terminal D (Local infra: Postgres + Redis via docker compose)__
  ```bash
  docker compose up -d postgres redis
  ```

- __Terminal B (Go API server on :8090)__
  ```bash
  go run ./cmd/runner/main.go
  ```

- __Terminal C (Actions)__
  - Missing nonce (expect 400 missing_field:nonce):
    ```bash
    curl -sS -X POST http://localhost:8090/api/v1/jobs \
      -H 'Content-Type: application/json' \
      -d '{
            "id":"demo","version":"1.0",
            "metadata":{"timestamp":"2025-08-22T14:50:32Z"},
            "signature":"","public_key":""
          }' | jq
    ```
  - Malformed timestamp (expect 400 timestamp_invalid/format_invalid):
    ```bash
    curl -sS -X POST http://localhost:8090/api/v1/jobs \
      -H 'Content-Type: application/json' \
      -d '{
            "id":"demo","version":"1.0",
            "metadata":{"timestamp":"2025/08/22 14:50:32","nonce":"n1"},
            "signature":"","public_key":""
          }' | jq
    ```
  - Redis unavailable with replay protection on (expect 503 protection_unavailable:replay):
    1) Stop Redis in Terminal D: `docker compose stop redis`
    2) Submit any request:
    ```bash
    curl -sS -X POST http://localhost:8090/api/v1/jobs \
      -H 'Content-Type: application/json' \
      -d '{
            "id":"demo","version":"1.0",
            "metadata":{"timestamp":"2025-08-22T14:50:32Z","nonce":"n2"},
            "signature":"","public_key":""
          }' | jq
    ```

**Check server logs:**
```bash
# Look for signature verification details
grep "signature verification" server.log
```

### Performance Considerations

**Redis Connection:**
- Ensure Redis is accessible for replay protection
- Monitor Redis memory usage for nonce storage
- Consider Redis clustering for high availability

**Signature Verification:**
- Ed25519 verification is fast (~0.1ms per signature)
- Rate limiting prevents abuse of verification failures
- Consider caching public key validation results

## Migration Guide

### Upgrading from v0 Signatures

Legacy v0 signatures are supported with deprecation warnings:

```
DEPRECATED: JobSpec benchmark-test signed with v0 canonicalization - please re-sign with current method
```

**Migration steps:**
1. **Re-sign JobSpecs** using current signing method
2. **Update tooling** to use new canonicalization
3. **Monitor logs** for deprecation warnings
4. **Plan removal** of v0 support in future versions

### Schema Evolution

**Current (v1):**
- Deterministic canonicalization
- Required security metadata
- Replay protection

**Future (v2):**
- Enhanced key rotation support
- Multi-signature support
- Hardware security module integration

## Security Best Practices

### Key Management
- **Rotate keys regularly** (recommended: annually)
- **Use hardware security modules** for production keys
- **Separate keys per environment** (dev/staging/prod)
- **Monitor key usage** and detect anomalies

### Operational Security
- **Enable all security features** in production
- **Monitor signature failure rates** for abuse detection
- **Use trusted keys allowlist** to control access
- **Regular security audits** of signing infrastructure

### Development Workflow
- **Never commit private keys** to version control
- **Use environment variables** for key storage
- **Test signature verification** in CI/CD pipelines
- **Validate examples** automatically on changes

## OpenAPI Schema

The JobSpec schema includes signature requirements:

```yaml
JobSpec:
  type: object
  required:
    - id
    - version
    - signature
    - public_key
    - metadata
  properties:
    signature:
      type: string
      format: base64
      description: Ed25519 signature of canonicalized JobSpec
    public_key:
      type: string
      format: base64
      description: Ed25519 public key for signature verification
    metadata:
      type: object
      required:
        - timestamp
        - nonce
      properties:
        timestamp:
          type: string
          format: date-time
          description: RFC3339 timestamp for freshness validation
        nonce:
          type: string
          description: Unique identifier for replay protection
```

## Support

For additional help:
- **GitHub Issues**: Report bugs and feature requests
- **Documentation**: Check `/docs` directory for updates
- **Examples**: Reference `/examples` for working JobSpecs
- **Tests**: Run test suite for validation: `go test ./...`
