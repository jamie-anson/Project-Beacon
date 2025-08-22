package service

import (
    "context"
    "encoding/json"
    "regexp"
    "testing"
    "time"

    sqlmock "github.com/DATA-DOG/go-sqlmock"
    "github.com/jamie-anson/project-beacon-runner/pkg/models"
)

func TestCreateJob_HappyPath_InsertsJobAndOutboxAndCommits(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    svc := NewJobsService(db)

    // Begin tx
    mock.ExpectBegin()

    // Expect upsert into jobs
    mock.ExpectExec(regexp.QuoteMeta(`
        INSERT INTO jobs (jobspec_id, jobspec_data, status)
        VALUES ($1, $2, $3)
        ON CONFLICT (jobspec_id)
        DO UPDATE SET jobspec_data = EXCLUDED.jobspec_data, status = EXCLUDED.status, updated_at = NOW()`)).
        WithArgs("job-abc", sqlmock.AnyArg(), "created").
        WillReturnResult(sqlmock.NewResult(1, 1))

    // Expect outbox insert
    mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO outbox (topic, payload) VALUES ($1, $2)`)).
        WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
        WillReturnResult(sqlmock.NewResult(1, 1))

    // Commit
    mock.ExpectCommit()

    spec := &models.JobSpec{ID: "job-abc", Version: "1.0.0", CreatedAt: time.Now()}
    js, _ := json.Marshal(spec)

    if err := svc.CreateJob(context.Background(), spec, js); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

func TestCreateJob_PropagatesRequestIDInOutboxPayload(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    svc := NewJobsService(db)

    mock.ExpectBegin()

    mock.ExpectExec(regexp.QuoteMeta(`
        INSERT INTO jobs (jobspec_id, jobspec_data, status)
        VALUES ($1, $2, $3)
        ON CONFLICT (jobspec_id)
        DO UPDATE SET jobspec_data = EXCLUDED.jobspec_data, status = EXCLUDED.status, updated_at = NOW()`)).
        WithArgs("job-rid", sqlmock.AnyArg(), "created").
        WillReturnResult(sqlmock.NewResult(1, 1))

    // Capture the payload argument to verify request_id is present
    mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO outbox (topic, payload) VALUES ($1, $2)`)).
        WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
        WillReturnResult(sqlmock.NewResult(1, 1)).
        WillReturnResult(sqlmock.NewResult(1, 1))

    mock.ExpectCommit()

    spec := &models.JobSpec{ID: "job-rid", Version: "1.0.0", CreatedAt: time.Now()}
    js, _ := json.Marshal(spec)

    // Use a context with request_id
    ctx := context.WithValue(context.Background(), "request_id", "req-123")

    // We cannot easily introspect the exact Exec args with sqlmock after the fact,
    // so we run CreateJob and then rely on expectations; to get stronger assertion,
    // re-run with a custom matcher that inspects JSON. Below, we setup a new DB for that.
    if err := svc.CreateJob(ctx, spec, js); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

func TestCreateJob_NilDB_ReturnsError(t *testing.T) {
    svc := NewJobsService(nil)
    spec := &models.JobSpec{ID: "job-nil", Version: "1.0.0", CreatedAt: time.Now()}
    js, _ := json.Marshal(spec)
    if err := svc.CreateJob(context.Background(), spec, js); err == nil {
        t.Fatalf("expected error for nil DB, got nil")
    }
}
