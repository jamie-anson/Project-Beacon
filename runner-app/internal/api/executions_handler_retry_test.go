package api

import (
    "bytes"
    "database/sql"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "os"
    "testing"

    sqlmock "github.com/DATA-DOG/go-sqlmock"
    miniredis "github.com/alicebob/miniredis/v2"
    "github.com/gin-gonic/gin"
    "github.com/jamie-anson/project-beacon-runner/internal/store"
)

func setupGin() *gin.Engine {
    gin.SetMode(gin.TestMode)
    r := gin.New()
    return r
}

func TestRetryQuestion_NotFound(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("failed to create sqlmock: %v", err)
    }
    defer db.Close()

    repo := &store.ExecutionsRepo{DB: db}
    h := &ExecutionsHandler{ExecutionsRepo: repo, RetryService: nil}

    mock.ExpectQuery("SELECT\\s+e.retry_count").
        WithArgs(int64(4040), "Europe").
        WillReturnError(sql.ErrNoRows)

    r := setupGin()
    r.POST("/api/v1/executions/:id/retry-question", h.RetryQuestion)

    body := bytes.NewBufferString(`{"region":"Europe","question_index":0}`)
    req, _ := http.NewRequest(http.MethodPost, "/api/v1/executions/4040/retry-question", body)
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusNotFound {
        t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
    }

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

func TestRetryQuestion_DedupeCollision_Miniredis(t *testing.T) {
    // Start miniredis
    mr, err := miniredis.Run()
    if err != nil {
        t.Fatalf("failed to start miniredis: %v", err)
    }
    defer mr.Close()
    os.Setenv("REDIS_URL", "redis://"+mr.Addr())
    defer os.Unsetenv("REDIS_URL")

    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("failed to create sqlmock: %v", err)
    }
    defer db.Close()

    repo := &store.ExecutionsRepo{DB: db}
    h := &ExecutionsHandler{ExecutionsRepo: repo, RetryService: nil}

    // First call: status failed -> will UPDATE
    rows1 := sqlmock.NewRows([]string{"retry_count", "max_retries", "status", "job_id", "jobspec_id", "model_id", "question_id"}).
        AddRow(0, 3, "failed", 42, "job-123", "llama3.2-1b", "q1")
    mock.ExpectQuery("SELECT\\s+e.retry_count").
        WithArgs(int64(5555), "US").
        WillReturnRows(rows1)
    mock.ExpectExec("UPDATE executions ").
        WithArgs(1, sqlmock.AnyArg(), int64(5555)).
        WillReturnResult(sqlmock.NewResult(0, 1))

    r := setupGin()
    r.POST("/api/v1/executions/:id/retry-question", h.RetryQuestion)

    body := bytes.NewBufferString(`{"region":"US","question_index":0}`)
    req1, _ := http.NewRequest(http.MethodPost, "/api/v1/executions/5555/retry-question", body)
    req1.Header.Set("Content-Type", "application/json")
    w1 := httptest.NewRecorder()
    r.ServeHTTP(w1, req1)
    if w1.Code != http.StatusOK {
        t.Fatalf("first call expected 200, got %d: %s", w1.Code, w1.Body.String())
    }

    // Second call immediately: status still failed in DB, but Redis key should block -> expect 202 and NO UPDATE
    rows2 := sqlmock.NewRows([]string{"retry_count", "max_retries", "status", "job_id", "jobspec_id", "model_id", "question_id"}).
        AddRow(0, 3, "failed", 42, "job-123", "llama3.2-1b", "q1")
    mock.ExpectQuery("SELECT\\s+e.retry_count").
        WithArgs(int64(5555), "US").
        WillReturnRows(rows2)

    body2 := bytes.NewBufferString(`{"region":"US","question_index":0}`)
    req2, _ := http.NewRequest(http.MethodPost, "/api/v1/executions/5555/retry-question", body2)
    req2.Header.Set("Content-Type", "application/json")
    w2 := httptest.NewRecorder()
    r.ServeHTTP(w2, req2)
    if w2.Code != http.StatusAccepted {
        t.Fatalf("second call expected 202, got %d: %s", w2.Code, w2.Body.String())
    }

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

