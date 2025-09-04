//go:build skip

package api

import (
    "bytes"
    "database/sql"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/DATA-DOG/go-sqlmock"
    "github.com/gin-gonic/gin"
    "github.com/jamie-anson/project-beacon-runner/internal/service"
    "github.com/jamie-anson/project-beacon-runner/internal/store"
    "github.com/jamie-anson/project-beacon-runner/pkg/models"
)

func setupRouter(h *JobsHandler) *gin.Engine {
    gin.SetMode(gin.TestMode)
    r := gin.New()
    r.POST("/jobs", h.CreateJob)
    r.GET("/jobs/:id", h.GetJob)
    return r
}

func TestCreateJob_InvalidJSON(t *testing.T) {
    h := &JobsHandler{jobsService: &service.JobsService{}}
    r := setupRouter(h)

    rr := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString("{"))
    r.ServeHTTP(rr, req)

    if rr.Code != http.StatusBadRequest {
        t.Fatalf("status=%d", rr.Code)
    }
}

func TestCreateJob_ValidationError(t *testing.T) {
    h := &JobsHandler{jobsService: &service.JobsService{}}
    r := setupRouter(h)

    // empty JobSpec should fail validator
    body, _ := json.Marshal(map[string]interface{}{})
    rr := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBuffer(body))
    req.Header.Set("Content-Type", "application/json")
    r.ServeHTTP(rr, req)

    if rr.Code != http.StatusBadRequest {
        t.Fatalf("status=%d", rr.Code)
    }
}

func TestGetJob_MissingID(t *testing.T) {
    h := &JobsHandler{jobsService: &service.JobsService{}}
    r := gin.New()
    r.GET("/jobs", h.GetJob) // no :id segment

    rr := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/jobs", nil)
    r.ServeHTTP(rr, req)

    if rr.Code != http.StatusBadRequest {
        t.Fatalf("status=%d", rr.Code)
    }
}

func TestGetJob_PersistenceUnavailable(t *testing.T) {
    // jobsService.DB is nil triggers 503
    h := &JobsHandler{jobsService: &service.JobsService{DB: nil}}
    r := setupRouter(h)

    rr := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/jobs/js-1", nil)
    r.ServeHTTP(rr, req)

    if rr.Code != http.StatusServiceUnavailable {
        t.Fatalf("status=%d", rr.Code)
    }
}

func TestGetJob_NotFoundMapping(t *testing.T) {
    mockDB, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock: %v", err) }
    defer mockDB.Close()

    js := &service.JobsService{DB: mockDB}
    js.JobsRepo = store.NewJobsRepo(mockDB)

    // Expect jobs query returns no rows
    mock.ExpectQuery("SELECT jobspec_data, status, created_at, updated_at ").
        WithArgs("js-404").
        WillReturnError(sql.ErrNoRows)

    h := &JobsHandler{jobsService: js}
    r := setupRouter(h)

    rr := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/jobs/js-404", nil)
    r.ServeHTTP(rr, req)

    if rr.Code != http.StatusNotFound {
        t.Fatalf("status=%d", rr.Code)
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

func mustJobspecJSON(t *testing.T, spec *models.JobSpec) []byte {
    t.Helper()
    b, err := json.Marshal(spec)
    if err != nil { t.Fatalf("marshal jobspec: %v", err) }
    return b
}

func mustReceiptJSON(t *testing.T, r *models.Receipt) []byte {
    t.Helper()
    b, err := json.Marshal(r)
    if err != nil { t.Fatalf("marshal receipt: %v", err) }
    return b
}

func TestGetJob_IncludeLatest(t *testing.T) {
    mockDB, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock: %v", err) }
    defer mockDB.Close()

    js := &service.JobsService{DB: mockDB}
    js.JobsRepo = store.NewJobsRepo(mockDB)
    js.ExecutionsRepo = store.NewExecutionsRepo(mockDB)

    now := time.Now()
    spec := &models.JobSpec{ID: "js-ok", Version: "1.0.0"}
    jobsRows := sqlmock.NewRows([]string{"jobspec_data", "status", "created_at", "updated_at"}).
        AddRow(mustJobspecJSON(t, spec), "queued", now, now)
    mock.ExpectQuery("SELECT jobspec_data, status, created_at, updated_at ").
        WithArgs("js-ok").
        WillReturnRows(jobsRows)

    rec := &models.Receipt{ExecutionDetails: models.ExecutionDetails{ProviderID: "p1", Region: "us", Status: "completed"}}
    mock.ExpectQuery("SELECT e.receipt_data\n\t\tFROM executions e\n\t\tJOIN jobs j ON e.job_id = j.id").
        WithArgs("js-ok").
        WillReturnRows(sqlmock.NewRows([]string{"receipt_data"}).AddRow(mustReceiptJSON(t, rec)))

    h := &JobsHandler{jobsService: js}
    r := setupRouter(h)

    rr := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/jobs/js-ok?include=latest", nil)
    r.ServeHTTP(rr, req)

    if rr.Code != http.StatusOK {
        t.Fatalf("status=%d", rr.Code)
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

func TestGetJob_IncludeAll_Fallback(t *testing.T) {
    mockDB, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock: %v", err) }
    defer mockDB.Close()

    js := &service.JobsService{DB: mockDB}
    js.JobsRepo = store.NewJobsRepo(mockDB)
    js.ExecutionsRepo = store.NewExecutionsRepo(mockDB)

    now := time.Now()
    spec := &models.JobSpec{ID: "js-all", Version: "1.0.0"}
    jobsRows := sqlmock.NewRows([]string{"jobspec_data", "status", "created_at", "updated_at"}).
        AddRow(mustJobspecJSON(t, spec), "running", now, now)
    mock.ExpectQuery("SELECT jobspec_data, status, created_at, updated_at ").
        WithArgs("js-all").
        WillReturnRows(jobsRows)

    // First (paginated) query returns error
    mock.ExpectQuery("SELECT e.receipt_data\n\t\tFROM executions e\n\t\tJOIN jobs j ON e.job_id = j.id\n\t\tWHERE j.jobspec_id = \$1 AND e.receipt_data IS NOT NULL\n\t\tORDER BY e.created_at DESC\n\t\tLIMIT \$2 OFFSET \$3").
        WithArgs("js-all", 20, 0).
        WillReturnError(sql.ErrConnDone)

    // Fallback non-paginated returns one row
    rec := &models.Receipt{ExecutionDetails: models.ExecutionDetails{ProviderID: "p1", Region: "us", Status: "completed"}}
    mock.ExpectQuery("SELECT e.receipt_data\n\t\tFROM executions e\n\t\tJOIN jobs j ON e.job_id = j.id\n\t\tWHERE j.jobspec_id = \$1 AND e.receipt_data IS NOT NULL\n\t\tORDER BY e.created_at DESC").
        WithArgs("js-all").
        WillReturnRows(sqlmock.NewRows([]string{"receipt_data"}).AddRow(mustReceiptJSON(t, rec)))

    h := &JobsHandler{jobsService: js}
    r := setupRouter(h)

    rr := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/jobs/js-all?include=all", nil)
    r.ServeHTTP(rr, req)

    if rr.Code != http.StatusOK {
        t.Fatalf("status=%d", rr.Code)
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}
