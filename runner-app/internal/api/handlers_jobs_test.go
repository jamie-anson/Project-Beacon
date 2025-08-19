package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"regexp"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jamie-anson/project-beacon-runner/internal/config"
	"github.com/jamie-anson/project-beacon-runner/internal/service"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
	pkcrypto "github.com/jamie-anson/project-beacon-runner/pkg/crypto"
)

func newTestRouter() *gin.Engine {
    cfg := &config.Config{HTTPPort: "8090"}
    return SetupRoutes(service.NewJobsService(nil), cfg)
}

func TestGetJob_IncludeExecutions_NoExecutions(t *testing.T) {
    // sqlmock DB
    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("sqlmock.New: %v", err)
    }
    defer db.Close()

    // Prepare stored JobSpec row
    stored := buildSignedJobSpec(t, "job-empty")
    storedJSON, _ := json.Marshal(stored)
    jobRows := sqlmock.NewRows([]string{"jobspec_data", "status", "created_at", "updated_at"}).
        AddRow(storedJSON, "created", time.Now(), time.Now())
    mock.ExpectQuery(regexp.QuoteMeta(`SELECT jobspec_data, status, created_at, updated_at 
        FROM jobs 
        WHERE jobspec_id = $1`)).
        WithArgs("job-empty").
        WillReturnRows(jobRows)

    // Paginated executions query returns zero rows (no executions yet)
    mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT e.receipt_data
        FROM executions e
        JOIN jobs j ON e.job_id = j.id
        WHERE j.jobspec_id = $1 AND e.receipt_data IS NOT NULL
        ORDER BY e.created_at DESC
        LIMIT $2 OFFSET $3
    `)).
        WithArgs("job-empty", 20, 0).
        WillReturnRows(sqlmock.NewRows([]string{"receipt_data"}))

    r := newTestRouterWithDB(db)
    req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/job-empty?include=executions", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d; body=%s", w.Code, w.Body.String())
    }
    var resp struct {
        Executions []json.RawMessage `json:"executions"`
    }
    if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
        t.Fatalf("unmarshal response: %v; body=%s", err, w.Body.String())
    }
    if len(resp.Executions) != 0 {
        t.Fatalf("expected empty executions, got %d", len(resp.Executions))
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}


func TestListJobs_HappyPathWithLimit(t *testing.T) {
    // sqlmock DB
    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("sqlmock.New: %v", err)
    }
    defer db.Close()

    // Expect query with limit=3 as per JobsRepo.ListRecentJobs
    now := time.Now()
    rows := sqlmock.NewRows([]string{"jobspec_id", "status", "created_at"}).
        AddRow("job-a", "queued", now).
        AddRow("job-b", "created", now.Add(-time.Minute)).
        AddRow("job-c", "completed", now.Add(-2*time.Minute))
    mock.ExpectQuery(regexp.QuoteMeta(`SELECT jobspec_id, status, created_at FROM jobs ORDER BY created_at DESC LIMIT $1`)).
        WithArgs(3).
        WillReturnRows(rows)

    r := newTestRouterWithDB(db)
    req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs?limit=3", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d; body=%s", w.Code, w.Body.String())
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

func TestListJobs_DBError(t *testing.T) {
    // sqlmock DB
    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("sqlmock.New: %v", err)
    }
    defer db.Close()

    // Force query error
    mock.ExpectQuery(regexp.QuoteMeta(`SELECT jobspec_id, status, created_at FROM jobs ORDER BY created_at DESC LIMIT $1`)).
        WithArgs(50). // default limit when not provided
        WillReturnError(sql.ErrConnDone)

    r := newTestRouterWithDB(db)
    req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusInternalServerError {
        t.Fatalf("expected 500, got %d; body=%s", w.Code, w.Body.String())
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

func TestGetJob_IncludeExecutions_Paginated(t *testing.T) {
    // sqlmock DB
    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("sqlmock.New: %v", err)
    }
    defer db.Close()

    // Prepare stored JobSpec row
    stored := buildSignedJobSpec(t, "job-exec-pg")
    storedJSON, _ := json.Marshal(stored)
    jobRows := sqlmock.NewRows([]string{"jobspec_data", "status", "created_at", "updated_at"}).
        AddRow(storedJSON, "created", time.Now(), time.Now())
    mock.ExpectQuery(regexp.QuoteMeta(`SELECT jobspec_data, status, created_at, updated_at 
        FROM jobs 
        WHERE jobspec_id = $1`)).
        WithArgs("job-exec-pg").
        WillReturnRows(jobRows)

    // Prepare paginated executions rows
    rec1 := models.NewReceipt("job-exec-pg", models.ExecutionDetails{ProviderID: "p1", Region: "US", Status: "completed", StartedAt: time.Now(), CompletedAt: time.Now()}, models.ExecutionOutput{Data: map[string]interface{}{"msg": "r1"}, Hash: "h1"}, models.ProvenanceInfo{})
    rec2 := models.NewReceipt("job-exec-pg", models.ExecutionDetails{ProviderID: "p2", Region: "EU", Status: "completed", StartedAt: time.Now(), CompletedAt: time.Now()}, models.ExecutionOutput{Data: map[string]interface{}{"msg": "r2"}, Hash: "h2"}, models.ProvenanceInfo{})
    rec1JSON, _ := json.Marshal(rec1)
    rec2JSON, _ := json.Marshal(rec2)
    mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT e.receipt_data
        FROM executions e
        JOIN jobs j ON e.job_id = j.id
        WHERE j.jobspec_id = $1 AND e.receipt_data IS NOT NULL
        ORDER BY e.created_at DESC
        LIMIT $2 OFFSET $3
    `)).
        WithArgs("job-exec-pg", 2, 0).
        WillReturnRows(sqlmock.NewRows([]string{"receipt_data"}).AddRow(rec1JSON).AddRow(rec2JSON))

    r := newTestRouterWithDB(db)
    req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/job-exec-pg?include=executions&exec_limit=2&exec_offset=0", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d; body=%s", w.Code, w.Body.String())
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

