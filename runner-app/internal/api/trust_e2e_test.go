package api

import (
    "context"
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "net/http/httptest"
    "os"
    "path/filepath"
    "testing"
    "time"

    sqlmock "github.com/DATA-DOG/go-sqlmock"
    miniredis "github.com/alicebob/miniredis/v2"
    "github.com/jamie-anson/project-beacon-runner/internal/config"
    "github.com/jamie-anson/project-beacon-runner/internal/service"
    "github.com/jamie-anson/project-beacon-runner/pkg/crypto"
    "github.com/jamie-anson/project-beacon-runner/pkg/models"
    "github.com/redis/go-redis/v9"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// Helper to spin up a test HTTP server with given cfg and redis
func setupTestServer(t *testing.T, cfg *config.Config) (*httptest.Server, sqlmock.Sqlmock, func()) {
    t.Helper()

    // In-memory Redis
    mr, err := miniredis.Run()
    require.NoError(t, err)
    redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})

    // sqlmock-backed service
    sqlDB, mock, err := sqlmock.New()
    require.NoError(t, err)
    jobsSvc := service.NewJobsService(sqlDB)

    router := SetupRoutes(jobsSvc, cfg, redisClient)
    server := httptest.NewServer(router)

    cleanup := func() {
        server.Close()
        sqlDB.Close()
        mr.Close()
        require.NoError(t, mock.ExpectationsWereMet())
    }
    return server, mock, cleanup
}

