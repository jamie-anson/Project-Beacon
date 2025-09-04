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

func TestCreateJob_Success_NoRequestID_PayloadOmitsValue(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()
    svc := NewJobsService(db)

    mock.ExpectBegin()

    mock.ExpectExec(regexp.QuoteMeta(`
        INSERT INTO jobs (jobspec_id, jobspec_data, status)
        VALUES ($1, $2, $3)
        ON CONFLICT (jobspec_id)
        DO UPDATE SET jobspec_data = EXCLUDED.jobspec_data, status = EXCLUDED.status, updated_at = NOW()`)).
        WithArgs("job-no-reqid", sqlmock.AnyArg(), "created").
        WillReturnResult(sqlmock.NewResult(1, 1))

    // request_id should be empty string in payload
    mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO outbox (topic, payload) VALUES ($1, $2)`)).
        WithArgs(svc.QueueName, payloadMatcher{wantID: "job-no-reqid", wantRequestID: ""}).
        WillReturnResult(sqlmock.NewResult(1, 1))

    mock.ExpectCommit()

    spec := &models.JobSpec{ID: "job-no-reqid", Version: "1.0.0", CreatedAt: time.Now()}
    js, _ := json.Marshal(spec)

    if err := svc.CreateJob(context.Background(), spec, js); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}
