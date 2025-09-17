//go:build !ci
// +build !ci

package worker

import (
    "context"
    "encoding/json"
    "errors"
    "regexp"
    "testing"
    "time"

    sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestHandleEnvelope_GetJob_DBError(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    // Expect UPDATE to set status to processing (this happens first)
    mock.ExpectExec(regexp.QuoteMeta("UPDATE jobs SET status = $1, updated_at = NOW() WHERE jobspec_id = $2")).
        WithArgs("processing", "job-err").
        WillReturnResult(sqlmock.NewResult(0, 1))
    
    // Force SELECT to error
    mock.ExpectQuery(regexp.QuoteMeta("SELECT jobspec_id, status, jobspec_data, created_at, updated_at FROM jobs WHERE jobspec_id = $1")).
        WithArgs("job-err").
        WillReturnError(errors.New("db down"))

    jr := NewJobRunner(db, nil, nil, nil)
    ctx := context.Background()
    env := map[string]any{"id": "job-err", "enqueued_at": time.Now(), "attempt": 1}
    b, _ := json.Marshal(env)
    err := jr.handleEnvelope(ctx, b)
    if err == nil || err.Error()[:8] != "get job:" {
        t.Fatalf("expected get job error, got %v", err)
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

func TestHandleEnvelope_ExecutionFail_InsertError(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    // JobsRepo.GetJob returns a spec with one region and tiny timeout to force ExecuteSingleRegion error
    js := buildJobSpecJSON("job-ins-fail", []string{"US"}, 1*time.Millisecond)
    rows := sqlmock.NewRows([]string{
        "jobspec_id",
        "status",
        "jobspec_data",
        "created_at",
        "updated_at",
    }).
        AddRow("job-ins-fail", "queued", js, time.Now(), time.Now())
    mock.ExpectQuery(regexp.QuoteMeta("SELECT jobspec_id, status, jobspec_data, created_at, updated_at FROM jobs WHERE jobspec_id = $1")).
        WithArgs("job-ins-fail").
        WillReturnRows(rows)

    // INSERT for failed execution returns error we expect to be propagated
    mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO executions (job_id, provider_id, region, status, started_at, completed_at, output_data, receipt_data) VALUES ((SELECT id FROM jobs WHERE jobspec_id = $1), $2, $3, $4, $5, $6, $7, $8) RETURNING id")).
        WithArgs("job-ins-fail", sqlmock.AnyArg(), "US", "failed", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
        WillReturnError(errors.New("insert failed"))

    jr := NewJobRunner(db, nil, golemSvcForTest(), nil)
    ctx := context.Background()
    t.Setenv("VALIDATION_SKIP_SIGNATURE", "true")
    env := map[string]any{"id": "job-ins-fail", "enqueued_at": time.Now().Add(-2 * time.Second), "attempt": 1}
    b, _ := json.Marshal(env)

    err := jr.handleEnvelope(ctx, b)
    if err == nil || err.Error() != "insert execution: insert failed" {
        t.Fatalf("expected wrapped insert error from execution failure path, got %v", err)
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

func TestHandleEnvelope_Success_InsertError(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    js := buildJobSpecJSON("job-succ-ins-fail", []string{"US"}, 5*time.Second)
    rows := sqlmock.NewRows(
        []string{
            "jobspec_id",
            "status",
            "jobspec_data",
            "created_at",
            "updated_at",
        },
    ).
        AddRow("job-succ-ins-fail", "queued", js, time.Now(), time.Now())
    mock.ExpectQuery(regexp.QuoteMeta("SELECT jobspec_id, status, jobspec_data, created_at, updated_at FROM jobs WHERE jobspec_id = $1")).
        WithArgs("job-succ-ins-fail").
        WillReturnRows(rows)

    // Expect the INSERT to return an error that is wrapped as "insert execution"
    mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO executions (job_id, provider_id, region, status, started_at, completed_at, output_data, receipt_data) VALUES ((SELECT id FROM jobs WHERE jobspec_id = $1), $2, $3, $4, $5, $6, $7, $8) RETURNING id")).
        WithArgs("job-succ-ins-fail", sqlmock.AnyArg(), "US", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
        WillReturnError(errors.New("db write error"))

    jr := NewJobRunner(db, nil, golemSvcForTest(), nil)
    ctx := context.Background()
    t.Setenv("VALIDATION_SKIP_SIGNATURE", "true")
    env := map[string]any{"id": "job-succ-ins-fail", "enqueued_at": time.Now().Add(-1 * time.Second), "attempt": 1}
    b, _ := json.Marshal(env)

    err := jr.handleEnvelope(ctx, b)
    if err == nil || err.Error()[:17] != "insert execution:" {
        t.Fatalf("expected wrapped insert execution error, got %v", err)
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}