func TestRetryQuestion_ShortCircuitCompleted(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("failed to create sqlmock: %v", err)
    }
    defer db.Close()

    repo := &store.ExecutionsRepo{DB: db}
    h := &ExecutionsHandler{ExecutionsRepo: repo, RetryService: nil}

    // Primary status lookup
    rows := sqlmock.NewRows([]string{"retry_count", "max_retries", "status", "job_id", "jobspec_id", "model_id", "question_id"}).
        AddRow(1, 3, "completed", 42, "job-123", "llama3.2-1b", "q1")
    mock.ExpectQuery("SELECT\\s+e.retry_count").
        WithArgs(int64(1001), "Europe").
        WillReturnRows(rows)

    // output_data fetch
    output := map[string]any{"response": "done"}
    b, _ := json.Marshal(output)
    mock.ExpectQuery("SELECT output_data FROM executions WHERE id = \\$1").
        WithArgs(int64(1001)).
        WillReturnRows(sqlmock.NewRows([]string{"output_data"}).AddRow(b))

    r := setupGin()
    r.POST("/api/v1/executions/:id/retry-question", h.RetryQuestion)

    body := bytes.NewBufferString(`{"region":"Europe","question_index":0}`)
    req, _ := http.NewRequest(http.MethodPost, "/api/v1/executions/1001/retry-question", body)
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
    }
    var resp map[string]any
    _ = json.Unmarshal(w.Body.Bytes(), &resp)
    if resp["status"] != "completed" {
        t.Fatalf("expected status completed, got %v", resp["status"])
    }
    if _, ok := resp["result"].(map[string]any); !ok {
        t.Fatalf("expected result payload present")
    }

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

func TestRetryQuestion_ShortCircuitRunning(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("failed to create sqlmock: %v", err)
    }
    defer db.Close()

    repo := &store.ExecutionsRepo{DB: db}
    h := &ExecutionsHandler{ExecutionsRepo: repo, RetryService: nil}

    rows := sqlmock.NewRows([]string{"retry_count", "max_retries", "status", "job_id", "jobspec_id", "model_id", "question_id"}).
        AddRow(0, 3, "running", 42, "job-123", "mistral-7b", "q1")
    mock.ExpectQuery("SELECT\\s+e.retry_count").
        WithArgs(int64(2002), "US").
        WillReturnRows(rows)

    r := setupGin()
    r.POST("/api/v1/executions/:id/retry-question", h.RetryQuestion)

    body := bytes.NewBufferString(`{"region":"US","question_index":1}`)
    req, _ := http.NewRequest(http.MethodPost, "/api/v1/executions/2002/retry-question", body)
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusAccepted {
        t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
    }

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

func TestRetryQuestion_RetryOnFailed(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("failed to create sqlmock: %v", err)
    }
    defer db.Close()

    repo := &store.ExecutionsRepo{DB: db}
    h := &ExecutionsHandler{ExecutionsRepo: repo, RetryService: nil}

    // status row -> failed
    rows := sqlmock.NewRows([]string{"retry_count", "max_retries", "status", "job_id", "jobspec_id", "model_id", "question_id"}).
        AddRow(1, 3, "failed", 42, "job-123", "llama3.2-1b", "q1")
    mock.ExpectQuery("SELECT\\s+e.retry_count").
        WithArgs(int64(3003), "Europe").
        WillReturnRows(rows)

    // UPDATE executions ... set retry_count, last_retry_at, retry_history, status='retrying'
    mock.ExpectExec("UPDATE executions ").
        WithArgs(2, sqlmock.AnyArg(), int64(3003)).
        WillReturnResult(sqlmock.NewResult(0, 1))

    r := setupGin()
    r.POST("/api/v1/executions/:id/retry-question", h.RetryQuestion)

    body := bytes.NewBufferString(`{"region":"Europe","question_index":0}`)
    req, _ := http.NewRequest(http.MethodPost, "/api/v1/executions/3003/retry-question", body)
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
    }

    var resp map[string]any
    _ = json.Unmarshal(w.Body.Bytes(), &resp)
    if resp["status"] != "retrying" {
        t.Fatalf("expected status retrying, got %v", resp["status"])
    }

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}
