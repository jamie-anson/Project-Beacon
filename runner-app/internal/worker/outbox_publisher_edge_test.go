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

func TestOutboxPublisher_FetchError_ThenMetrics(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    // First loop: fetch unpublished returns error
    mock.ExpectQuery(regexp.QuoteMeta("SELECT id, topic, payload\n\t\tFROM outbox\n\t\tWHERE published_at IS NULL\n\t\tORDER BY id ASC\n\t\tLIMIT $1")).
        WithArgs(100).
        WillReturnError(errors.New("db down"))

    mr, qc, _ := newTestQueue(t)
    defer mr.Close()
    defer qc.Close()

    p := NewOutboxPublisher(db, qc)
    ctx, cancel := context.WithCancel(context.Background())

    go func() {
        // Allow one iteration to hit the fetch error then stop
        time.Sleep(150 * time.Millisecond)
        cancel()
    }()

    p.Start(ctx)

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

func TestOutboxPublisher_MarkPublishedError_Continues(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    payload := map[string]any{"id": "job-mark", "enqueued_at": time.Now().UTC(), "attempt": 1}
    pb, _ := json.Marshal(payload)

    // First loop: one row
    rows := sqlmock.NewRows([]string{"id", "topic", "payload"}).AddRow(int64(20), "jobs", pb)
    mock.ExpectQuery(regexp.QuoteMeta("SELECT id, topic, payload\n\t\tFROM outbox\n\t\tWHERE published_at IS NULL\n\t\tORDER BY id ASC\n\t\tLIMIT $1")).
        WithArgs(100).
        WillReturnRows(rows)

    // MarkPublished fails
    mock.ExpectExec(regexp.QuoteMeta("UPDATE outbox SET published_at = NOW() WHERE id = $1")).
        WithArgs(int64(20)).
        WillReturnError(errors.New("update fail"))

    // After mark failure, loop continues; expect metrics stats query in idle path
    mock.ExpectQuery(regexp.QuoteMeta("SELECT \n\t\t\tCOUNT(*) as count,\n\t\t\tCOALESCE(EXTRACT(EPOCH FROM (NOW() - MIN(created_at))), 0) as oldest_age_seconds\n\t\tFROM outbox \n\t\tWHERE published_at IS NULL")).
        WillReturnRows(sqlmock.NewRows([]string{"count", "oldest_age_seconds"}).AddRow(1, 0))

    mr, qc, _ := newTestQueue(t)
    defer mr.Close()
    defer qc.Close()

    p := NewOutboxPublisher(db, qc)
    ctx, cancel := context.WithCancel(context.Background())

    go func() {
        // Allow one iteration to fetch and attempt mark, then stop
        time.Sleep(150 * time.Millisecond)
        cancel()
    }()

    p.Start(ctx)

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}
