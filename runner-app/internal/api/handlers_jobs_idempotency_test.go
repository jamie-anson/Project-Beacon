package api

import (
    "bytes"
    "database/sql"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "regexp"
    "testing"

    "github.com/DATA-DOG/go-sqlmock"
    "github.com/gin-gonic/gin"
    "github.com/jamie-anson/project-beacon-runner/internal/config"
    "github.com/jamie-anson/project-beacon-runner/internal/service"
    pkcrypto "github.com/jamie-anson/project-beacon-runner/pkg/crypto"
    "github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// helper to build a signed jobspec
func buildSignedSpec(t *testing.T, id string) models.JobSpec {
    t.Helper()
    kp, err := pkcrypto.GenerateKeyPair()
    if err != nil { t.Fatalf("GenerateKeyPair: %v", err) }
    js := models.JobSpec{
        // Don't set ID before signing - let it be set after
        Version: "1.0",
        Benchmark: models.BenchmarkSpec{
            Name: "Test",
            Container: models.ContainerSpec{Image: "alpine:latest"},
            Input: models.InputSpec{Hash: "abc123"},
        },
        Constraints: models.ExecutionConstraints{Regions: []string{"US"}},
    }
    // Sign first without ID
    if err := js.Sign(kp.PrivateKey); err != nil { t.Fatalf("jobspec.Sign: %v", err) }
    // Then set ID (simulating server-side ID assignment)
    js.ID = id
    return js
}

func newRouterWithDB(db *sql.DB) *gin.Engine {
    cfg := &config.Config{
        HTTPPort: "8090",
        SigBypass: true, // Enable signature bypass for tests
    }
    return SetupRoutes(service.NewJobsService(db), cfg, nil)
}

func TestCreateJob_Idempotent_NewKey_202(t *testing.T) {
    t.Parallel()
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock.New: %v", err) }
    defer db.Close()

    // Expectations: Begin -> Upsert jobs -> Insert idempotency -> Insert outbox -> Commit
    mock.ExpectBegin()
    mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO jobs (jobspec_id, jobspec_data, status)
        VALUES ($1, $2, $3)
        ON CONFLICT (jobspec_id)
        DO UPDATE SET jobspec_data = EXCLUDED.jobspec_data, status = EXCLUDED.status, updated_at = NOW()`)).
        WithArgs("job-idem-new", sqlmock.AnyArg(), "created").
        WillReturnResult(sqlmock.NewResult(0, 1))

    mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO idempotency_keys (idem_key, jobspec_id)
        VALUES ($1, $2)
        ON CONFLICT (idem_key) DO NOTHING`)).
        WithArgs("k-123", "job-idem-new").
        WillReturnResult(sqlmock.NewResult(0, 1))

    mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO outbox (topic, payload) VALUES ($1, $2)`)).
        WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
        WillReturnResult(sqlmock.NewResult(1, 1))

    mock.ExpectCommit()

    r := newRouterWithDB(db)
    spec := buildSignedSpec(t, "job-idem-new")
    body, _ := json.Marshal(spec)
    req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Idempotency-Key", "k-123")

    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusAccepted {
        t.Fatalf("expected 202, got %d; body=%s", w.Code, w.Body.String())
    }
    var resp struct{ ID string `json:"id"`; Status string `json:"status"` }
    if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
        t.Fatalf("unmarshal: %v; body=%s", err, w.Body.String())
    }
    if resp.ID != "job-idem-new" || resp.Status != "enqueued" {
        t.Fatalf("unexpected response: %+v", resp)
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

func TestCreateJob_Idempotent_DuplicateKey_200(t *testing.T) {
    t.Parallel()
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock.New: %v", err) }
    defer db.Close()

    // Pre-check: IdempotencyRepo.GetByKey reads directly from DB
    mock.ExpectQuery(regexp.QuoteMeta(`SELECT jobspec_id FROM idempotency_keys WHERE idem_key = $1`)).
        WithArgs("k-dup").
        WillReturnRows(sqlmock.NewRows([]string{"jobspec_id"}).AddRow("job-abc"))

    r := newRouterWithDB(db)
    spec := buildSignedSpec(t, "job-ignored") // should be ignored due to duplicate key
    body, _ := json.Marshal(spec)
    req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Idempotency-Key", "k-dup")

    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d; body=%s", w.Code, w.Body.String())
    }
    var resp struct{ ID string `json:"id"`; Idempotent bool `json:"idempotent"` }
    if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
        t.Fatalf("unmarshal: %v; body=%s", err, w.Body.String())
    }
    if resp.ID != "job-abc" || !resp.Idempotent {
        t.Fatalf("unexpected response: %+v", resp)
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}
