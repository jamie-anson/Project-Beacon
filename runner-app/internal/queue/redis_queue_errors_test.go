package queue

import (
    "context"
    "errors"
    "testing"
    "time"

    miniredis "github.com/alicebob/miniredis/v2"
    "github.com/go-redis/redis/v8"
)

func newQueueForTest(t *testing.T) (*RedisQueue, *miniredis.Miniredis) {
    t.Helper()
    mr, err := miniredis.Run()
    if err != nil { t.Fatalf("miniredis: %v", err) }
    client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
    return &RedisQueue{
        client:            client,
        queueName:         "jobs",
        retryQueue:        "jobs:retry",
        deadQueue:         "jobs:dead",
        maxRetries:        3,
        retryDelay:        time.Millisecond,
        visibilityTimeout: time.Second,
    }, mr
}

func TestFail_Retry_ZAddError(t *testing.T) {
    q := &RedisQueue{queueName: "jobs", retryQueue: "jobs:retry", deadQueue: "jobs:dead", retryDelay: time.Millisecond}
    q.WithTestAdapter(&fakeAdapter{zaddErr: errors.New("boom")})

    msg := &JobMessage{ID: "m1", JobSpecID: "j1", Attempts: 1, MaxRetries: 3}
    err := q.Fail(context.Background(), msg, errors.New("proc err"))
    if err == nil || err.Error() == "" || err.Error()[:23] != "failed to add job to retry"[:23] {
        t.Fatalf("expected retry zadd error, got: %v", err)
    }
}

func TestFail_Dead_LPushError(t *testing.T) {
    q := &RedisQueue{queueName: "jobs", retryQueue: "jobs:retry", deadQueue: "jobs:dead"}
    q.WithTestAdapter(&fakeAdapter{lpushErr: errors.New("no push")})

    msg := &JobMessage{ID: "m2", JobSpecID: "j2", Attempts: 3, MaxRetries: 3}
    err := q.Fail(context.Background(), msg, errors.New("final err"))
    if err == nil || err.Error() == "" || err.Error()[:23] != "failed to add job to dead"[:23] {
        t.Fatalf("expected dead-letter lpush error, got: %v", err)
    }
}

func TestComplete_DelErrorIsLoggedButNoError(t *testing.T) {
    q, mr := newQueueForTest(t)
    // Close server to force DEL errors
    mr.Close()

    msg := &JobMessage{ID: "m3", JobSpecID: "j3"}
    if err := q.Complete(context.Background(), msg); err != nil {
        t.Fatalf("Complete should not error even if DEL fails; got %v", err)
    }
}

func TestGetQueueStats_PipelineExecError(t *testing.T) {
    q, mr := newQueueForTest(t)
    // Ensure some keys exist then close to force Exec error
    // Push one item so lengths would be >0 if reachable
    if err := q.client.LPush(context.Background(), q.queueName, "x").Err(); err != nil {
        t.Fatalf("pre-push: %v", err)
    }
    mr.Close()

    if _, err := q.GetQueueStats(context.Background()); err == nil {
        t.Fatalf("expected error from GetQueueStats when pipeline exec fails")
    }
}
