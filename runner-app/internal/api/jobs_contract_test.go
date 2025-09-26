package api

import (
    "bytes"
    "database/sql"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "regexp"
    "testing"
    "time"

    "github.com/DATA-DOG/go-sqlmock"
    "github.com/jamie-anson/project-beacon-runner/internal/config"
    "github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// Trust enforcement: when enabled and signing key is not in allowlist, CreateJob returns 400 with error_code
func TestContract_CreateJob_TrustEnforce_Untrusted_400(t *testing.T) {
    t.Parallel()
    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("sqlmock.New: %v", err)
    }
    defer db.Close()

    cfg := &config.Config{
        HTTPPort:                "8090",
        TrustEnforce:            true,
        SigBypass:               false,
        ReplayProtectionEnabled: false,
        TimestampMaxSkew:        5 * time.Minute,
        TimestampMaxAge:         10 * time.Minute,
    }
    r := newTestRouterWithDBConfig(db, cfg)

    // Build a valid signed spec with a random key (not in allowlist)
    js := buildSignedJobSpec(t, "job-trust-bad")
    b, _ := json.Marshal(js)
    req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(b))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusBadRequest { t.Fatalf("want 400, got %d; body=%s", w.Code, w.Body.String()) }
    if w.Header().Get("X-Request-ID") == "" { t.Fatalf("missing X-Request-ID header") }

    jsonEqual(t, w.Body.Bytes(), mustReadGolden(t, "create_job_trust_violation_400.json"))
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unexpected DB usage: %v", err)
    }
}

// Rate limiting: hammer an endpoint until 429, assert headers and body via golden
func TestContract_RateLimit_429(t *testing.T) {
    t.Parallel()
    r := newTestRouter()

    // Use a constant RemoteAddr to ensure same bucket
    var got429 *httptest.ResponseRecorder
    for i := 0; i < 80; i++ { // exceed 60 tokens
        req := httptest.NewRequest(http.MethodGet, "/health", nil)
        req.RemoteAddr = "192.0.2.55:12345"
        w := httptest.NewRecorder()
        r.ServeHTTP(w, req)
        if w.Code == http.StatusTooManyRequests {
            got429 = w
            break
        }
    }
    if got429 == nil {
        t.Fatalf("did not reach 429 after many requests; rate limiting not enforced")
    }
    // Headers (stable values only)
    if got429.Header().Get("X-Request-ID") == "" { t.Fatalf("missing X-Request-ID header on 429") }
    if got429.Header().Get("X-RateLimit-Limit") != "60" { t.Fatalf("expected X-RateLimit-Limit=60, got %q", got429.Header().Get("X-RateLimit-Limit")) }
    if got429.Header().Get("X-RateLimit-Remaining") != "0" { t.Fatalf("expected X-RateLimit-Remaining=0, got %q", got429.Header().Get("X-RateLimit-Remaining")) }
    if got429.Header().Get("X-RateLimit-Reset") == "" { t.Fatalf("missing X-RateLimit-Reset header") }

    // Normalize volatile fields and compare to golden
    var body map[string]any
    if err := json.Unmarshal(got429.Body.Bytes(), &body); err != nil {
        t.Fatalf("unmarshal 429 body: %v; body=%s", err, got429.Body.String())
    }
    // Normalize request_id to deterministic placeholder
    body["request_id"] = "RID"
    norm, err := json.Marshal(body)
    if err != nil { t.Fatalf("marshal normalized body: %v", err) }
    jsonEqual(t, norm, mustReadGolden(t, "rate_limit_429.json"))
}

func TestContract_CreateJob_202(t *testing.T) {
    t.Parallel()
    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("sqlmock.New: %v", err)
    }
    defer db.Close()

    mock.ExpectBegin()
    mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO jobs (jobspec_id, jobspec_data, status)
        VALUES ($1, $2, $3)
        ON CONFLICT (jobspec_id)
        DO UPDATE SET jobspec_data = EXCLUDED.jobspec_data, status = EXCLUDED.status, updated_at = NOW()`)).
        WithArgs("job-golden-202", sqlmock.AnyArg(), "created").
        WillReturnResult(sqlmock.NewResult(0, 1))
    mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO outbox (topic, payload) VALUES ($1, $2)`)).
        WithArgs("jobs", sqlmock.AnyArg()).
        WillReturnResult(sqlmock.NewResult(1, 1))
    mock.ExpectCommit()

    r := newTestRouterWithDB(db)
    // Build a valid signed spec
    js := buildSignedJobSpec(t, "job-golden-202")
    b, _ := json.Marshal(js)
    req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(b))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusAccepted { t.Fatalf("want 202, got %d; body=%s", w.Code, w.Body.String()) }
    if w.Header().Get("X-Request-ID") == "" { t.Fatalf("missing X-Request-ID header") }
    jsonEqual(t, w.Body.Bytes(), mustReadGolden(t, "create_job_202.json"))
    if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet expectations: %v", err) }
}

