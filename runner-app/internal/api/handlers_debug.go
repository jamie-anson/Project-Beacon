package api

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "net/http"

    "github.com/gin-gonic/gin"
    beaconcrypto "github.com/jamie-anson/project-beacon-runner/pkg/crypto"
)

// DebugVerify computes server-side canonicalization fingerprints for a submitted JSON body.
// Admin-only. Returns canonical length, sha256, field presence booleans, and optional verify result.
func DebugVerify(c *gin.Context) {
    raw, err := c.GetRawData()
    if err != nil || len(raw) == 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_body"})
        return
    }

    // Parse JSON for field presence and signature/public key extraction
    var obj map[string]interface{}
    _ = json.Unmarshal(raw, &obj)

    // Presence checks
    hasExecType := false
    hasEstCost := false
    hasCreatedAt := false
    if m, ok := obj["metadata"].(map[string]interface{}); ok {
        if _, ok2 := m["execution_type"]; ok2 { hasExecType = true }
        if _, ok2 := m["estimated_cost"]; ok2 { hasEstCost = true }
    }
    if _, ok := obj["created_at"]; ok { hasCreatedAt = true }

    // Strip keys and canonicalize
    generic, err := beaconcrypto.StripKeysFromJSON(raw, []string{"id", "signature", "public_key"})
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json", "details": err.Error()})
        return
    }
    canon, err := beaconcrypto.CanonicalizeGenericV1(generic)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "canonicalize_failed", "details": err.Error()})
        return
    }
    sum := sha256.Sum256(canon)

    // Attempt verification if signature and public_key are present
    verifyResult := "skipped"
    if sigV, ok := obj["signature"].(string); ok && sigV != "" {
        if pubV, ok2 := obj["public_key"].(string); ok2 && pubV != "" {
            if err := beaconcrypto.VerifySignatureRaw(raw, sigV, pubV, []string{"id", "signature", "public_key"}); err == nil {
                verifyResult = "ok"
            } else {
                verifyResult = "failed"
            }
        }
    }

    c.JSON(http.StatusOK, gin.H{
        "server_canonical_len": len(canon),
        "server_canonical_sha256": hex.EncodeToString(sum[:]),
        "has_execution_type": hasExecType,
        "has_estimated_cost": hasEstCost,
        "has_created_at": hasCreatedAt,
        "verify": verifyResult,
    })
}
