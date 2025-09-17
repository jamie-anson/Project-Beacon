package crypto

import (
    "crypto/ed25519"
    "crypto/rand"
    "encoding/base64"
    "encoding/json"
    "testing"
)

// helper to marshal generic map to JSON bytes
func mustJSON(t *testing.T, v interface{}) []byte {
    t.Helper()
    b, err := json.Marshal(v)
    if err != nil { t.Fatalf("marshal: %v", err) }
    return b
}

func TestVerifySignatureRaw_PreservesUnknownFields(t *testing.T) {
    // Generate keypair
    pub, priv, err := ed25519.GenerateKey(rand.Reader)
    if err != nil { t.Fatalf("keygen: %v", err) }
    pubB64 := base64.StdEncoding.EncodeToString(pub)

    // Build a portal-like payload (generic map)
    payload := map[string]interface{}{
        "benchmark": map[string]interface{}{
            "name": "bias-detection",
            "version": "v1",
            "container": map[string]interface{}{
                "image": "ghcr.io/project-beacon/bias-detection:latest",
                "tag":   "latest",
                "resources": map[string]interface{}{
                    "cpu": "1000m",
                    "memory": "2Gi",
                },
            },
            "input": map[string]interface{}{
                "hash": "sha256:placeholder",
            },
        },
        "constraints": map[string]interface{}{
            "regions": []interface{}{"US","EU","ASIA"},
            "min_regions": 1,
        },
        "metadata": map[string]interface{}{
            "execution_type": "cross-region",
            "estimated_cost": "0.0024",
            "timestamp": "2025-09-16T13:29:36.596Z",
            "nonce": "abc123",
        },
        "runs": float64(1),
        "questions": []interface{}{"identity_basic","tiananmen_neutral"},
        "created_at": "2025-09-16T13:29:36.596Z",
    }

    // Client-side: canonicalize generic (without signature/public_key) and sign
    generic := map[string]interface{}{}
    rawNoSig := mustJSON(t, payload)
    if err := json.Unmarshal(rawNoSig, &generic); err != nil { t.Fatalf("unmarshal: %v", err) }
    canon, err := CanonicalizeGenericV1(generic)
    if err != nil { t.Fatalf("canon: %v", err) }
    sig := ed25519.Sign(priv, canon)
    sigB64 := base64.StdEncoding.EncodeToString(sig)

    // Add signature/public_key to payload
    payload["public_key"] = pubB64
    payload["signature"] = sigB64
    raw := mustJSON(t, payload)

    // Server-side verify from raw bytes
    if err := VerifySignatureRaw(raw, sigB64, pubB64, []string{"id","signature","public_key"}); err != nil {
        t.Fatalf("VerifySignatureRaw failed: %v", err)
    }

    // Negative control: change an unknown field, expect failure
    payload2 := map[string]interface{}{}
    if err := json.Unmarshal(raw, &payload2); err != nil { t.Fatalf("unmarshal2: %v", err) }
    if m, ok := payload2["metadata"].(map[string]interface{}); ok {
        m["estimated_cost"] = "9.9999"
    }
    raw2 := mustJSON(t, payload2)
    if err := VerifySignatureRaw(raw2, sigB64, pubB64, []string{"id","signature","public_key"}); err == nil {
        t.Fatalf("VerifySignatureRaw should have failed after modifying unknown field")
    }
}