// Verifies that StartTrustedKeysReloader's ticker applies file changes without manual ResetTrustedKeysCache.
func TestE2E_TrustedKeys_HotReload_Ticker(t *testing.T) {
    // Generate keypair and initial trusted keys including our key as active
    kp, err := crypto.GenerateKeyPair()
    require.NoError(t, err)
    pub := crypto.PublicKeyToBase64(kp.PublicKey)

    dir := t.TempDir()
    active := "[\n  {\n    \"kid\": \"hotreload-ticker\",\n    \"public_key\": \"" + pub + "\",\n    \"status\": \"active\"\n  }\n]"
    tkPath := writeTrustedKeysFile(t, dir, active)

    // Point env to our file; do NOT call ResetTrustedKeysCache()
    os.Setenv("TRUSTED_KEYS_FILE", tkPath)
    defer os.Unsetenv("TRUSTED_KEYS_FILE")

    // Start the periodic reloader with a short interval
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    config.StartTrustedKeysReloader(ctx, tkPath, 200*time.Millisecond)

    // Configure server
    cfg := &config.Config{ReplayProtectionEnabled: true, TimestampMaxSkew: 5 * time.Minute, TimestampMaxAge: 10 * time.Minute, TrustEnforce: true}
    server, mock, cleanup := setupTestServer(t, cfg)
    defer cleanup()

    // Prepare a valid spec and sign it
    js := &models.JobSpec{
        ID:      "hotreload-ticker-accept",
        Version: "1.0",
        Benchmark: models.BenchmarkSpec{
            Name: "HotReload Ticker Test", Container: models.ContainerSpec{Image: "test/image:latest", Resources: models.ResourceSpec{CPU: "100m", Memory: "128Mi"}},
            Input: models.InputSpec{Type: "text", Data: map[string]interface{}{"prompt": "hi"}, Hash: "h"},
            Scoring: models.ScoringSpec{Method: "none", Parameters: map[string]interface{}{}},
        },
        Constraints: models.ExecutionConstraints{Regions: []string{"US"}},
        Metadata: map[string]interface{}{"timestamp": time.Now().UTC().Format(time.RFC3339), "nonce": "nonce-hrt-1"},
    }
    require.NoError(t, js.Sign(kp.PrivateKey))
    payload, err := json.Marshal(js)
    require.NoError(t, err)

    // Expect DB interactions for the accepted first submit
    mock.ExpectBegin()
    mock.ExpectExec("INSERT INTO jobs").WithArgs(js.ID, sqlmock.AnyArg(), "created").WillReturnResult(sqlmock.NewResult(0, 1))
    mock.ExpectExec("INSERT INTO outbox").WithArgs("jobs", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
    mock.ExpectCommit()
    resp1, err := http.Post(server.URL+"/api/v1/jobs", "application/json", bytes.NewBuffer(payload))
    require.NoError(t, err)
    defer resp1.Body.Close()
    assert.Equal(t, http.StatusAccepted, resp1.StatusCode)

    // Now revoke the key on disk; do NOT call ResetTrustedKeysCache()
    revoked := "[\n  {\n    \"kid\": \"hotreload-ticker\",\n    \"public_key\": \"" + pub + "\",\n    \"status\": \"revoked\"\n  }\n]"
    require.NoError(t, os.WriteFile(tkPath, []byte(revoked), 0o644))

    // Wait for at least one ticker interval to ensure reload occurred
    time.Sleep(500 * time.Millisecond)

    // New request with new nonce should now be rejected due to revocation
    js.Metadata["nonce"] = "nonce-hrt-2"
    payload2, err := json.Marshal(js)
    require.NoError(t, err)
    resp2, err := http.Post(server.URL+"/api/v1/jobs", "application/json", bytes.NewBuffer(payload2))
    require.NoError(t, err)
    defer resp2.Body.Close()
    assert.Equal(t, http.StatusBadRequest, resp2.StatusCode)
    var body map[string]interface{}
    require.NoError(t, json.NewDecoder(resp2.Body).Decode(&body))
    assert.Equal(t, "trust_violation:revoked", body["error_code"])
}

// E2E tests begin here
// -------------------------------------------------
// Malformed timestamp format should yield timestamp_invalid with details.reason = format_invalid
// when trust enforcement is enabled.
func TestE2E_TimestampValidation_MalformedFormat(t *testing.T) {
    // Generate keypair and trusted keys entry
    kp, err := crypto.GenerateKeyPair()
    require.NoError(t, err)
    pub := crypto.PublicKeyToBase64(kp.PublicKey)

    dir := t.TempDir()
    tk := "[\n  {\n    \"kid\": \"dev-ts-badfmt\",\n    \"public_key\": \"" + pub + "\",\n    \"status\": \"active\",\n    \"not_before\": \"2025-01-01T00:00:00Z\",\n    \"not_after\": \"2026-12-31T23:59:59Z\"\n  }\n]"
    tkPath := writeTrustedKeysFile(t, dir, tk)

    os.Setenv("TRUSTED_KEYS_FILE", tkPath)
    config.ResetTrustedKeysCache()

    cfg := &config.Config{
        ReplayProtectionEnabled: true,
        TimestampMaxSkew:        5 * time.Minute,
        TimestampMaxAge:         10 * time.Minute,
        TrustEnforce:            true,
    }

    server, _, cleanup := setupTestServer(t, cfg)
    defer cleanup()

    // Malformed timestamp (wrong format)
    badTS := "2025/08/22 14:50:32"
    js := &models.JobSpec{
        ID:      "ts-badfmt",
        Version: "1.0",
        Benchmark: models.BenchmarkSpec{
            Name: "TS BadFmt",
            Container: models.ContainerSpec{Image: "test/image:latest", Resources: models.ResourceSpec{CPU: "100m", Memory: "128Mi"}},
            Input:  models.InputSpec{Type: "text", Data: map[string]interface{}{"prompt": "hi"}, Hash: "h"},
            Scoring: models.ScoringSpec{Method: "none", Parameters: map[string]interface{}{}},
        },
        Constraints: models.ExecutionConstraints{Regions: []string{"US"}},
        Metadata: map[string]interface{}{
            "timestamp": badTS,
            "nonce":     "nonce-badfmt",
        },
    }
    require.NoError(t, js.Sign(kp.PrivateKey))
    payload, err := json.Marshal(js)
    require.NoError(t, err)

    resp, err := http.Post(server.URL+"/api/v1/jobs", "application/json", bytes.NewBuffer(payload))
    require.NoError(t, err)
    defer resp.Body.Close()

    assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
    var body map[string]interface{}
    require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
    assert.Equal(t, "timestamp_invalid", body["error_code"])
    // details.reason should be format_invalid
    if details, ok := body["details"].(map[string]interface{}); ok {
        assert.Equal(t, "format_invalid", details["reason"])
    } else {
        t.Fatalf("missing details field in response: %#v", body)
    }
}
// Rate limiting E2E: after 5 signature failures per key within 1 minute,
// the 6th request should be rejected with HTTP 429 and error_code rate_limit_exceeded.
func TestE2E_RateLimiting_SignatureFailures_Returns429AfterThreshold(t *testing.T) {
    // Generate keypair and trusted keys entry
    kp, err := crypto.GenerateKeyPair()
    require.NoError(t, err)
    pub := crypto.PublicKeyToBase64(kp.PublicKey)

    dir := t.TempDir()
    tk := "[\n  {\n    \"kid\": \"dev-rate-limit\",\n    \"public_key\": \"" + pub + "\",\n    \"status\": \"active\",\n    \"not_before\": \"2025-01-01T00:00:00Z\",\n    \"not_after\": \"2026-12-31T23:59:59Z\"\n  }\n]"
    tkPath := writeTrustedKeysFile(t, dir, tk)

    os.Setenv("TRUSTED_KEYS_FILE", tkPath)
    config.ResetTrustedKeysCache()

    cfg := &config.Config{
        ReplayProtectionEnabled: true,
        TimestampMaxSkew:        5 * time.Minute,
        TimestampMaxAge:         10 * time.Minute,
        TrustEnforce:            true,
    }

    server, _, cleanup := setupTestServer(t, cfg)
    defer cleanup()

    // Build a base valid spec and sign it
    base := &models.JobSpec{
        ID:      "rate-limit-test",
        Version: "1.0",
        Benchmark: models.BenchmarkSpec{
            Name: "Rate Limit Test",
            Container: models.ContainerSpec{Image: "test/image:latest", Resources: models.ResourceSpec{CPU: "100m", Memory: "128Mi"}},
            Input:  models.InputSpec{Type: "text", Data: map[string]interface{}{"prompt": "hi"}, Hash: "h"},
            Scoring: models.ScoringSpec{Method: "none", Parameters: map[string]interface{}{}},
        },
        Constraints: models.ExecutionConstraints{Regions: []string{"US"}},
        Metadata: map[string]interface{}{
            "timestamp": time.Now().UTC().Format(time.RFC3339),
            "nonce":     "nonce-0",
        },
    }
    require.NoError(t, base.Sign(kp.PrivateKey))

    // Send 5 requests that each produce signature_mismatch and record failures
    for i := 1; i <= 5; i++ {
        js := *base // shallow copy
        // mutate fields AFTER signing so signature becomes invalid
        js.Benchmark.Name = "Rate Limit Test - Tampered"
        // ensure unique nonce to avoid replay protection
        js.Metadata = map[string]interface{}{
            "timestamp": time.Now().UTC().Format(time.RFC3339),
            "nonce":     fmt.Sprintf("nonce-%d", i),
        }
        payload, err := json.Marshal(&js)
        require.NoError(t, err)

        resp, err := http.Post(server.URL+"/api/v1/jobs", "application/json", bytes.NewBuffer(payload))
        require.NoError(t, err)
        defer resp.Body.Close()

        // Should be a validation failure before rate limit trips
        assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
        var body map[string]interface{}
        require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
        assert.Equal(t, "signature_mismatch", body["error_code"])
    }

    // 6th request should be blocked by rate limiter with 429
    js := *base
    js.Benchmark.Name = "Rate Limit Test - Tampered Final"
    js.Metadata = map[string]interface{}{
        "timestamp": time.Now().UTC().Format(time.RFC3339),
        "nonce":     "nonce-6",
    }
    payload, err := json.Marshal(&js)
    require.NoError(t, err)

    resp, err := http.Post(server.URL+"/api/v1/jobs", "application/json", bytes.NewBuffer(payload))
    require.NoError(t, err)
    defer resp.Body.Close()

    assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
    var body map[string]interface{}
    require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
    assert.Equal(t, "rate_limit_exceeded", body["error_code"])
}

func TestE2E_ReplayProtection_RejectsDuplicateNonce(t *testing.T) {
    // Generate keypair and trusted keys entry
    kp, err := crypto.GenerateKeyPair()
    require.NoError(t, err)
    pub := crypto.PublicKeyToBase64(kp.PublicKey)

    dir := t.TempDir()
    tk := "[\n  {\n    \"kid\": \"dev-replay\",\n    \"public_key\": \"" + pub + "\",\n    \"status\": \"active\",\n    \"not_before\": \"2025-01-01T00:00:00Z\",\n    \"not_after\": \"2026-12-31T23:59:59Z\"\n  }\n]"
    tkPath := writeTrustedKeysFile(t, dir, tk)

    os.Setenv("TRUSTED_KEYS_FILE", tkPath)
    config.ResetTrustedKeysCache()

    cfg := &config.Config{
        ReplayProtectionEnabled: true,
        TimestampMaxSkew:        5 * time.Minute,
        TimestampMaxAge:         10 * time.Minute,
        TrustEnforce:            true,
    }

    server, mock, cleanup := setupTestServer(t, cfg)
    defer cleanup()

    // Prepare signed spec with a fixed nonce
    js := &models.JobSpec{
        ID:      "replay-test",
        Version: "1.0",
        Benchmark: models.BenchmarkSpec{
            Name: "Replay Test",
            Container: models.ContainerSpec{Image: "test/image:latest", Resources: models.ResourceSpec{CPU: "100m", Memory: "128Mi"}},
            Input:  models.InputSpec{Type: "text", Data: map[string]interface{}{"prompt": "hi"}, Hash: "h"},
            Scoring: models.ScoringSpec{Method: "none", Parameters: map[string]interface{}{}},
        },
        Constraints: models.ExecutionConstraints{Regions: []string{"US"}},
        Metadata: map[string]interface{}{
            "timestamp": time.Now().UTC().Format(time.RFC3339),
            "nonce":     "fixed-nonce-123",
        },
    }
    require.NoError(t, js.Sign(kp.PrivateKey))
    payload, err := json.Marshal(js)
    require.NoError(t, err)

    // First request should be accepted and hit DB
    mock.ExpectBegin()
    mock.ExpectExec("INSERT INTO jobs").WithArgs(js.ID, sqlmock.AnyArg(), "created").WillReturnResult(sqlmock.NewResult(0, 1))
    mock.ExpectExec("INSERT INTO outbox").WithArgs("jobs", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
    mock.ExpectCommit()

    resp1, err := http.Post(server.URL+"/api/v1/jobs", "application/json", bytes.NewBuffer(payload))
    require.NoError(t, err)
    defer resp1.Body.Close()
    assert.Equal(t, http.StatusAccepted, resp1.StatusCode)

    // Second request with same payload (same nonce) should be rejected due to replay
    resp2, err := http.Post(server.URL+"/api/v1/jobs", "application/json", bytes.NewBuffer(payload))
    require.NoError(t, err)
    defer resp2.Body.Close()
    assert.Equal(t, http.StatusBadRequest, resp2.StatusCode)

    var body map[string]interface{}
    require.NoError(t, json.NewDecoder(resp2.Body).Decode(&body))
    assert.Equal(t, "replay_detected", body["error_code"]) // from handlers_simple.go
}

func TestE2E_TimestampValidation_Invalid(t *testing.T) {
    // Generate keypair and trusted keys entry
    kp, err := crypto.GenerateKeyPair()
    require.NoError(t, err)
    pub := crypto.PublicKeyToBase64(kp.PublicKey)

    dir := t.TempDir()
    tk := "[\n  {\n    \"kid\": \"dev-ts\",\n    \"public_key\": \"" + pub + "\",\n    \"status\": \"active\",\n    \"not_before\": \"2025-01-01T00:00:00Z\",\n    \"not_after\": \"2026-12-31T23:59:59Z\"\n  }\n]"
    tkPath := writeTrustedKeysFile(t, dir, tk)

    os.Setenv("TRUSTED_KEYS_FILE", tkPath)
    config.ResetTrustedKeysCache()

    // Configure short age/skew windows to ensure timestamp invalidation
    cfg := &config.Config{
        ReplayProtectionEnabled: true,
        TimestampMaxSkew:        1 * time.Minute,
        TimestampMaxAge:         2 * time.Minute,
        TrustEnforce:            true,
    }

    server, _, cleanup := setupTestServer(t, cfg)
    defer cleanup()

    // Prepare spec with timestamp too old beyond max age
    oldTS := time.Now().UTC().Add(-10 * time.Minute).Format(time.RFC3339)
    js := &models.JobSpec{
        ID:      "ts-invalid",
        Version: "1.0",
        Benchmark: models.BenchmarkSpec{
            Name: "TS Test",
            Container: models.ContainerSpec{Image: "test/image:latest", Resources: models.ResourceSpec{CPU: "100m", Memory: "128Mi"}},
            Input:  models.InputSpec{Type: "text", Data: map[string]interface{}{"prompt": "hi"}, Hash: "h"},
            Scoring: models.ScoringSpec{Method: "none", Parameters: map[string]interface{}{}},
        },
        Constraints: models.ExecutionConstraints{Regions: []string{"US"}},
        Metadata: map[string]interface{}{
            "timestamp": oldTS,
            "nonce":     "nonce-ts-invalid",
        },
    }
    require.NoError(t, js.Sign(kp.PrivateKey))
    payload, err := json.Marshal(js)
    require.NoError(t, err)

    resp, err := http.Post(server.URL+"/api/v1/jobs", "application/json", bytes.NewBuffer(payload))
    require.NoError(t, err)
    defer resp.Body.Close()

    assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
    var body map[string]interface{}
    require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
    assert.Equal(t, "timestamp_invalid", body["error_code"]) // from handlers_simple.go
}

func TestE2E_TimestampValidation_FutureBeyondSkew(t *testing.T) {
    // Generate keypair and trusted keys entry
    kp, err := crypto.GenerateKeyPair()
    require.NoError(t, err)
    pub := crypto.PublicKeyToBase64(kp.PublicKey)

    dir := t.TempDir()
    tk := "[\n  {\n    \"kid\": \"dev-ts-future\",\n    \"public_key\": \"" + pub + "\",\n    \"status\": \"active\",\n    \"not_before\": \"2025-01-01T00:00:00Z\",\n    \"not_after\": \"2026-12-31T23:59:59Z\"\n  }\n]"
    tkPath := writeTrustedKeysFile(t, dir, tk)

    os.Setenv("TRUSTED_KEYS_FILE", tkPath)
    config.ResetTrustedKeysCache()

    // Configure very small skew to force failure on future timestamp
    cfg := &config.Config{
        ReplayProtectionEnabled: true,
        TimestampMaxSkew:        1 * time.Minute,
        TimestampMaxAge:         10 * time.Minute,
        TrustEnforce:            true,
    }

    server, _, cleanup := setupTestServer(t, cfg)
    defer cleanup()

    // Prepare spec with timestamp too far in the future beyond skew
    futureTS := time.Now().UTC().Add(10 * time.Minute).Format(time.RFC3339)
    js := &models.JobSpec{
        ID:      "ts-future",
        Version: "1.0",
        Benchmark: models.BenchmarkSpec{
            Name: "TS Future Test",
            Container: models.ContainerSpec{Image: "test/image:latest", Resources: models.ResourceSpec{CPU: "100m", Memory: "128Mi"}},
            Input:  models.InputSpec{Type: "text", Data: map[string]interface{}{"prompt": "hi"}, Hash: "h"},
            Scoring: models.ScoringSpec{Method: "none", Parameters: map[string]interface{}{}},
        },
        Constraints: models.ExecutionConstraints{Regions: []string{"US"}},
        Metadata: map[string]interface{}{
            "timestamp": futureTS,
            "nonce":     "nonce-ts-future",
        },
    }
    require.NoError(t, js.Sign(kp.PrivateKey))
    payload, err := json.Marshal(js)
    require.NoError(t, err)

    resp, err := http.Post(server.URL+"/api/v1/jobs", "application/json", bytes.NewBuffer(payload))
    require.NoError(t, err)
    defer resp.Body.Close()

    assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
    var body map[string]interface{}
    require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
    assert.Equal(t, "timestamp_invalid", body["error_code"]) // future beyond skew
}

func TestE2E_InvalidSignature_Rejected(t *testing.T) {
    // Generate keypair and trusted keys entry
    kp, err := crypto.GenerateKeyPair()
    require.NoError(t, err)
    pub := crypto.PublicKeyToBase64(kp.PublicKey)

    dir := t.TempDir()
    tk := "[\n  {\n    \"kid\": \"dev-sig\",\n    \"public_key\": \"" + pub + "\",\n    \"status\": \"active\",\n    \"not_before\": \"2025-01-01T00:00:00Z\",\n    \"not_after\": \"2026-12-31T23:59:59Z\"\n  }\n]"
    tkPath := writeTrustedKeysFile(t, dir, tk)

    os.Setenv("TRUSTED_KEYS_FILE", tkPath)
    config.ResetTrustedKeysCache()

    cfg := &config.Config{
        ReplayProtectionEnabled: true,
        TimestampMaxSkew:        5 * time.Minute,
        TimestampMaxAge:         10 * time.Minute,
        TrustEnforce:            true,
    }

    server, _, cleanup := setupTestServer(t, cfg)
    defer cleanup()

    // Prepare a valid spec and sign it
    js := &models.JobSpec{
        ID:      "sig-invalid",
        Version: "1.0",
        Benchmark: models.BenchmarkSpec{
            Name: "Sig Test",
            Container: models.ContainerSpec{Image: "test/image:latest", Resources: models.ResourceSpec{CPU: "100m", Memory: "128Mi"}},
            Input:  models.InputSpec{Type: "text", Data: map[string]interface{}{"prompt": "hi"}, Hash: "h"},
            Scoring: models.ScoringSpec{Method: "none", Parameters: map[string]interface{}{}},
        },
        Constraints: models.ExecutionConstraints{Regions: []string{"US"}},
        Metadata: map[string]interface{}{
            "timestamp": time.Now().UTC().Format(time.RFC3339),
            "nonce":     "nonce-sig-invalid",
        },
    }
    require.NoError(t, js.Sign(kp.PrivateKey))

    // Tamper with content after signing to force signature mismatch
    js.Benchmark.Name = "Sig Test - Tampered"

    payload, err := json.Marshal(js)
    require.NoError(t, err)

    resp, err := http.Post(server.URL+"/api/v1/jobs", "application/json", bytes.NewBuffer(payload))
    require.NoError(t, err)
    defer resp.Body.Close()

    assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
    var body map[string]interface{}
    require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
    assert.Equal(t, "signature_mismatch", body["error_code"]) // from handlers_simple.go
}

func TestE2E_SigBypass_AllowsMissingSignatureAndPublicKey(t *testing.T) {
    // Configure handler with signature bypass enabled
    cfg := &config.Config{
        ReplayProtectionEnabled: true,
        TimestampMaxSkew:        5 * time.Minute,
        TimestampMaxAge:         10 * time.Minute,
        TrustEnforce:            true,  // still on, but trust check only runs when PublicKey != ""
        SigBypass:               true,  // bypass signature verification
    }

    server, mock, cleanup := setupTestServer(t, cfg)
    defer cleanup()

    // Build a minimal JobSpec WITHOUT signature and public key
    js := &models.JobSpec{
        ID:      "bypass-accept",
        Version: "1.0",
        Benchmark: models.BenchmarkSpec{
            Name: "Bypass Test",
            Container: models.ContainerSpec{Image: "test/image:latest", Resources: models.ResourceSpec{CPU: "100m", Memory: "128Mi"}},
            Input:  models.InputSpec{Type: "text", Data: map[string]interface{}{"prompt": "hi"}, Hash: "h"},
            Scoring: models.ScoringSpec{Method: "none", Parameters: map[string]interface{}{}},
        },
        Constraints: models.ExecutionConstraints{Regions: []string{"US"}},
        Metadata: map[string]interface{}{
            "timestamp": time.Now().UTC().Format(time.RFC3339),
            "nonce":     "nonce-bypass",
        },
        // No PublicKey, No Signature
    }
    payload, err := json.Marshal(js)
    require.NoError(t, err)

    // Expect DB interactions since it should be accepted due to bypass
    mock.ExpectBegin()
    mock.ExpectExec("INSERT INTO jobs").WithArgs(js.ID, sqlmock.AnyArg(), "created").WillReturnResult(sqlmock.NewResult(0, 1))
    mock.ExpectExec("INSERT INTO outbox").WithArgs("jobs", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
    mock.ExpectCommit()

    resp, err := http.Post(server.URL+"/api/v1/jobs", "application/json", bytes.NewBuffer(payload))
    require.NoError(t, err)
    defer resp.Body.Close()
    assert.Equal(t, http.StatusAccepted, resp.StatusCode)
}

func TestE2E_TrustEnforcement_ExpiredRejected(t *testing.T) {
    kp, err := crypto.GenerateKeyPair()
    require.NoError(t, err)
    pub := crypto.PublicKeyToBase64(kp.PublicKey)

    dir := t.TempDir()
    // not_after in the past relative to current local time
    past := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
    tk := "[\n  {\n    \"kid\": \"expired-key\",\n    \"public_key\": \"" + pub + "\",\n    \"status\": \"active\",\n    \"not_after\": \"" + past + "\"\n  }\n]"
    tkPath := writeTrustedKeysFile(t, dir, tk)

    os.Setenv("TRUSTED_KEYS_FILE", tkPath)
    config.ResetTrustedKeysCache()

    cfg := &config.Config{ReplayProtectionEnabled: true, TimestampMaxSkew: 5 * time.Minute, TimestampMaxAge: 10 * time.Minute, TrustEnforce: true}
    server, _, cleanup := setupTestServer(t, cfg)
    defer cleanup()

    js := &models.JobSpec{
        ID:      "trust-expired",
        Version: "1.0",
        Benchmark: models.BenchmarkSpec{
            Name: "Trust Test", Container: models.ContainerSpec{Image: "test/image:latest", Resources: models.ResourceSpec{CPU: "100m", Memory: "128Mi"}},
            Input: models.InputSpec{Type: "text", Data: map[string]interface{}{"prompt": "hi"}, Hash: "h"},
            Scoring: models.ScoringSpec{Method: "none", Parameters: map[string]interface{}{}},
        },
        Constraints: models.ExecutionConstraints{Regions: []string{"US"}},
        Metadata: map[string]interface{}{"timestamp": time.Now().UTC().Format(time.RFC3339), "nonce": "nonce-expired"},
    }
    require.NoError(t, js.Sign(kp.PrivateKey))

    payload, _ := json.Marshal(js)
    resp, err := http.Post(server.URL+"/api/v1/jobs", "application/json", bytes.NewBuffer(payload))
    require.NoError(t, err)
    defer resp.Body.Close()

    assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
    var body map[string]interface{}
    require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
    assert.Equal(t, "trust_violation:expired", body["error_code"])
}

func TestE2E_TrustedKeys_HotReload_RevocationTakesEffect(t *testing.T) {
    // Generate keypair and initial trusted keys including our key as active
    kp, err := crypto.GenerateKeyPair()
    require.NoError(t, err)
    pub := crypto.PublicKeyToBase64(kp.PublicKey)

    dir := t.TempDir()
    active := "[\n  {\n    \"kid\": \"hotreload\",\n    \"public_key\": \"" + pub + "\",\n    \"status\": \"active\"\n  }\n]"
    tkPath := writeTrustedKeysFile(t, dir, active)
    os.Setenv("TRUSTED_KEYS_FILE", tkPath)
    config.ResetTrustedKeysCache()

    cfg := &config.Config{ReplayProtectionEnabled: true, TimestampMaxSkew: 5 * time.Minute, TimestampMaxAge: 10 * time.Minute, TrustEnforce: true}
    server, mock, cleanup := setupTestServer(t, cfg)
    defer cleanup()

    // Build a valid spec and sign
    js := &models.JobSpec{
        ID:      "hotreload-accept",
        Version: "1.0",
        Benchmark: models.BenchmarkSpec{
            Name: "HotReload Test", Container: models.ContainerSpec{Image: "test/image:latest", Resources: models.ResourceSpec{CPU: "100m", Memory: "128Mi"}},
            Input: models.InputSpec{Type: "text", Data: map[string]interface{}{"prompt": "hi"}, Hash: "h"},
            Scoring: models.ScoringSpec{Method: "none", Parameters: map[string]interface{}{}},
        },
        Constraints: models.ExecutionConstraints{Regions: []string{"US"}},
        Metadata: map[string]interface{}{"timestamp": time.Now().UTC().Format(time.RFC3339), "nonce": "nonce-hr-1"},
    }
    require.NoError(t, js.Sign(kp.PrivateKey))
    payload, err := json.Marshal(js)
    require.NoError(t, err)

    // First submit should be accepted
    mock.ExpectBegin()
    mock.ExpectExec("INSERT INTO jobs").WithArgs(js.ID, sqlmock.AnyArg(), "created").WillReturnResult(sqlmock.NewResult(0, 1))
    mock.ExpectExec("INSERT INTO outbox").WithArgs("jobs", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
    mock.ExpectCommit()
    resp1, err := http.Post(server.URL+"/api/v1/jobs", "application/json", bytes.NewBuffer(payload))
    require.NoError(t, err)
    defer resp1.Body.Close()
    assert.Equal(t, http.StatusAccepted, resp1.StatusCode)

    // Now revoke the key in the file and reset cache
    revoked := "[\n  {\n    \"kid\": \"hotreload\",\n    \"public_key\": \"" + pub + "\",\n    \"status\": \"revoked\"\n  }\n]"
    require.NoError(t, os.WriteFile(tkPath, []byte(revoked), 0o644))
    config.ResetTrustedKeysCache()

    // New request with new nonce should be rejected
    js.Metadata["nonce"] = "nonce-hr-2"
    payload2, err := json.Marshal(js)
    require.NoError(t, err)
    resp2, err := http.Post(server.URL+"/api/v1/jobs", "application/json", bytes.NewBuffer(payload2))
    require.NoError(t, err)
    defer resp2.Body.Close()
    assert.Equal(t, http.StatusBadRequest, resp2.StatusCode)
    var body map[string]interface{}
    require.NoError(t, json.NewDecoder(resp2.Body).Decode(&body))
    assert.Equal(t, "trust_violation:revoked", body["error_code"])
}

func TestE2E_ReplayProtection_Disabled_AllowsDuplicateNonce(t *testing.T) {
    // Generate keypair and trusted keys entry
    kp, err := crypto.GenerateKeyPair()
    require.NoError(t, err)
    pub := crypto.PublicKeyToBase64(kp.PublicKey)

    dir := t.TempDir()
    tk := "[\n  {\n    \"kid\": \"dev-replay-off\",\n    \"public_key\": \"" + pub + "\",\n    \"status\": \"active\",\n    \"not_before\": \"2025-01-01T00:00:00Z\",\n    \"not_after\": \"2026-12-31T23:59:59Z\"\n  }\n]"
    tkPath := writeTrustedKeysFile(t, dir, tk)

    os.Setenv("TRUSTED_KEYS_FILE", tkPath)
    config.ResetTrustedKeysCache()

    cfg := &config.Config{
        ReplayProtectionEnabled: false, // disabled
        TimestampMaxSkew:        5 * time.Minute,
        TimestampMaxAge:         10 * time.Minute,
        TrustEnforce:            true,
    }

    server, mock, cleanup := setupTestServer(t, cfg)
    defer cleanup()

    js := &models.JobSpec{
        ID:      "replay-off",
        Version: "1.0",
        Benchmark: models.BenchmarkSpec{
            Name: "Replay Off",
            Container: models.ContainerSpec{Image: "test/image:latest", Resources: models.ResourceSpec{CPU: "100m", Memory: "128Mi"}},
            Input:  models.InputSpec{Type: "text", Data: map[string]interface{}{"prompt": "hi"}, Hash: "h"},
            Scoring: models.ScoringSpec{Method: "none", Parameters: map[string]interface{}{}},
        },
        Constraints: models.ExecutionConstraints{Regions: []string{"US"}},
        Metadata: map[string]interface{}{
            "timestamp": time.Now().UTC().Format(time.RFC3339),
            "nonce":     "dup-nonce-xyz",
        },
    }
    require.NoError(t, js.Sign(kp.PrivateKey))
    payload, err := json.Marshal(js)
    require.NoError(t, err)

    // Both submits should be accepted and hit DB twice (expected once disable flag is honored)
    mock.ExpectBegin()
    mock.ExpectExec("INSERT INTO jobs").WithArgs(js.ID, sqlmock.AnyArg(), "created").WillReturnResult(sqlmock.NewResult(0, 1))
    mock.ExpectExec("INSERT INTO outbox").WithArgs("jobs", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
    mock.ExpectCommit()
    resp1, err := http.Post(server.URL+"/api/v1/jobs", "application/json", bytes.NewBuffer(payload))
    require.NoError(t, err)
    defer resp1.Body.Close()
    assert.Equal(t, http.StatusAccepted, resp1.StatusCode)

    mock.ExpectBegin()
    mock.ExpectExec("INSERT INTO jobs").WithArgs(js.ID, sqlmock.AnyArg(), "created").WillReturnResult(sqlmock.NewResult(0, 1))
    mock.ExpectExec("INSERT INTO outbox").WithArgs("jobs", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
    mock.ExpectCommit()
    resp2, err := http.Post(server.URL+"/api/v1/jobs", "application/json", bytes.NewBuffer(payload))
    require.NoError(t, err)
    defer resp2.Body.Close()
    assert.Equal(t, http.StatusAccepted, resp2.StatusCode)
}

func writeTrustedKeysFile(t *testing.T, dir string, content string) string {
    t.Helper()
    path := filepath.Join(dir, "trusted-keys.test.json")
    require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
    return path
}

func TestE2E_TrustEnforcement_UnknownRejected(t *testing.T) {
    // Generate a keypair used to sign
    kp, err := crypto.GenerateKeyPair()
    require.NoError(t, err)

    // Trusted file that does NOT include our key
    dir := t.TempDir()
    tk := `[
      {"kid":"someone-else","public_key":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=","status":"active","not_before":"2025-01-01T00:00:00Z","not_after":"2026-12-31T23:59:59Z"}
    ]`
    tkPath := writeTrustedKeysFile(t, dir, tk)

    // Ensure trust cache reloads from our file
    os.Setenv("TRUSTED_KEYS_FILE", tkPath)
    config.ResetTrustedKeysCache()

    cfg := &config.Config{
        ReplayProtectionEnabled: true,
        TimestampMaxSkew:        5 * time.Minute,
        TimestampMaxAge:         10 * time.Minute,
        TrustEnforce:            true,
    }

    server, _, cleanup := setupTestServer(t, cfg)
    defer cleanup()

    // Build a valid JobSpec and sign with our key
    js := &models.JobSpec{
        ID:      "trust-unknown",
        Version: "1.0",
        Benchmark: models.BenchmarkSpec{
            Name:        "Trust Test",
            Description: "unknown key should be rejected",
            Container: models.ContainerSpec{
                Image: "test/image:latest",
                Resources: models.ResourceSpec{CPU: "100m", Memory: "128Mi"},
            },
            Input: models.InputSpec{Type: "text", Data: map[string]interface{}{"prompt": "hi"}, Hash: "h"},
            Scoring: models.ScoringSpec{Method: "none", Parameters: map[string]interface{}{}},
        },
        Constraints: models.ExecutionConstraints{Regions: []string{"US"}},
        Metadata: map[string]interface{}{
            "timestamp": time.Now().UTC().Format(time.RFC3339),
            "nonce":     "nonce-unknown",
        },
    }
    require.NoError(t, js.Sign(kp.PrivateKey))
    payload, err := json.Marshal(js)
    require.NoError(t, err)

    resp, err := http.Post(server.URL+"/api/v1/jobs", "application/json", bytes.NewBuffer(payload))
    require.NoError(t, err)
    defer resp.Body.Close()

    assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
    var body map[string]interface{}
    require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
    assert.Equal(t, "trust_violation:unknown", body["error_code"])
}

func TestE2E_TrustEnforcement_AcceptWhenTrusted(t *testing.T) {
    // Generate keypair
    kp, err := crypto.GenerateKeyPair()
    require.NoError(t, err)
    pub := crypto.PublicKeyToBase64(kp.PublicKey)

    // Trusted keys file that includes our key as active and valid
    dir := t.TempDir()
    tk := "[\n  {\n    \"kid\": \"dev-2025-q3\",\n    \"public_key\": \"" + pub + "\",\n    \"status\": \"active\",\n    \"not_before\": \"2025-01-01T00:00:00Z\",\n    \"not_after\": \"2026-12-31T23:59:59Z\"\n  }\n]"
    tkPath := writeTrustedKeysFile(t, dir, tk)

    os.Setenv("TRUSTED_KEYS_FILE", tkPath)
    config.ResetTrustedKeysCache()

    cfg := &config.Config{
        ReplayProtectionEnabled: true,
        TimestampMaxSkew:        5 * time.Minute,
        TimestampMaxAge:         10 * time.Minute,
        TrustEnforce:            true,
    }

    server, mock, cleanup := setupTestServer(t, cfg)
    defer cleanup()

    // Prepare signed spec
    js := &models.JobSpec{
        ID:      "trust-accept",
        Version: "1.0",
        Benchmark: models.BenchmarkSpec{
            Name:        "Trust Test",
            Description: "trusted key should be accepted",
            Container: models.ContainerSpec{
                Image: "test/image:latest",
                Resources: models.ResourceSpec{CPU: "100m", Memory: "128Mi"},
            },
            Input: models.InputSpec{Type: "text", Data: map[string]interface{}{"prompt": "hi"}, Hash: "h"},
            Scoring: models.ScoringSpec{Method: "none", Parameters: map[string]interface{}{}},
        },
        Constraints: models.ExecutionConstraints{Regions: []string{"US"}},
        Metadata: map[string]interface{}{
            "timestamp": time.Now().UTC().Format(time.RFC3339),
            "nonce":     "nonce-accept",
        },
    }
    require.NoError(t, js.Sign(kp.PrivateKey))
    payload, err := json.Marshal(js)
    require.NoError(t, err)

    // Expect DB interactions when accepted
    mock.ExpectBegin()
    mock.ExpectExec("INSERT INTO jobs").WithArgs(js.ID, sqlmock.AnyArg(), "created").WillReturnResult(sqlmock.NewResult(0, 1))
    mock.ExpectExec("INSERT INTO outbox").WithArgs("jobs", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
    mock.ExpectCommit()

    resp, err := http.Post(server.URL+"/api/v1/jobs", "application/json", bytes.NewBuffer(payload))
    require.NoError(t, err)
    defer resp.Body.Close()

    assert.Equal(t, http.StatusAccepted, resp.StatusCode)
}

func TestE2E_TrustEnforcement_RevokedRejected(t *testing.T) {
    kp, err := crypto.GenerateKeyPair()
    require.NoError(t, err)
    pub := crypto.PublicKeyToBase64(kp.PublicKey)

    dir := t.TempDir()
    tk := "[\n  {\n    \"kid\": \"revoked-key\",\n    \"public_key\": \"" + pub + "\",\n    \"status\": \"revoked\"\n  }\n]"
    tkPath := writeTrustedKeysFile(t, dir, tk)

    os.Setenv("TRUSTED_KEYS_FILE", tkPath)
    config.ResetTrustedKeysCache()

    cfg := &config.Config{ReplayProtectionEnabled: true, TimestampMaxSkew: 5 * time.Minute, TimestampMaxAge: 10 * time.Minute, TrustEnforce: true}
    server, _, cleanup := setupTestServer(t, cfg)
    defer cleanup()

    js := &models.JobSpec{
        ID:      "trust-revoked",
        Version: "1.0",
        Benchmark: models.BenchmarkSpec{
            Name: "Trust Test", Container: models.ContainerSpec{Image: "test/image:latest", Resources: models.ResourceSpec{CPU: "100m", Memory: "128Mi"}},
            Input: models.InputSpec{Type: "text", Data: map[string]interface{}{"prompt": "hi"}, Hash: "h"},
            Scoring: models.ScoringSpec{Method: "none", Parameters: map[string]interface{}{}},
        },
        Constraints: models.ExecutionConstraints{Regions: []string{"US"}},
        Metadata: map[string]interface{}{"timestamp": time.Now().UTC().Format(time.RFC3339), "nonce": "nonce-revoked"},
    }
    require.NoError(t, js.Sign(kp.PrivateKey))

    payload, _ := json.Marshal(js)
    resp, err := http.Post(server.URL+"/api/v1/jobs", "application/json", bytes.NewBuffer(payload))
    require.NoError(t, err)
    defer resp.Body.Close()

    assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
    var body map[string]interface{}
    require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
    assert.Equal(t, "trust_violation:revoked", body["error_code"])
}

func TestE2E_TrustEnforcement_NotYetValid_Expired(t *testing.T) {
    kp, err := crypto.GenerateKeyPair()
    require.NoError(t, err)
    pub := crypto.PublicKeyToBase64(kp.PublicKey)

    dir := t.TempDir()
    // Two entries: one not yet valid, one expired
    tk := "[\n  {\n    \"kid\": \"nyv\",\n    \"public_key\": \"" + pub + "\",\n    \"status\": \"active\",\n    \"not_before\": \"2999-01-01T00:00:00Z\"\n  }\n]"
    tkPath := writeTrustedKeysFile(t, dir, tk)

    os.Setenv("TRUSTED_KEYS_FILE", tkPath)
    config.ResetTrustedKeysCache()

    cfg := &config.Config{ReplayProtectionEnabled: true, TimestampMaxSkew: 5 * time.Minute, TimestampMaxAge: 10 * time.Minute, TrustEnforce: true}
    server, _, cleanup := setupTestServer(t, cfg)
    defer cleanup()

    js := &models.JobSpec{
        ID:      "trust-nyv",
        Version: "1.0",
        Benchmark: models.BenchmarkSpec{
            Name: "Trust Test", Container: models.ContainerSpec{Image: "test/image:latest", Resources: models.ResourceSpec{CPU: "100m", Memory: "128Mi"}},
            Input: models.InputSpec{Type: "text", Data: map[string]interface{}{"prompt": "hi"}, Hash: "h"},
            Scoring: models.ScoringSpec{Method: "none", Parameters: map[string]interface{}{}},
        },
        Constraints: models.ExecutionConstraints{Regions: []string{"US"}},
        Metadata: map[string]interface{}{"timestamp": time.Now().UTC().Format(time.RFC3339), "nonce": "nonce-nyv"},
    }
    require.NoError(t, js.Sign(kp.PrivateKey))

    payload, _ := json.Marshal(js)
    resp, err := http.Post(server.URL+"/api/v1/jobs", "application/json", bytes.NewBuffer(payload))
    require.NoError(t, err)
    defer resp.Body.Close()

    assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
    var body map[string]interface{}
    require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
    assert.Equal(t, "trust_violation:not_yet_valid", body["error_code"])
}
