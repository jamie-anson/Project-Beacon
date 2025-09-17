package api

import (
    "bytes"
    "crypto/ed25519"
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "encoding/hex"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/jamie-anson/project-beacon-runner/internal/config"
    beaconcrypto "github.com/jamie-anson/project-beacon-runner/pkg/crypto"
)

func TestDebugVerifyEndpoint_ReturnsCanonicalFingerprintsAndVerifiesOK(t *testing.T) {
    gin.SetMode(gin.TestMode)

    // Minimal config with Admin token for auth middleware
    cfg := &config.Config{
        AdminToken:     "test-admin",
        HTTPPort:       ":8090",
        PortStrategy:   "fallback",
        PortRangeStart: 8090,
        PortRangeEnd:   8099,
        JobsQueueName:  "jobs",
    }

    // Build router (no services needed for this endpoint)
    r := SetupRoutes(nil, cfg, nil)

    // Generate a keypair
    pub, priv, err := ed25519.GenerateKey(rand.Reader)
    if err != nil {
        t.Fatalf("keygen: %v", err)
    }
    pubB64 := base64.StdEncoding.EncodeToString(pub)

    // Build a portal-like payload without signature/public_key
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

    // Canonicalize generic and sign to get signature
    canon, err := beaconcrypto.CanonicalizeGenericV1(payload)
    if err != nil {
        t.Fatalf("canonicalize: %v", err)
    }
    sig := ed25519.Sign(priv, canon)
    sigB64 := base64.StdEncoding.EncodeToString(sig)

    // Add signature/public_key to payload and marshal raw
    payload["public_key"] = pubB64
    payload["signature"] = sigB64
    raw, err := json.Marshal(payload)
    if err != nil {
        t.Fatalf("marshal: %v", err)
    }

    // Prepare request
    req := httptest.NewRequest(http.MethodPost, "/api/v1/debug/verify", bytes.NewReader(raw))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+cfg.AdminToken)

    // Execute
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
    }

    // Parse response
    var resp struct {
        ServerCanonicalLen     int    `json:"server_canonical_len"`
        ServerCanonicalSHA256  string `json:"server_canonical_sha256"`
        HasExecutionType       bool   `json:"has_execution_type"`
        HasEstimatedCost       bool   `json:"has_estimated_cost"`
        HasCreatedAt           bool   `json:"has_created_at"`
        Verify                 string `json:"verify"`
    }
    if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
        t.Fatalf("unmarshal response: %v", err)
    }

    // Compute expected
    sum := sha256.Sum256(canon)
    expectedSHA := hex.EncodeToString(sum[:])

    if resp.ServerCanonicalLen != len(canon) {
        t.Fatalf("server_canonical_len mismatch: got %d, want %d", resp.ServerCanonicalLen, len(canon))
    }
    if resp.ServerCanonicalSHA256 != expectedSHA {
        t.Fatalf("server_canonical_sha256 mismatch: got %s, want %s", resp.ServerCanonicalSHA256, expectedSHA)
    }
    if resp.Verify != "ok" {
        t.Fatalf("expected verify=ok, got %s", resp.Verify)
    }
    if !resp.HasExecutionType || !resp.HasEstimatedCost || !resp.HasCreatedAt {
        t.Fatalf("expected field presence flags to be true, got exec=%v est=%v created_at=%v", resp.HasExecutionType, resp.HasEstimatedCost, resp.HasCreatedAt)
    }
}
