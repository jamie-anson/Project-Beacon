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
    miniredis "github.com/alicebob/miniredis/v2"
    "github.com/jamie-anson/project-beacon-runner/internal/queue"
    "github.com/redis/go-redis/v9"
)

// helper: create a queue client backed by miniredis and return its URL and client
func newTestQueue(t *testing.T) (*miniredis.Miniredis, *queue.Client, string) {
    t.Helper()
    mr, err := miniredis.Run()
    if err != nil {
        t.Fatalf("failed to start miniredis: %v", err)
    }
    url := "redis://" + mr.Addr()
    t.Setenv("REDIS_URL", url)
    qc, err := queue.NewFromEnv()
    if err != nil {
        t.Fatalf("failed to create queue client: %v", err)
    }
    return mr, qc, url
}

func TestOutboxPublisher_PublishesAndMarks(t *testing.T) {
    // Arrange DB
    db, mock, _ := sqlmock.New()
    defer db.Close()

    // One unpublished outbox row
    payload := map[string]any{"id": "job-xyz", "enqueued_at": time.Now().UTC(), "attempt": 1}
    pb, _ := json.Marshal(payload)
    rows := sqlmock.NewRows([]string{"id", "topic", "payload"}).AddRow(int64(1), "jobs", pb)
    mock.ExpectQuery(regexp.QuoteMeta("SELECT id, topic, payload\n\t\tFROM outbox\n\t\tWHERE published_at IS NULL\n\t\tORDER BY id ASC\n\t\tLIMIT $1")).
        WithArgs(100).
        WillReturnRows(rows)

    // Expect mark published
    mock.ExpectExec(regexp.QuoteMeta("UPDATE outbox SET published_at = NOW() WHERE id = $1")).
        WithArgs(int64(1)).
        WillReturnResult(sqlmock.NewResult(0, 1))

    // Arrange Redis
    mr, qc, url := newTestQueue(t)
    defer mr.Close()
    defer qc.Close()

    // Act: start publisher
    p := NewOutboxPublisher(db, qc)
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    done := make(chan struct{})
    go func() {
        p.Start(ctx)
        close(done)
    }()

    // Wait until job appears on the list, then cancel
    rdb := redisClientForURL(url)
    defer rdb.Close()

    deadline := time.Now().Add(2 * time.Second)
    for {
        if time.Now().After(deadline) {
            t.Fatal("timeout waiting for enqueued job")
        }
        llen := rdb.LLen(ctx, "jobs").Val()
        if llen >= 1 {
            break
        }
        time.Sleep(10 * time.Millisecond)
    }
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

func TestOutboxPublisher_InvalidJSON_SkipsAndDoesNotMark(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    // Return one row with invalid JSON payload
    rows := sqlmock.NewRows([]string{"id", "topic", "payload"}).AddRow(int64(2), "jobs", []byte("not-json"))
    mock.ExpectQuery(regexp.QuoteMeta("SELECT id, topic, payload\n\t\tFROM outbox\n\t\tWHERE published_at IS NULL\n\t\tORDER BY id ASC\n\t\tLIMIT $1")).
        WithArgs(100).
        WillReturnRows(rows)

    // Next loop: no rows so it will call metrics update
    mock.ExpectQuery(regexp.QuoteMeta("SELECT \n\t\t\tCOUNT(*) as count,\n\t\t\tCOALESCE(EXTRACT(EPOCH FROM (NOW() - MIN(created_at))), 0) as oldest_age_seconds\n\t\tFROM outbox \n\t\tWHERE published_at IS NULL")).
        WillReturnRows(sqlmock.NewRows([]string{"count", "oldest_age_seconds"}).AddRow(0, 0))

    mr, qc, _ := newTestQueue(t)
    defer mr.Close()
    defer qc.Close()

    p := NewOutboxPublisher(db, qc)
    ctx, cancel := context.WithCancel(context.Background())

    // Stop after short delay
    go func() {
        time.Sleep(200 * time.Millisecond)
        cancel()
    }()
    p.Start(ctx)

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

func TestOutboxPublisher_MissingID_SkipsAndDoesNotMark(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    // Payload missing jobspec id field
    pb, _ := json.Marshal(map[string]any{"foo": "bar"})
    rows := sqlmock.NewRows([]string{"id", "topic", "payload"}).AddRow(int64(3), "jobs", pb)
    mock.ExpectQuery(regexp.QuoteMeta("SELECT id, topic, payload\n\t\tFROM outbox\n\t\tWHERE published_at IS NULL\n\t\tORDER BY id ASC\n\t\tLIMIT $1")).
        WithArgs(100).
        WillReturnRows(rows)

    // Next loop: no rows -> metrics
    mock.ExpectQuery(regexp.QuoteMeta("SELECT \n\t\t\tCOUNT(*) as count,\n\t\t\tCOALESCE(EXTRACT(EPOCH FROM (NOW() - MIN(created_at))), 0) as oldest_age_seconds\n\t\tFROM outbox \n\t\tWHERE published_at IS NULL")).
        WillReturnRows(sqlmock.NewRows([]string{"count", "oldest_age_seconds"}).AddRow(0, 0))

    mr, qc, _ := newTestQueue(t)
    defer mr.Close()
    defer qc.Close()

    p := NewOutboxPublisher(db, qc)
    ctx, cancel := context.WithCancel(context.Background())

    go func() {
        time.Sleep(200 * time.Millisecond)
        cancel()
    }()
    p.Start(ctx)

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

// redisClientForURL returns a go-redis client for assertions
func redisClientForURL(url string) *redis.Client {
    opt, _ := redis.ParseURL(url)
    return redis.NewClient(opt)
}
