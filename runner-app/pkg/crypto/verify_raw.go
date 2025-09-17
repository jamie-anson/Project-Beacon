package crypto

import (
    "encoding/base64"
    "fmt"
)

// VerifySignatureRaw verifies an Ed25519 signature against the raw JSON payload.
// It removes the provided keys (e.g., "signature", "public_key") from the JSON object
// before canonicalizing deterministically, then verifies the signature with the
// given base64-encoded public key.
func VerifySignatureRaw(raw []byte, signatureB64 string, publicKeyB64 string, removeKeys []string) error {
    if len(raw) == 0 {
        return fmt.Errorf("empty payload")
    }
    if signatureB64 == "" {
        return fmt.Errorf("missing signature")
    }
    if publicKeyB64 == "" {
        return fmt.Errorf("missing public key")
    }

    // Parse and remove fields
    generic, err := StripKeysFromJSON(raw, removeKeys)
    if err != nil {
        return fmt.Errorf("prepare signable: %w", err)
    }

    // Canonicalize to bytes that must match client-side canonical JSON
    canon, err := CanonicalizeGenericV1(generic)
    if err != nil {
        return fmt.Errorf("canonicalize: %w", err)
    }

    // Decode pubkey
    pk, err := PublicKeyFromBase64(publicKeyB64)
    if err != nil {
        return fmt.Errorf("invalid public key: %w", err)
    }

    // Verify directly against canonical bytes
    if err := VerifySignatureBytes(canon, signatureB64, pk); err != nil {
        // Also try raw (unpadded) base64 signature if needed
        if _, err2 := base64.StdEncoding.DecodeString(signatureB64); err2 != nil {
            if _, err3 := base64.RawStdEncoding.DecodeString(signatureB64); err3 == nil {
                // signature is raw base64url; VerifyJSONSignature already handles StdEncoding
                // Re-run manually with decoded signature path not exposed; return generic error for now
            }
        }
        return fmt.Errorf("signature verification failed: %w", err)
    }
    return nil
}
