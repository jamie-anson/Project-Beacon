package worker

import (
    "context"
    "encoding/json"
    "regexp"
    "testing"
    "time"

    sqlmock "github.com/DATA-DOG/go-sqlmock"
)

// Success path: publishes one row, marks published
func TestOutboxPublisher_SuccessPath_MarksPublished(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    payload := map[string]any{"id": "job-ok", "enqueued_at": time.Now().UTC(), "attempt": 1}
    pb, _ := json.Marshal(payload)

    // Fetch one unpublished row
    rows := sqlmock.NewRows([]string{"id", "topic", "payload"}).AddRow(int64(10), "jobs", pb)
    mock.ExpectQuery(regexp.QuoteMeta("SELECT id, topic, payload\n\t\tFROM outbox\n\t\tWHERE published_at IS NULL\n\t\tORDER BY id ASC\n\t\tLIMIT $1")).
        WithArgs(100).
        WillReturnRows(rows)

    // Expect mark published
    mock.ExpectExec(regexp.QuoteMeta("UPDATE outbox SET published_at = NOW() WHERE id = $1")).
        WithArgs(int64(10)).
        WillReturnResult(sqlmock.NewResult(0, 1))

    // No metrics query expected in same loop since publishedAny==true; we will cancel immediately

    // Redis
    mr, qc, _ := newTestQueue(t)
    defer mr.Close()
    defer qc.Close()

    p := NewOutboxPublisher(db, qc)
    ctx, cancel := context.WithCancel(context.Background())

    done := make(chan struct{})
    go func(){
        p.Start(ctx)
        close(done)
    }()

    // Give it a moment to run one iteration, then cancel
    time.Sleep(150 * time.Millisecond)
    cancel()

    select {
    case <-done:
    case <-time.After(2 * time.Second):
        t.Fatal("publisher did not stop in time")
    }

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

// Enqueue failure: Redis error means no mark published; metrics stats query expected
func TestOutboxPublisher_EnqueueFailure_DoesNotMarkPublished(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    payload := map[string]any{"id": "job-fail", "enqueued_at": time.Now().UTC(), "attempt": 1}
    pb, _ := json.Marshal(payload)

    // One row
    rows := sqlmock.NewRows([]string{"id", "topic", "payload"}).AddRow(int64(11), "jobs", pb)
    mock.ExpectQuery(regexp.QuoteMeta("SELECT id, topic, payload\n\t\tFROM outbox\n\t\tWHERE published_at IS NULL\n\t\tORDER BY id ASC\n\t\tLIMIT $1")).
        WithArgs(100).
        WillReturnRows(rows)

    // After failure, loop no publish -> metrics query
    mock.ExpectQuery(regexp.QuoteMeta("SELECT \n\t\t\tCOUNT(*) as count,\n\t\t\tCOALESCE(EXTRACT(EPOCH FROM (NOW() - MIN(created_at))), 0) as oldest_age_seconds\n\t\tFROM outbox \n\t\tWHERE published_at IS NULL")).
        WillReturnRows(sqlmock.NewRows([]string{"count", "oldest_age_seconds"}).AddRow(1, 0))

    // Redis setup then force failure by closing before enqueue
    mr, qc, _ := newTestQueue(t)

    // Close server to make RPush fail
    mr.Close()
    defer qc.Close()

    p := NewOutboxPublisher(db, qc)
    ctx, cancel := context.WithCancel(context.Background())

    go func(){
        // allow a little time to attempt and then cancel
        time.Sleep(200 * time.Millisecond)
        cancel()
    }()

    p.Start(ctx)

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}
