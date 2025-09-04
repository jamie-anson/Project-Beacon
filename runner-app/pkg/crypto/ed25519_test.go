package crypto

import (
    "encoding/base64"
    "testing"
)

type sampleSpec struct {
    ID        string `json:"id"`
    Name      string `json:"name"`
    PublicKey string `json:"public_key"`
    Signature string `json:"signature"`
    Extra     map[string]any `json:"extra"`
}

func TestGenerateKeyPair_AndBase64RoundTrip(t *testing.T) {
    kp, err := GenerateKeyPair()
    if err != nil {
        t.Fatalf("GenerateKeyPair error: %v", err)
    }
    if len(kp.PublicKey) == 0 || len(kp.PrivateKey) == 0 {
        t.Fatalf("expected non-empty keys")
    }

    pubB64 := PublicKeyToBase64(kp.PublicKey)
    privB64 := PrivateKeyToBase64(kp.PrivateKey)

    pub2, err := PublicKeyFromBase64(pubB64)
    if err != nil {
        t.Fatalf("PublicKeyFromBase64 error: %v", err)
    }
    priv2, err := PrivateKeyFromBase64(privB64)
    if err != nil {
        t.Fatalf("PrivateKeyFromBase64 error: %v", err)
    }

    if string(pub2) != string(kp.PublicKey) {
        t.Fatalf("public key mismatch after round-trip")
    }
    if string(priv2) != string(kp.PrivateKey) {
        t.Fatalf("private key mismatch after round-trip")
    }
}

func TestBase64Decode_InvalidSizes(t *testing.T) {
    // invalid base64 string
    if _, err := PublicKeyFromBase64("@@@not-base64@@@"); err == nil {
        t.Fatalf("expected error for invalid base64 public key")
    }
    if _, err := PrivateKeyFromBase64("@@@not-base64@@@"); err == nil {
        t.Fatalf("expected error for invalid base64 private key")
    }
    // valid base64 but wrong sizes
    short := base64.StdEncoding.EncodeToString([]byte("short"))
    if _, err := PublicKeyFromBase64(short); err == nil {
        t.Fatalf("expected error for wrong public key size")
    }
    if _, err := PrivateKeyFromBase64(short); err == nil {
        t.Fatalf("expected error for wrong private key size")
    }
}

func TestSignAndVerifyJSON_CanonicalizationAndNegativeCases(t *testing.T) {
    kp, err := GenerateKeyPair()
    if err != nil {
        t.Fatalf("GenerateKeyPair error: %v", err)
    }

    // Maps with different insertion orders should produce identical canonical JSON
    obj1 := map[string]any{"b": 2, "a": 1, "c": []any{3, 2, 1}}
    obj2 := map[string]any{"c": []any{3, 2, 1}, "a": 1, "b": 2}

    sig1, err := SignJSON(obj1, kp.PrivateKey)
    if err != nil {
        t.Fatalf("SignJSON error: %v", err)
    }
    if err := VerifyJSONSignature(obj2, sig1, kp.PublicKey); err != nil {
        t.Fatalf("VerifyJSONSignature failed for same logical data: %v", err)
    }

    // Negative: mutate data
    obj3 := map[string]any{"b": 2, "a": 999, "c": []any{3, 2, 1}}
    if err := VerifyJSONSignature(obj3, sig1, kp.PublicKey); err == nil {
        t.Fatalf("expected verification to fail for mutated data")
    }

    // Negative: use wrong public key
    kp2, _ := GenerateKeyPair()
    if err := VerifyJSONSignature(obj1, sig1, kp2.PublicKey); err == nil {
        t.Fatalf("expected verification to fail with wrong public key")
    }

    // Negative: corrupt signature
    if len(sig1) < 5 {
        t.Fatalf("unexpected short signature encoding")
    }
    badSig := sig1[:len(sig1)-2] + "aa"
    if err := VerifyJSONSignature(obj1, badSig, kp.PublicKey); err == nil {
        t.Fatalf("expected verification to fail with corrupted signature")
    }
}

func TestCreateSignableJobSpec_StructAndMap(t *testing.T) {
    orig := &sampleSpec{
        ID:        "job-1",
        Name:      "BenchmarkFoo",
        PublicKey: "PUB",
        Signature: "SIG",
        Extra:     map[string]any{"z": 1, "a": 2},
    }

    cleaned, err := CreateSignableJobSpec(orig)
    if err != nil {
        t.Fatalf("CreateSignableJobSpec error: %v", err)
    }

    // Signature and PublicKey should be zeroed for struct copy
    cleanedSpec, ok := cleaned.(sampleSpec)
    if !ok {
        t.Fatalf("expected struct copy type")
    }
    if cleanedSpec.Signature != "" || cleanedSpec.PublicKey != "" {
        t.Fatalf("expected signature and public_key to be cleared in struct copy")
    }

    // Map fallback should delete keys
    m := map[string]any{"id": "job-2", "public_key": "PK", "signature": "SIG", "x": 1}
    cleaned2, err := CreateSignableJobSpec(m)
    if err != nil {
        t.Fatalf("CreateSignableJobSpec(map) error: %v", err)
    }
    mm, ok := cleaned2.(map[string]any)
    if !ok {
        t.Fatalf("expected map result for map input")
    }
    if _, exists := mm["public_key"]; exists {
        t.Fatalf("expected public_key removed in map path")
    }
    if _, exists := mm["signature"]; exists {
        t.Fatalf("expected signature removed in map path")
    }
}

func TestCreateSignableReceipt_MapAndStruct(t *testing.T) {
    type receipt struct {
        ID        string `json:"id"`
        PublicKey string `json:"public_key"`
        Signature string `json:"signature"`
        Score     int    `json:"score"`
    }

    r := &receipt{ID: "r1", PublicKey: "PK", Signature: "SIG", Score: 42}

    cleaned, err := CreateSignableReceipt(r)
    if err != nil {
        t.Fatalf("CreateSignableReceipt error: %v", err)
    }
    rr, ok := cleaned.(receipt)
    if !ok {
        t.Fatalf("expected struct copy for receipt")
    }
    if rr.PublicKey != "" || rr.Signature != "" {
        t.Fatalf("expected signature/public_key to be cleared in receipt struct")
    }

    // Map input
    rm := map[string]any{"id": "r2", "public_key": "PK", "signature": "SIG", "score": 99}
    cleaned2, err := CreateSignableReceipt(rm)
    if err != nil {
        t.Fatalf("CreateSignableReceipt(map) error: %v", err)
    }
    mm, ok := cleaned2.(map[string]any)
    if !ok {
        t.Fatalf("expected map result for receipt map input")
    }
    if _, exists := mm["public_key"]; exists {
        t.Fatalf("expected public_key removed in receipt map path")
    }
    if _, exists := mm["signature"]; exists {
        t.Fatalf("expected signature removed in receipt map path")
    }
}
