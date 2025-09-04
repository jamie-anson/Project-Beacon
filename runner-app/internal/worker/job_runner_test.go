package worker

import (
    "context"
    "encoding/json"
    "regexp"
    "testing"
    "time"

    sqlmock "github.com/DATA-DOG/go-sqlmock"
    "github.com/jamie-anson/project-beacon-runner/internal/golem"
)

// helper: build a minimal schema-compliant JobSpec JSON
func buildJobSpecJSON(id string, regions []string, timeout time.Duration) []byte {
    // durations are encoded as nanoseconds (int) for time.Duration fields
    if timeout <= 0 {
        timeout = 1 * time.Second
    }
    js := map[string]any{
        "id":      id,
        "version": "1.0.0",
        "benchmark": map[string]any{
            "name":        "Who Are You?",
            "description": "",
            "container": map[string]any{
                "image": "alpine",
                "tag":   "3",
                "resources": map[string]any{
                    "cpu":    "1000m",
                    "memory": "512Mi",
                },
            },
            "input": map[string]any{
                "type": "prompt",
                "data": map[string]any{"prompt": "hi"},
                "hash": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            },
            "scoring": map[string]any{
                "method":     "similarity",
                "parameters": map[string]any{},
            },
        },
        "constraints": map[string]any{
            "regions":          regions,
            "min_regions":      maxInt(1, len(regions)),
            "min_success_rate": 0.67,
            "timeout":          int64(timeout),
            "provider_timeout": int64(time.Second * 30),
        },
        "metadata":  map[string]any{},
        "created_at": time.Now().Format(time.RFC3339),
        // signature/public_key required by VerifySignature but will be skipped via env
        "signature":  "skip",
        "public_key": "skip",
    }
    b, _ := json.Marshal(js)
    return b
}

func maxInt(a, b int) int { if a > b { return a }; return b }

func TestHandleEnvelope_InvalidJSON(t *testing.T) {
    db, _, _ := sqlmock.New()
    defer db.Close()

    jr := NewJobRunner(db, nil, nil, nil)
    ctx := context.Background()
    err := jr.handleEnvelope(ctx, []byte("{not-json}"))
    if err == nil || err.Error() == "" {
        t.Fatalf("expected invalid envelope error, got %v", err)
    }
}

func TestHandleEnvelope_MissingID(t *testing.T) {
    db, _, _ := sqlmock.New()
    defer db.Close()

    jr := NewJobRunner(db, nil, nil, nil)
    ctx := context.Background()
    env := map[string]any{
        "enqueued_at": time.Now(),
        "attempt":     1,
    }
    b, _ := json.Marshal(env)
    err := jr.handleEnvelope(ctx, b)
    if err == nil || err.Error() != "missing job id in envelope" {
        t.Fatalf("expected missing id error, got %v", err)
    }
}

func TestHandleEnvelope_EmptyJobSpec(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    // JobsRepo.GetJob query expectation
    rows := sqlmock.NewRows([]string{"jobspec_id", "status", "jobspec_data", "created_at", "updated_at"}).
        AddRow("job-1", "queued", []byte(nil), time.Now(), time.Now())
    mock.ExpectQuery(regexp.QuoteMeta("SELECT jobspec_id, status, jobspec_data, created_at, updated_at FROM jobs WHERE jobspec_id = $1")).
        WithArgs("job-1").
        WillReturnRows(rows)

    jr := NewJobRunner(db, nil, nil, nil)
    ctx := context.Background()
    env := map[string]any{"id": "job-1", "enqueued_at": time.Now(), "attempt": 1}
    b, _ := json.Marshal(env)
    err := jr.handleEnvelope(ctx, b)
    if err == nil || err.Error() != "empty jobspec JSON for job-1" {
        t.Fatalf("expected empty jobspec error, got %v", err)
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

func TestHandleEnvelope_NoRegions(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    js := buildJobSpecJSON("job-2", []string{}, time.Second)
    rows := sqlmock.NewRows([]string{"jobspec_id", "status", "jobspec_data", "created_at", "updated_at"}).
        AddRow("job-2", "queued", js, time.Now(), time.Now())
    mock.ExpectQuery(regexp.QuoteMeta("SELECT jobspec_id, status, jobspec_data, created_at, updated_at FROM jobs WHERE jobspec_id = $1")).
        WithArgs("job-2").
        WillReturnRows(rows)

    jr := NewJobRunner(db, nil, nil, nil)
    ctx := context.Background()
    t.Setenv("VALIDATION_SKIP_SIGNATURE", "true")
    env := map[string]any{"id": "job-2", "enqueued_at": time.Now(), "attempt": 1}
    b, _ := json.Marshal(env)
    err := jr.handleEnvelope(ctx, b)
    if err == nil || err.Error() != "jobspec validate: validation failed: at least one region constraint is required" {
        t.Fatalf("expected no regions error, got %v", err)
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

func TestHandleEnvelope_ExecutionFail_PersistsFailedExecution(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    // JobsRepo.GetJob returns a spec with one region and tiny timeout to force cancellation
    js := buildJobSpecJSON("job-3", []string{"US"}, 1*time.Millisecond)
    rows := sqlmock.NewRows([]string{"jobspec_id", "status", "jobspec_data", "created_at", "updated_at"}).
        AddRow("job-3", "queued", js, time.Now(), time.Now())
    mock.ExpectQuery(regexp.QuoteMeta("SELECT jobspec_id, status, jobspec_data, created_at, updated_at FROM jobs WHERE jobspec_id = $1")).
        WithArgs("job-3").
        WillReturnRows(rows)

    // Expect INSERT into executions (legacy InsertExecution path)
    mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO executions (job_id, provider_id, region, status, started_at, completed_at, output_data, receipt_data) VALUES ((SELECT id FROM jobs WHERE jobspec_id = $1), $2, $3, $4, $5, $6, $7, $8) RETURNING id")).
        WithArgs("job-3", sqlmock.AnyArg(), "US", "failed", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
        WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))

    jr := NewJobRunner(db, nil, golemSvcForTest(), nil)
    ctx := context.Background()
    t.Setenv("VALIDATION_SKIP_SIGNATURE", "true")
    env := map[string]any{"id": "job-3", "enqueued_at": time.Now().Add(-2 * time.Second), "attempt": 1}
    b, _ := json.Marshal(env)
    // Expect no error returned from handleEnvelope even if execution failed (it returns insert error only)
    if err := jr.handleEnvelope(ctx, b); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

// golemSvcForTest returns a default golem.Service suitable for ExecuteSingleRegion invocation
func golemSvcForTest() *golem.Service {
    // Use default config (mock backend); no special setup needed
    return golem.NewService("", "testnet")
}
