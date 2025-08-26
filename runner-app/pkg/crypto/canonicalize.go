package crypto

import (
	"bytes"
	"fmt"
)

// CanonicalizeJobSpecV1 returns RFC8785-like canonical JSON bytes for signing a JobSpec.
// It excludes the signature-bearing fields ("signature" and "public_key").
// Implementation note: this leverages the package's CanonicalJSON() encoder which
// sorts object keys and emits deterministic JSON, and CreateSignableJobSpec() to
// strip signature/public key fields without requiring a concrete JobSpec type.
func CanonicalizeJobSpecV1(spec interface{}) ([]byte, error) {
	// Build a signable copy (zeroes Signature/PublicKey if present)
	signable, err := CreateSignableJobSpec(spec)
	if err != nil {
		return nil, fmt.Errorf("canonicalize v1: create signable: %w", err)
	}
	// Encode deterministically using the signable struct directly (keys retained, zeroed)
	b, err := CanonicalJSON(signable)
	if err != nil {
		return nil, fmt.Errorf("canonicalize v1: encode: %w", err)
	}
	// Avoid accidental trailing spaces/newlines
	return bytes.TrimSpace(b), nil
}

// CanonicalizeGenericV1 canonicalizes any JSON-serializable value using the
// same deterministic encoder used for signatures (useful for tests/vectors).
func CanonicalizeGenericV1(v interface{}) ([]byte, error) {
	b, err := CanonicalJSON(v)
	if err != nil {
		return nil, fmt.Errorf("canonicalize v1 generic: %w", err)
	}
	return bytes.TrimSpace(b), nil
}