func TestGetJob_IncludeExecutions_FallbackToNonPaginated(t *testing.T) {
    // sqlmock DB
    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("sqlmock.New: %v", err)
    }
    defer db.Close()

    // Prepare stored JobSpec row
    stored := buildSignedJobSpec(t, "job-exec-fb")
    storedJSON, _ := json.Marshal(stored)
    jobRows := sqlmock.NewRows([]string{"jobspec_data", "status", "created_at", "updated_at"}).
        AddRow(storedJSON, "created", time.Now(), time.Now())
    mock.ExpectQuery(regexp.QuoteMeta(`SELECT jobspec_data, status, created_at, updated_at 
        FROM jobs 
        WHERE jobspec_id = $1`)).
        WithArgs("job-exec-fb").
        WillReturnRows(jobRows)

    // Force paginated query to error, triggering fallback
    mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT e.receipt_data
        FROM executions e
        JOIN jobs j ON e.job_id = j.id
        WHERE j.jobspec_id = $1 AND e.receipt_data IS NOT NULL
        ORDER BY e.created_at DESC
        LIMIT $2 OFFSET $3
    `)).
        WithArgs("job-exec-fb", 20, 0).
        WillReturnError(sql.ErrConnDone)

    // Fallback non-paginated list
    rec := models.NewReceipt("job-exec-fb", models.ExecutionDetails{ProviderID: "p3", Region: "APAC", Status: "completed", StartedAt: time.Now(), CompletedAt: time.Now()}, models.ExecutionOutput{Data: map[string]interface{}{"msg": "only"}, Hash: "h3"}, models.ProvenanceInfo{})
    recJSON, _ := json.Marshal(rec)
    mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT e.receipt_data
        FROM executions e
        JOIN jobs j ON e.job_id = j.id
        WHERE j.jobspec_id = $1 AND e.receipt_data IS NOT NULL
        ORDER BY e.created_at DESC
    `)).
        WithArgs("job-exec-fb").
        WillReturnRows(sqlmock.NewRows([]string{"receipt_data"}).AddRow(recJSON))

    r := newTestRouterWithDB(db)
    req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/job-exec-fb?include=executions", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d; body=%s", w.Code, w.Body.String())
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

func newTestRouterWithDB(mockDB *sql.DB) *gin.Engine {
	cfg := &config.Config{HTTPPort: "8090"}
	return SetupRoutes(service.NewJobsService(mockDB), cfg)
}

