package integration

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "os"
    "path/filepath"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/require"

    "github.com/jamie-anson/project-beacon-runner/internal/api"
    appdb "github.com/jamie-anson/project-beacon-runner/internal/db"
    "github.com/jamie-anson/project-beacon-runner/internal/config"
    "github.com/jamie-anson/project-beacon-runner/internal/service"
    "github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// buildUnsignedSpec creates a minimal valid spec; signature checks are bypassed via cfg.SigBypass.
func buildUnsignedSpec(id string) *models.JobSpec {
    return &models.JobSpec{
        ID:      id,
        Version: "1.0",
        Benchmark: models.BenchmarkSpec{
            Name:     "IdemTest",
            Container: models.ContainerSpec{Image: "alpine:latest"},
            Input:    models.InputSpec{Hash: "h"},
        },
        Constraints: models.ExecutionConstraints{Regions: []string{"US"}},
    }
}

func setupRouterWithDB(t *testing.T) (*gin.Engine, *appdb.DB) {
    t.Helper()

    // Ensure migrations run (for idempotency_keys table)
    _ = os.Setenv("USE_MIGRATIONS", "true")
    // Point to migrations dir relative to repo root (runner-app/migrations)
    cwd, _ := os.Getwd()
    mig := filepath.Join(cwd, "..", "..", "migrations")
    _ = os.Setenv("MIGRATIONS_PATH", mig)

    // Require DATABASE_URL or skip
    if os.Getenv("DATABASE_URL") == "" {
        t.Skip("Skipping idempotency integration: DATABASE_URL not set")
    }

    db, err := appdb.Initialize("")
    require.NoError(t, err)
    require.NotNil(t, db)
    require.NotNil(t, db.DB)

    // Clean tables to avoid cross-test flake
    _, _ = db.Exec(`DELETE FROM outbox`)
    _, _ = db.Exec(`DELETE FROM idempotency_keys`)
    _, _ = db.Exec(`DELETE FROM jobs`)

    // Build routes with SigBypass enabled to avoid signing overhead
    cfg := config.Load()
    cfg.SigBypass = true
    cfg.TrustEnforce = false
    cfg.HTTPPort = ":8090"

    r := api.SetupRoutes(service.NewJobsService(db.DB), cfg, nil)
    return r, db
}

func TestIntegration_Idempotency_DuplicateKey(t *testing.T) {
    r, db := setupRouterWithDB(t)
    defer func() { if db != nil && db.DB != nil { db.Close() } }()

    // First request with key => 202 Accepted
    spec1 := buildUnsignedSpec("job-int-1")
    b1, _ := json.Marshal(spec1)
    req1 := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(b1))
    req1.Header.Set("Content-Type", "application/json")
    req1.Header.Set("Idempotency-Key", "K1")
    w1 := httptest.NewRecorder()
    r.ServeHTTP(w1, req1)
    if w1.Code != http.StatusAccepted {
        t.Fatalf("expected 202 first, got %d body=%s", w1.Code, w1.Body.String())
    }
    var resp1 struct{ ID string `json:"id"` }
    _ = json.Unmarshal(w1.Body.Bytes(), &resp1)

    // Second request with same key, different job id in body => 200 OK, returns existing id
    spec2 := buildUnsignedSpec("job-int-ignored")
    b2, _ := json.Marshal(spec2)
    req2 := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(b2))
    req2.Header.Set("Content-Type", "application/json")
    req2.Header.Set("Idempotency-Key", "K1")
    w2 := httptest.NewRecorder()
    r.ServeHTTP(w2, req2)
    if w2.Code != http.StatusOK {
        t.Fatalf("expected 200 duplicate, got %d body=%s", w2.Code, w2.Body.String())
    }
    var resp2 struct{ ID string `json:"id"`; Idempotent bool `json:"idempotent"` }
    _ = json.Unmarshal(w2.Body.Bytes(), &resp2)
    if !resp2.Idempotent || resp2.ID != resp1.ID {
        t.Fatalf("expected idempotent true and same id; got resp2=%+v resp1=%+v", resp2, resp1)
    }
}

func TestIntegration_Idempotency_DifferentKeys_NewJobs(t *testing.T) {
    r, db := setupRouterWithDB(t)
    defer func() { if db != nil && db.DB != nil { db.Close() } }()

    // Request A with K2
    specA := buildUnsignedSpec("job-int-A")
    bA, _ := json.Marshal(specA)
    reqA := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(bA))
    reqA.Header.Set("Content-Type", "application/json")
    reqA.Header.Set("Idempotency-Key", "K2")
    wA := httptest.NewRecorder()
    r.ServeHTTP(wA, reqA)
    if wA.Code != http.StatusAccepted { t.Fatalf("expected 202, got %d", wA.Code) }
    var rA struct{ ID string `json:"id"` }
    _ = json.Unmarshal(wA.Body.Bytes(), &rA)

    // Request B with K3
    specB := buildUnsignedSpec("job-int-B")
    bB, _ := json.Marshal(specB)
    reqB := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(bB))
    reqB.Header.Set("Content-Type", "application/json")
    reqB.Header.Set("Idempotency-Key", "K3")
    wB := httptest.NewRecorder()
    r.ServeHTTP(wB, reqB)
    if wB.Code != http.StatusAccepted { t.Fatalf("expected 202, got %d", wB.Code) }
    var rB struct{ ID string `json:"id"` }
    _ = json.Unmarshal(wB.Body.Bytes(), &rB)

    if rA.ID == rB.ID {
        t.Fatalf("expected different job ids for different keys; got %s == %s", rA.ID, rB.ID)
    }
}

func TestIntegration_NoKey_CreatesNewJob(t *testing.T) {
    r, db := setupRouterWithDB(t)
    defer func() { if db != nil && db.DB != nil { db.Close() } }()

    spec := buildUnsignedSpec("job-int-nokey")
    b, _ := json.Marshal(spec)
    req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(b))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    if w.Code != http.StatusAccepted {
        t.Fatalf("expected 202 without idempotency header, got %d body=%s", w.Code, w.Body.String())
    }
}
