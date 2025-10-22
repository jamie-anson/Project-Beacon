# Signature Verification Failure Plan

## Problem Statement
- **Symptom**: `TestE2E_JobCreation_SecurityFlow` fails (valid signature rejected, replay attack classified as signature mismatch).
- **Severity**: High—indicates live runner may reject legitimate signed requests and misreport replay attacks.

## Objectives
- **Ensure** canonical JSON + signing implementation matches test fixtures.
- **Confirm** replay detection emits correct `error_code`.
- **Document** signing pipeline inputs/outputs for regression prevention.

## Current Signals
- Logs show canonical JSON length 572/601 in runtime; test-generated signature likely based on different canonical ordering.
- Replay test receives `signature_mismatch` instead of `replay_detected`, implying canonical hash mismatch before nonce/timestamp comparison.
- sqlmock expectation (`ExpectedBegin`) unmet, indicating DB transaction not started (request rejected early).

## Diagnostics
1. **Reproduce single test**
   - `go test ./internal/api -run TestE2E_JobCreation_SecurityFlow -count=1 -v`.
2. **Inspect canonical JSON**
   - Add temporary logging in `models.JobSpec.CanonicalJSON()` for both test fixture generation and runtime validation.
   - Dump canonical string and `sha256` for reference.
3. **Compare fixtures**
   - Locate signing helper in tests (likely `internal/api/testdata/fixtures` or helper function) and ensure it mirrors production canonicalization.
4. **Validate signing keys**
   - Confirm deterministic private key used in tests matches runner `TRUSTED_KEYS_FILE` entry.
5. **Replay detection logic**
   - Trace validation path in `service/jobspec_validation.go` (or equivalent) to confirm ordering: signature → timestamp/nonce.

## Hypotheses
- **H1**: Test fixture uses outdated field ordering or missing new fields (e.g., `provider_timeout`) causing canonical mismatch.
- **H2**: Canonical JSON includes whitespace or float formatting differences between test generator and runtime.
- **H3**: Replay detection bypassed because signature invalid prior to nonce check; need to adjust test to use valid signature with duplicate `nonce`.

## Action Plan
- [ ] Locate and review test signing code (`internal/api/helpers_signing_test.go` or similar).
- [ ] Rebuild test fixtures by calling actual canonicalization + signing utilities used in production (`models.SignJobSpec`).
- [ ] Update test to use helper that mimics production environment (load key via `crypto/signing`).
- [ ] Add explicit assertions for canonical JSON equality between test and runtime in tests (optional but helpful).
- [ ] For replay test, craft second request with same canonical body & signature to ensure first passes, second fails with `replay_detected`.
- [ ] Validate `ExpectedBegin` by ensuring DB transaction begins after signature check; adjust mocks accordingly.

## Verification
- **Tests**: `go test ./internal/api -run TestE2E_JobCreation_SecurityFlow -count=1`
- **Integration**: Use `scripts/submit-signed-job.js` against local runner to confirm 202 responses.
- **Regression**: Negative controls (bad signature, future timestamp) should continue returning 400 with appropriate `error_code`.

## Documentation
- Update `docs/runbooks/signature-verification.md` with canonicalization rules, sample payloads, and debugging steps.
- Record `sha256` of canonical JSON for fixtures in repository for future comparison.
