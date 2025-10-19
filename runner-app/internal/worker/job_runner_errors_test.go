//go:build !ci
// +build !ci

package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/jamie-anson/project-beacon-runner/internal/queue"
	"github.com/stretchr/testify/require"
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

	jr := newTestJobRunner(t, db)
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

func newTestJobRunner(t *testing.T, db *sql.DB) *JobRunner {
	t.Helper()

	mini, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mini.Close)

	t.Setenv("REDIS_URL", "redis://"+mini.Addr())

	q, err := queue.NewFromEnv()
	require.NoError(t, err)
	t.Cleanup(func() { _ = q.Close() })

	jr := NewJobRunner(db, q, golemSvcForTest(), nil)
	// Disable ExecutionSvc interactions; tests cover job repo directly with sqlmock
	jr.ExecutionSvc = nil
	return jr
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
	mock.ExpectExec(regexp.QuoteMeta("UPDATE jobs SET status = $1, updated_at = NOW() WHERE jobspec_id = $2")).
		WithArgs("processing", "job-ins-fail").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT jobspec_id, status, jobspec_data, created_at, updated_at FROM jobs WHERE jobspec_id = $1")).
		WithArgs("job-ins-fail").
		WillReturnRows(rows)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM jobs WHERE jobspec_id = $1")).
		WithArgs("job-ins-fail").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

	// INSERT for failed execution returns error we expect to be propagated
	mock.ExpectQuery("INSERT INTO executions \\(job_id, provider_id, region, status, started_at, completed_at, output_data, receipt_data, model_id, question_id\\)\\s+VALUES \\(\\$1, \\$2, \\$3, \\$4, \\$5, \\$6, \\$7, \\$8, \\$9, \\$10\\)\\s+RETURNING id").
		WithArgs(int64(1), sqlmock.AnyArg(), "US", "failed", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(errors.New("insert failed"))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE jobs SET status = $1, updated_at = NOW() WHERE jobspec_id = $2")).
		WithArgs("failed", "job-ins-fail").
		WillReturnResult(sqlmock.NewResult(0, 1))

	jr := newTestJobRunner(t, db)
	ctx := context.Background()
	t.Setenv("VALIDATION_SKIP_SIGNATURE", "true")
	env := map[string]any{"id": "job-ins-fail", "enqueued_at": time.Now().Add(-2 * time.Second), "attempt": 1}
	b, _ := json.Marshal(env)

	err := jr.handleEnvelope(ctx, b)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
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
	mock.ExpectExec(regexp.QuoteMeta("UPDATE jobs SET status = $1, updated_at = NOW() WHERE jobspec_id = $2")).
		WithArgs("processing", "job-succ-ins-fail").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT jobspec_id, status, jobspec_data, created_at, updated_at FROM jobs WHERE jobspec_id = $1")).
		WithArgs("job-succ-ins-fail").
		WillReturnRows(rows)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM jobs WHERE jobspec_id = $1")).
		WithArgs("job-succ-ins-fail").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

	// Expect the INSERT to return an error that is wrapped as "insert execution"
	mock.ExpectQuery("INSERT INTO executions \\(job_id, provider_id, region, status, started_at, completed_at, output_data, receipt_data, model_id, question_id\\)\\s+VALUES \\(\\$1, \\$2, \\$3, \\$4, \\$5, \\$6, \\$7, \\$8, \\$9, \\$10\\)\\s+RETURNING id").
		WithArgs(int64(1), sqlmock.AnyArg(), "US", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(errors.New("db write error"))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE jobs SET status = $1, updated_at = NOW() WHERE jobspec_id = $2")).
		WithArgs("completed", "job-succ-ins-fail").
		WillReturnResult(sqlmock.NewResult(0, 1))

	jr := newTestJobRunner(t, db)
	ctx := context.Background()
	t.Setenv("VALIDATION_SKIP_SIGNATURE", "true")
	env := map[string]any{"id": "job-succ-ins-fail", "enqueued_at": time.Now().Add(-1 * time.Second), "attempt": 1}
	b, _ := json.Marshal(env)

	err := jr.handleEnvelope(ctx, b)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
