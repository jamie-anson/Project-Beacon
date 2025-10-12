# Portal ↔ Runner Signing Contract (SoT)

Last updated: 2025-10-10

## Scope
- Defines the canonicalization and signing rules for JobSpec v1 sent from the Portal to the Runner.
- Prevents signature mismatches (RUNNER_EARLY_FAILURE) due to payload differences.

## Sign THIS object (and only this)
- The inner `jobspec` object in the request body.
- Do NOT include wrapper fields like `target_regions`, `min_regions`, `min_success_rate`, `enable_analysis` in the signed bytes.

## Fields to EXCLUDE before canonicalization
- `id`
- `created_at`
- `signature`
- `public_key`

Notes:
- The server zeros/removes these before verification; clients must sign without them.
- You may still send a non-empty `id` to the server, but the signature must be computed without it.

## Canonicalization (deterministic JSON)
- Recursively sort object keys (stable order).
- UTF-8, no trailing newline or extra whitespace.
- Remove null values and empty objects recursively.
- Keep arrays even if empty; preserve array element order.
- Keep numeric zero values EXCEPT the special case below.
- Special case: Omit `constraints.min_success_rate` if its value is 0 (or effectively absent).

Authoritative implementation in runner:
- `pkg/crypto/canonicalize.go` → `CanonicalizeJobSpecV1()`
  - Deletes `id`, `created_at`, `signature`, `public_key`.
  - Removes null/empty objects; preserves arrays.
  - Removes `constraints.min_success_rate` when value is 0.
  - Encodes with stable key ordering.
- `pkg/models/signature.go` → `JobSpec.VerifySignature()` uses the same canonicalization before verification.

## Signing
- Algorithm: Ed25519
- Steps:
  1) Start from the inner jobspec.
  2) Produce a signable copy that removes: `id`, `created_at`, `signature`, `public_key`.
  3) Canonicalize to deterministic JSON bytes as above.
  4) Sign bytes with the Ed25519 private key.
  5) Base64-encode the signature; set `signature` and `public_key` back on the jobspec you send.

## What the server verifies
- The server constructs the same signable copy of the received jobspec and canonicalizes it using `CanonicalizeJobSpecV1()`; then it verifies the Ed25519 signature against those bytes.

## Testing and Diagnostics
- Admin endpoint: `POST /api/v1/debug/verify` accepts a JSON body of the inner jobspec and returns:
  - `verify`: "ok" or "failed"
  - `server_canonical_len`, `server_canonical_sha256`
  - field presence flags (e.g., `has_created_at`)
- Rate limit: If you receive `rate_limit_exceeded`, wait ~10 seconds and retry.

Suggested workflow:
1) Capture exact inner jobspec from DevTools (Request Payload → View source).
2) Verify as-is:
   - If `verify: failed`, try removing `id` and `created_at` and re-run.
3) Ensure the Portal’s signing routine matches these rules.

## Common Pitfalls
- Signing the wrapper instead of the inner jobspec.
- Including `id` and/or `created_at` in the signed bytes.
- Not removing `signature` / `public_key` before canonicalization.
- Using JSON.stringify without stable key sorting.
- Trailing whitespace/newlines introduced by serializers.
- Sending `constraints.min_success_rate: 0` in the signed bytes (server omits it when 0).

## JS Reference (pseudocode)
```js
// signableJobspec = deepClone(jobspec)
delete signableJobspec.id
delete signableJobspec.created_at
delete signableJobspec.signature
delete signableJobspec.public_key
if (signableJobspec.constraints?.min_success_rate === 0) {
  delete signableJobspec.constraints.min_success_rate
}

// Use a stable canonicalizer (keys sorted at every object level)
const canonical = stableCanonicalize(signableJobspec) // returns Uint8Array/Buffer
const signature = ed25519.sign(privateKey, canonical) // Uint8Array
jobspec.public_key = base64(publicKey)
jobspec.signature = base64(signature)
```

## Trust List
- Runner may enforce a trust list of allowed public keys via `TRUSTED_KEYS_JSON`.
- Ensure the Portal’s base64 public key is included.

## Change Management
- Any change to these rules is a breaking contract and must be coordinated across Portal and Runner.
- Update this SoT document and add test vectors in both repos.

## References
- Runner: `pkg/crypto/canonicalize.go`, `pkg/models/signature.go`, `pkg/models/validator.go`
- Debug: `POST /api/v1/debug/verify`
- Error surface: `RUNNER_EARLY_FAILURE` during job initialization when verification fails.
