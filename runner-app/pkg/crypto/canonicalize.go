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
	
	// Remove null and empty values to match portal's JavaScript JSON.stringify behavior
	// JavaScript doesn't include undefined/null fields, Go includes them
	removeNullAndEmptyValues(m)
	
	// Encode deterministically
	b, err := CanonicalJSON(m)
	if err != nil {
		return nil, fmt.Errorf("canonicalize v1: encode: %w", err)
	}
	// Avoid accidental trailing spaces/newlines
	return bytes.TrimSpace(b), nil
}

// removeNullAndEmptyValues recursively removes null values and empty strings from a map
// This matches JavaScript's JSON.stringify behavior where undefined fields are omitted
// EXCEPT for signature and public_key which must be retained (zeroed) for deterministic structure
func removeNullAndEmptyValues(m map[string]interface{}) {
	for k, v := range m {
		// Never remove signature or public_key fields (they must be present, even if empty)
		if k == "signature" || k == "public_key" {
			continue
		}
		
		switch val := v.(type) {
		case map[string]interface{}:
			removeNullAndEmptyValues(val)
			// Remove if empty after recursive cleanup
			if len(val) == 0 {
				delete(m, k)
			}
		case string:
			// Remove empty strings
			if val == "" {
				delete(m, k)
			}
		case nil:
			// Remove null values
			delete(m, k)
		case float64:
			// Remove zero values for numeric fields (matches omitempty behavior)
			if val == 0 {
				delete(m, k)
			}
		}
	}
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