func TestContract_GetJob_404(t *testing.T) {
    t.Parallel()
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock.New: %v", err) }
    defer db.Close()

    mock.ExpectQuery(regexp.QuoteMeta(`SELECT jobspec_data, status, created_at, updated_at 
        FROM jobs 
        WHERE jobspec_id = $1`)).
        WithArgs("job-missing").
        WillReturnError(sql.ErrNoRows)

    r := newTestRouterWithDB(db)
    req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/job-missing", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusNotFound { t.Fatalf("want 404, got %d; body=%s", w.Code, w.Body.String()) }
    if w.Header().Get("X-Request-ID") == "" { t.Fatalf("missing X-Request-ID header") }
    jsonEqual(t, w.Body.Bytes(), mustReadGolden(t, "get_job_404.json"))
    if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet expectations: %v", err) }
}

func TestContract_GetJob_200(t *testing.T) {
    t.Parallel()
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock.New: %v", err) }
    defer db.Close()

    // Stored JobSpec row
    storedJobSpec := models.JobSpec{
        ID:      "job-g200",
        Version: "1.0",
        Benchmark: models.BenchmarkSpec{
            Name:        "Test",
            Description: "",
            Container: models.ContainerSpec{
                Image: "alpine:latest",
                Resources: models.ResourceSpec{
                    CPU:    "",
                    Memory: "",
                },
            },
            Input: models.InputSpec{
                Type: "",
                Data: nil,
                Hash: "abc123",
            },
            Scoring: models.ScoringSpec{
                Method:     "",
                Parameters: nil,
            },
            Metadata: nil,
        },
        Constraints: models.ExecutionConstraints{
            Regions: []string{"US"},
        },
        Signature: "sig",
        PublicKey: "pk",
    }
    jobspecJSON, _ := json.Marshal(storedJobSpec)
    fixedTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
    jobRows := sqlmock.NewRows([]string{"jobspec_data", "status", "created_at", "updated_at"}).
        AddRow(jobspecJSON, "created", fixedTime, fixedTime)
    mock.ExpectQuery(regexp.QuoteMeta(`SELECT jobspec_data, status, created_at, updated_at 
        FROM jobs 
        WHERE jobspec_id = $1`)).
        WithArgs("job-g200").
        WillReturnRows(jobRows)

    r := newTestRouterWithDB(db)
    req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/job-g200", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusOK { t.Fatalf("want 200, got %d; body=%s", w.Code, w.Body.String()) }
    if w.Header().Get("X-Request-ID") == "" { t.Fatalf("missing X-Request-ID header") }
    jsonEqual(t, w.Body.Bytes(), mustReadGolden(t, "get_job_200.json"))
    if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet expectations: %v", err) }
}

func TestContract_CreateJob_InvalidJSON_400(t *testing.T) {
    t.Parallel()
    r := newTestRouter()

    req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewBufferString("{ not-json }"))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusBadRequest { t.Fatalf("want 400, got %d", w.Code) }
    if w.Header().Get("X-Request-ID") == "" { t.Fatalf("missing X-Request-ID header") }

    jsonEqual(t, w.Body.Bytes(), mustReadGolden(t, "create_job_invalid_json_400.json"))
}

func TestContract_ListJobs_200(t *testing.T) {
    t.Parallel()
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock.New: %v", err) }
    defer db.Close()

    fixed := time.Date(2025, 8, 12, 0, 0, 0, 0, time.UTC)
    rows := sqlmock.NewRows([]string{"jobspec_id", "status", "created_at"}).
        AddRow("job-golden", "queued", fixed)
    mock.ExpectQuery(regexp.QuoteMeta(`SELECT jobspec_id, status, created_at FROM jobs ORDER BY created_at DESC LIMIT $1`)).
        WithArgs(50).
        WillReturnRows(rows)

    r := newTestRouterWithDB(db)
    req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusOK { t.Fatalf("want 200, got %d; body=%s", w.Code, w.Body.String()) }
    if w.Header().Get("X-Request-ID") == "" { t.Fatalf("missing X-Request-ID header") }

    jsonEqual(t, w.Body.Bytes(), mustReadGolden(t, "list_jobs_200.json"))
    if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet expectations: %v", err) }
}

func TestContract_ListJobs_500(t *testing.T) {
    t.Parallel()
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock.New: %v", err) }
    defer db.Close()

    mock.ExpectQuery(regexp.QuoteMeta(`SELECT jobspec_id, status, created_at FROM jobs ORDER BY created_at DESC LIMIT $1`)).
        WithArgs(50).
        WillReturnError(sql.ErrConnDone)

    r := newTestRouterWithDB(db)
    req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusInternalServerError { t.Fatalf("want 500, got %d", w.Code) }
    if w.Header().Get("X-Request-ID") == "" { t.Fatalf("missing X-Request-ID header") }

    jsonEqual(t, w.Body.Bytes(), mustReadGolden(t, "list_jobs_500.json"))
    if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet expectations: %v", err) }
}
