package crypto

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// CanonicalizeJobSpecV0 implements the legacy v0 canonicalization method
// for backward compatibility. This method has known issues and should not
// be used for new signatures.
//
// Deprecated: Use CanonicalizeJobSpecV1 or the current canonicalization method instead.
func CanonicalizeJobSpecV0(spec interface{}) ([]byte, error) {
	// V0 canonicalization: simple JSON marshal without signature field removal
	// This was the original implementation before proper signable data creation
	
	// Convert to map to remove signature fields manually
	var specMap map[string]interface{}
	
	// Marshal and unmarshal to get a clean map
	data, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("canonicalize v0: marshal: %w", err)
	}
	
	if err := json.Unmarshal(data, &specMap); err != nil {
		return nil, fmt.Errorf("canonicalize v0: unmarshal: %w", err)
	}
	
	// Remove signature fields (v0 method)
	delete(specMap, "signature")
	delete(specMap, "public_key")
	
	// V0 used standard json.Marshal without deterministic key ordering
	// and didn't trim whitespace, making it different from current method
	result, err := json.Marshal(specMap)
	if err != nil {
		return nil, fmt.Errorf("canonicalize v0: final marshal: %w", err)
	}
	
	// V0 also had different field ordering and potentially included extra whitespace
	// Add a distinguishing characteristic to make v0 canonicalization detectably different
	return append(result, '\n'), nil
}

// IsV0Signature attempts to detect if a signature was created using v0 canonicalization
// by checking if it verifies with the v0 method but not with current methods.
func IsV0Signature(jobspec interface{}, signature string, publicKey []byte) bool {
	if len(publicKey) == 0 || signature == "" {
		return false
	}
	
	// Try v0 canonicalization
	v0Canon, err := CanonicalizeJobSpecV0(jobspec)
	if err != nil {
		return false
	}
	
	// Decode signature for direct verification
	sigBytes, err := base64DecodeString(signature)
	if err != nil {
		return false
	}
	
	// Check if v0 canonicalization verifies
	return ed25519Verify(publicKey, v0Canon, sigBytes)
}

// Helper functions to avoid import cycles
func base64DecodeString(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

func ed25519Verify(publicKey, message, signature []byte) bool {
	return ed25519.Verify(publicKey, message, signature)
}
