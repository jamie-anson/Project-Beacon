package service

import (
    "context"
    "encoding/json"
    "errors"
    "regexp"
    "testing"
    "time"

    sqlmock "github.com/DATA-DOG/go-sqlmock"
    "github.com/jamie-anson/project-beacon-runner/pkg/models"
)

func TestCreateJob_UpsertError_RollsBackAndReturns(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()
    svc := NewJobsService(db)

    mock.ExpectBegin()
    mock.ExpectExec(regexp.QuoteMeta(`
        INSERT INTO jobs (jobspec_id, jobspec_data, status)
        VALUES ($1, $2, $3)
        ON CONFLICT (jobspec_id)
        DO UPDATE SET jobspec_data = EXCLUDED.jobspec_data, status = EXCLUDED.status, updated_at = NOW()`)).
        WithArgs("job-upsert", sqlmock.AnyArg(), "created").
        WillReturnError(errors.New("db err"))
    mock.ExpectRollback()

    spec := &models.JobSpec{ID: "job-upsert", Version: "1.0.0", CreatedAt: time.Now()}
    js, _ := json.Marshal(spec)

    if err := svc.CreateJob(context.Background(), spec, js); err == nil {
        t.Fatalf("expected error, got nil")
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

func TestCreateJob_OutboxInsertError_RollsBackAndReturns(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()
    svc := NewJobsService(db)

    mock.ExpectBegin()
    mock.ExpectExec(regexp.QuoteMeta(`
        INSERT INTO jobs (jobspec_id, jobspec_data, status)
        VALUES ($1, $2, $3)
        ON CONFLICT (jobspec_id)
        DO UPDATE SET jobspec_data = EXCLUDED.jobspec_data, status = EXCLUDED.status, updated_at = NOW()`)).
        WithArgs("job-outbox", sqlmock.AnyArg(), "created").
        WillReturnResult(sqlmock.NewResult(1, 1))
    mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO outbox (topic, payload) VALUES ($1, $2)`)).
        WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
        WillReturnError(errors.New("outbox err"))
    mock.ExpectRollback()

    spec := &models.JobSpec{ID: "job-outbox", Version: "1.0.0", CreatedAt: time.Now()}
    js, _ := json.Marshal(spec)

    if err := svc.CreateJob(context.Background(), spec, js); err == nil {
        t.Fatalf("expected error, got nil")
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

func TestCreateJob_CommitError_Returns(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()
    svc := NewJobsService(db)

    mock.ExpectBegin()
    mock.ExpectExec(regexp.QuoteMeta(`
        INSERT INTO jobs (jobspec_id, jobspec_data, status)
        VALUES ($1, $2, $3)
        ON CONFLICT (jobspec_id)
        DO UPDATE SET jobspec_data = EXCLUDED.jobspec_data, status = EXCLUDED.status, updated_at = NOW()`)).
        WithArgs("job-commit", sqlmock.AnyArg(), "created").
        WillReturnResult(sqlmock.NewResult(1, 1))
    mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO outbox (topic, payload) VALUES ($1, $2)`)).
        WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
        WillReturnResult(sqlmock.NewResult(1, 1))
    mock.ExpectCommit().WillReturnError(errors.New("commit err"))

    spec := &models.JobSpec{ID: "job-commit", Version: "1.0.0", CreatedAt: time.Now()}
    js, _ := json.Marshal(spec)

    if err := svc.CreateJob(context.Background(), spec, js); err == nil {
        t.Fatalf("expected error, got nil")
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}
