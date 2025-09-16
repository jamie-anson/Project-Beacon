//go:build !ci
// +build !ci

package worker

import (
    "context"
    "encoding/json"
    "regexp"
    "testing"
    "time"

    sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestHandleEnvelope_Success_InsertsCompletedExecution(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    // JobsRepo.GetJob returns a valid spec with one region US and ample timeout
    js := buildJobSpecJSON("job-succ", []string{"US"}, 5*time.Second)
    rows := sqlmock.NewRows([]string{"jobspec_id", "status", "jobspec_data", "created_at", "updated_at"}).
        AddRow("job-succ", "queued", js, time.Now(), time.Now())
    mock.ExpectQuery(regexp.QuoteMeta("SELECT jobspec_id, status, jobspec_data, created_at, updated_at FROM jobs WHERE jobspec_id = $1")).
        WithArgs("job-succ").
        WillReturnRows(rows)

    // Expect INSERT into executions with status completed for region US
    mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO executions (job_id, provider_id, region, status, started_at, completed_at, output_data, receipt_data) VALUES ((SELECT id FROM jobs WHERE jobspec_id = $1), $2, $3, $4, $5, $6, $7, $8) RETURNING id")).
        WithArgs("job-succ", sqlmock.AnyArg(), "US", "completed", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
        WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(7)))

    jr := NewJobRunner(db, nil, golemSvcForTest(), nil)
    ctx := context.Background()
    t.Setenv("VALIDATION_SKIP_SIGNATURE", "true")
    env := map[string]any{"id": "job-succ", "enqueued_at": time.Now().Add(-2 * time.Second), "attempt": 1}
    b, _ := json.Marshal(env)

    if err := jr.handleEnvelope(ctx, b); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}
