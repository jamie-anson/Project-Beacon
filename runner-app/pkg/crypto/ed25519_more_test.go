package crypto

import (
    "testing"
)

func TestCanonicalJSON_DeterminismAndOrdering(t *testing.T) {
    a := map[string]any{"b": 2, "a": 1, "c": []any{3, 2, 1}}
    b := map[string]any{"c": []any{3, 2, 1}, "a": 1, "b": 2}

    ba1, err := CanonicalJSON(a)
    if err != nil {
        t.Fatalf("CanonicalJSON(a) error: %v", err)
    }
    bb1, err := CanonicalJSON(b)
    if err != nil {
        t.Fatalf("CanonicalJSON(b) error: %v", err)
    }
    if string(ba1) != string(bb1) {
        t.Fatalf("expected canonical bytes equal, got %q vs %q", string(ba1), string(bb1))
    }
    // Deterministic across repeated calls
    ba2, _ := CanonicalJSON(a)
    bb2, _ := CanonicalJSON(b)
    if string(ba1) != string(ba2) || string(bb1) != string(bb2) {
        t.Fatalf("expected deterministic output across calls")
    }
}

func TestSignJSON_NonSerializable_ReturnsError(t *testing.T) {
    // Channels are not JSON-serializable; encoding/json should error
    v := map[string]any{"ch": make(chan int)}
    kp, err := GenerateKeyPair()
    if err != nil {
        t.Fatalf("GenerateKeyPair error: %v", err)
    }
    if _, err := SignJSON(v, kp.PrivateKey); err == nil {
        t.Fatalf("expected error when signing non-serializable value")
    }
}

func TestVerifyJSONSignature_InvalidBase64(t *testing.T) {
    kp, err := GenerateKeyPair()
    if err != nil {
        t.Fatalf("GenerateKeyPair error: %v", err)
    }
    data := map[string]any{"x": 1}
    // Invalid base64 signature string
    if err := VerifyJSONSignature(data, "@@not-base64@@", kp.PublicKey); err == nil {
        t.Fatalf("expected error on invalid base64 signature input")
    }
}