// buildSignedJobSpec creates a minimally valid, signed JobSpec for tests
func buildSignedJobSpec(t *testing.T, id string) models.JobSpec {
    t.Helper()
    kp, err := pkcrypto.GenerateKeyPair()
    if err != nil {
        t.Fatalf("GenerateKeyPair: %v", err)
    }
    js := models.JobSpec{
        ID:      id,
        Version: "1.0",
        Benchmark: models.BenchmarkSpec{
            Name:      "Test",
            Container: models.ContainerSpec{Image: "alpine:latest"},
            Input:     models.InputSpec{Hash: "abc123"},
        },
        Constraints: models.ExecutionConstraints{Regions: []string{"US"}, MinRegions: 3, MinSuccessRate: 0.67, Timeout: 10 * time.Minute, ProviderTimeout: 2 * time.Minute},
        CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
    }
    if err := js.Sign(kp.PrivateKey); err != nil {
        t.Fatalf("jobspec.Sign: %v", err)
    }
    return js
}

func TestCreateJob_InvalidJSON(t *testing.T) {
	r := newTestRouter()
	body := bytes.NewBufferString("{ not-json }")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d; body=%s", w.Code, w.Body.String())
	}
}

func TestCreateJob_UnsignedSpec_FailsValidation(t *testing.T) {
	r := newTestRouter()
	// Minimal but unsigned jobspec; ValidateAndVerify should reject signature missing
	spec := models.JobSpec{
		ID:      "job-invalid-unsigned",
		Version: "1.0",
		Benchmark: models.BenchmarkSpec{
			Name: "Test",
			Container: models.ContainerSpec{Image: "alpine:latest"},
			Input: models.InputSpec{Hash: "abc123"},
		},
		Constraints: models.ExecutionConstraints{Regions: []string{"US"}},
	}
	b, _ := json.Marshal(spec)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unsigned spec, got %d; body=%s", w.Code, w.Body.String())
	}
}

func TestGetJob_PersistenceUnavailable(t *testing.T) {
	r := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/job-123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when DB is nil, got %d; body=%s", w.Code, w.Body.String())
	}
}

func TestCreateJob_HappyPath202(t *testing.T) {
	// sqlmock DB
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	// Expectations: BEGIN -> UpsertJobTx -> Outbox.InsertTx -> COMMIT
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO jobs (jobspec_id, jobspec_data, status)
        VALUES ($1, $2, $3)
        ON CONFLICT (jobspec_id)
        DO UPDATE SET jobspec_data = EXCLUDED.jobspec_data, status = EXCLUDED.status, updated_at = NOW()`)).
		WithArgs("job-ok", sqlmock.AnyArg(), "created").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO outbox (topic, payload) VALUES ($1, $2)`)).
		WithArgs("jobs", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	r := newTestRouterWithDB(db)
	js := buildSignedJobSpec(t, "job-ok")
	b, _ := json.Marshal(js)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d; body=%s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetJob_IncludeLatest(t *testing.T) {
	// sqlmock DB
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	// Prepare stored JobSpec row
	stored := buildSignedJobSpec(t, "job-1")
	storedJSON, _ := json.Marshal(stored)
	jobRows := sqlmock.NewRows([]string{"jobspec_data", "status", "created_at", "updated_at"}).
		AddRow(storedJSON, "created", time.Now(), time.Now())
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT jobspec_data, status, created_at, updated_at 
		FROM jobs 
		WHERE jobspec_id = $1`)).
		WithArgs("job-1").
		WillReturnRows(jobRows)

	// Prepare latest receipt row
	rec := models.NewReceipt("job-1", models.ExecutionDetails{ProviderID: "prov-1", Region: "US", Status: "completed", StartedAt: time.Now(), CompletedAt: time.Now()}, models.ExecutionOutput{Data: map[string]interface{}{"msg": "ok"}, Hash: "h"}, models.ProvenanceInfo{})
	recJSON, _ := json.Marshal(rec)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT e.receipt_data
		FROM executions e
		JOIN jobs j ON e.job_id = j.id
		WHERE j.jobspec_id = $1 AND e.receipt_data IS NOT NULL
		ORDER BY e.created_at DESC
		LIMIT 1`)).
		WithArgs("job-1").
		WillReturnRows(sqlmock.NewRows([]string{"receipt_data"}).AddRow(recJSON))

	r := newTestRouterWithDB(db)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/job-1?include=latest", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body=%s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
