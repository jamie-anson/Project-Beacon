package crypto

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// CanonicalizeJobSpecV1 returns RFC8785-like canonical JSON bytes for signing a JobSpec.
// It excludes the signature-bearing fields ("signature" and "public_key"), and also
// removes "id" and "created_at" fields to support portal-style signing.
// Portal signs without ID (server adds it later), so we must exclude it for verification.
func CanonicalizeJobSpecV1(spec interface{}) ([]byte, error) {
	// Build a signable copy (zeroes Signature/PublicKey if present)
	signable, err := CreateSignableJobSpec(spec)
	if err != nil {
		return nil, fmt.Errorf("canonicalize v1: create signable: %w", err)
	}
	
	// Convert to map to remove id and created_at fields (for portal compatibility)
	var m map[string]interface{}
	signableBytes, err := json.Marshal(signable)
	if err != nil {
		return nil, fmt.Errorf("canonicalize v1: marshal signable: %w", err)
	}
	if err := json.Unmarshal(signableBytes, &m); err != nil {
		return nil, fmt.Errorf("canonicalize v1: unmarshal signable: %w", err)
	}
	
	// Remove id and created_at (portal signs without these, server adds them later)
	delete(m, "id")
	delete(m, "created_at")
	
	// Encode deterministically
	b, err := CanonicalJSON(m)
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
