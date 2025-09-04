package service

import (
    "context"
    "database/sql/driver"
    "encoding/json"
    "regexp"
    "testing"
    "time"

    sqlmock "github.com/DATA-DOG/go-sqlmock"
    "github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// payloadMatcher verifies the outbox payload JSON contains expected fields
// like id and request_id when propagated from context.
type payloadMatcher struct{
    wantID string
    wantRequestID string
}

func (m payloadMatcher) Match(v driver.Value) bool {
    b, ok := v.([]byte)
    if !ok {
        // if driver provides string, try to coerce
        if s, ok2 := v.(string); ok2 {
            b = []byte(s)
        } else {
            return false
        }
    }
    var obj map[string]any
    if err := json.Unmarshal(b, &obj); err != nil {
        return false
    }
    if id, ok := obj["id"].(string); !ok || id != m.wantID {
        return false
    }
    if rid, ok := obj["request_id"].(string); !ok || rid != m.wantRequestID {
        return false
    }
    // attempt field should exist and be numeric (0 for first enqueue)
    if _, ok := obj["attempt"]; !ok {
        return false
    }
    return true
}

func TestCreateJob_Success_CommitsAndPublishesOutbox(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()
    svc := NewJobsService(db)

    // Expect transaction begin
    mock.ExpectBegin()

    // Expect jobs upsert (matches existing tests' SQL)
    mock.ExpectExec(regexp.QuoteMeta(`
        INSERT INTO jobs (jobspec_id, jobspec_data, status)
        VALUES ($1, $2, $3)
        ON CONFLICT (jobspec_id)
        DO UPDATE SET jobspec_data = EXCLUDED.jobspec_data, status = EXCLUDED.status, updated_at = NOW()`)).
        WithArgs("job-success", sqlmock.AnyArg(), "created").
        WillReturnResult(sqlmock.NewResult(1, 1))

    // Expect outbox insert; verify topic equals svc.QueueName and payload carries request_id
    mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO outbox (topic, payload) VALUES ($1, $2)`)).
        WithArgs(svc.QueueName, payloadMatcher{wantID: "job-success", wantRequestID: "req-123"}).
        WillReturnResult(sqlmock.NewResult(1, 1))

    // Expect commit
    mock.ExpectCommit()

    spec := &models.JobSpec{ID: "job-success", Version: "1.0.0", CreatedAt: time.Now()}
    js, _ := json.Marshal(spec)

    ctx := context.WithValue(context.Background(), "request_id", "req-123")

    if err := svc.CreateJob(ctx, spec, js); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

func TestCreateJob_BeginTxError_Returns(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()
    svc := NewJobsService(db)

    mock.ExpectBegin().WillReturnError(assertErr("begin err"))

    spec := &models.JobSpec{ID: "job-begin-err", Version: "1.0.0", CreatedAt: time.Now()}
    js, _ := json.Marshal(spec)

    if err := svc.CreateJob(context.Background(), spec, js); err == nil {
        t.Fatalf("expected error, got nil")
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

// assertErr implements error with a fixed message for sqlmock expectations
type assertErr string
func (e assertErr) Error() string { return string(e) }
