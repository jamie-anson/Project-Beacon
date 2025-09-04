package api

import (
    "encoding/json"
    "regexp"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/DATA-DOG/go-sqlmock"
    "github.com/jamie-anson/project-beacon-runner/internal/config"
    "github.com/jamie-anson/project-beacon-runner/internal/service"
)

func TestGetJob_NotFound_Returns404(t *testing.T) {
    // sqlmock DB
    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("sqlmock.New: %v", err)
    }
    defer db.Close()

    // Expect SELECT ... WHERE jobspec_id = $1 -> no rows
    mock.ExpectQuery(regexp.QuoteMeta(`SELECT jobspec_data, status, created_at, updated_at 
        FROM jobs 
        WHERE jobspec_id = $1`)).
        WithArgs("job-missing").
        WillReturnRows(sqlmock.NewRows([]string{"jobspec_data", "status", "created_at", "updated_at"}))

    r := SetupRoutes(service.NewJobsService(db), &config.Config{HTTPPort: "8090"}, nil)
    req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/job-missing", nil)
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)

    if w.Code != http.StatusNotFound {
        t.Fatalf("expected 404, got %d; body=%s", w.Code, w.Body.String())
    }
    // Optional: check error message shape
    var resp map[string]any
    _ = json.Unmarshal(w.Body.Bytes(), &resp)

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

func TestGetJob_Latest_NoReceiptYet_ReturnsEmptyExecutions(t *testing.T) {
    // sqlmock DB
    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("sqlmock.New: %v", err)
    }
    defer db.Close()

    // Stored job exists
    storedJSON := []byte(`{"id":"job-latest-empty","version":"1.0","benchmark":{"name":"Test","container":{"image":"alpine:latest"},"input":{"hash":"abc"}},"constraints":{"regions":["US"],"min_regions":1}}`)
    now := time.Now()
    mock.ExpectQuery(regexp.QuoteMeta(`SELECT jobspec_data, status, created_at, updated_at 
        FROM jobs 
        WHERE jobspec_id = $1`)).
        WithArgs("job-latest-empty").
        WillReturnRows(sqlmock.NewRows([]string{"jobspec_data", "status", "created_at", "updated_at"}).
            AddRow(storedJSON, "created", now, now))

    // Latest receipt query returns no rows
    mock.ExpectQuery(regexp.QuoteMeta(`SELECT e.receipt_data
        FROM executions e
        JOIN jobs j ON e.job_id = j.id
        WHERE j.jobspec_id = $1 AND e.receipt_data IS NOT NULL
        ORDER BY e.created_at DESC
        LIMIT 1`)).
        WithArgs("job-latest-empty").
        WillReturnRows(sqlmock.NewRows([]string{"receipt_data"}))

    r := SetupRoutes(service.NewJobsService(db), &config.Config{HTTPPort: "8090"}, nil)
    req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/job-latest-empty?include=latest", nil)
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d; body=%s", w.Code, w.Body.String())
    }
}
