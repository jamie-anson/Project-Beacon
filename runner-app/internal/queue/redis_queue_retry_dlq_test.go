package queue

import (
    "context"
    "encoding/json"
    "errors"
    "testing"
    "time"

    miniredis "github.com/alicebob/miniredis/v2"
    r8 "github.com/go-redis/redis/v8"
)

// helper mirrors newMiniredisQueue but lets us tweak retries easily
func newFastRetryQueue(t *testing.T, queueName string, maxRetries int, retryDelay time.Duration) (*RedisQueue, *miniredis.Miniredis) {
    t.Helper()
    mr := miniredis.RunT(t)
    client := r8.NewClient(&r8.Options{Addr: mr.Addr()})
    q := &RedisQueue{
        client:            client,
        queueName:         queueName,
        retryQueue:        queueName + ":retry",
        deadQueue:         queueName + ":dead",
        maxRetries:        maxRetries,
        retryDelay:        retryDelay,
        visibilityTimeout: time.Second,
    }
    return q, mr
}

func TestRetryAndDeadLetterFlow(t *testing.T) {
    // Very small backoff so retries become ready immediately
    q, mr := newFastRetryQueue(t, "jobs", 3, time.Nanosecond)
    defer q.Close()
    defer mr.Close()

    ctx := context.Background()

    // Enqueue one job
    if err := q.Enqueue(ctx, "spec-retry", "execute", map[string]any{"k": "v"}); err != nil {
        t.Fatalf("enqueue: %v", err)
    }

    // 1) Dequeue from main and simulate worker failure
    msg1, err := q.Dequeue(ctx)
    if err != nil || msg1 == nil { t.Fatalf("dequeue1 err=%v msg=%v", err, msg1) }
    if msg1.Attempts != 1 { t.Fatalf("want attempts=1 got %d", msg1.Attempts) }
    if err := q.Fail(ctx, msg1, errors.New("boom-1")); err != nil {
        t.Fatalf("fail1: %v", err)
    }

    // Should be in retry set, ready now (give it a moment for sorted set operations)
    time.Sleep(10 * time.Millisecond)
    
    // 2) Dequeue from retry and fail again
    msg2, err := q.Dequeue(ctx)
    if err != nil || msg2 == nil { t.Fatalf("dequeue2 err=%v msg=%v", err, msg2) }
    if msg2.Attempts != 2 { t.Fatalf("want attempts=2 got %d", msg2.Attempts) }
    if err := q.Fail(ctx, msg2, errors.New("boom-2")); err != nil {
        t.Fatalf("fail2: %v", err)
    }

    // Give it a moment for retry scheduling
    time.Sleep(10 * time.Millisecond)
    
    // 3) Dequeue from retry a second time and fail -> should dead-letter
    msg3, err := q.Dequeue(ctx)
    if err != nil || msg3 == nil { t.Fatalf("dequeue3 err=%v msg=%v", err, msg3) }
    if msg3.Attempts != 3 { t.Fatalf("want attempts=3 got %d", msg3.Attempts) }
    if err := q.Fail(ctx, msg3, errors.New("boom-3")); err != nil {
        t.Fatalf("fail3: %v", err)
    }

    // Assert retry set empty and one item in dead queue
    if n, _ := q.client.ZCard(ctx, q.retryQueue).Result(); n != 0 {
        t.Fatalf("expected retry empty, got %d", n)
    }
    if deadN, _ := q.client.LLen(ctx, q.deadQueue).Result(); deadN != 1 {
        t.Fatalf("expected dead=1 got %d", deadN)
    }

    // Validate dead-letter content minimally
    raw, err := q.client.BRPop(ctx, time.Second, q.deadQueue).Result()
    if err != nil { t.Fatalf("read dead: %v", err) }
    var deadMsg JobMessage
    if err := json.Unmarshal([]byte(raw[1]), &deadMsg); err != nil {
        t.Fatalf("unmarshal dead: %v", err)
    }
    if deadMsg.ID == "" || deadMsg.JobSpecID != "spec-retry" || deadMsg.Error == "" {
        t.Fatalf("unexpected dead message: %+v", deadMsg)
    }

    // Ensure processing key removed
    if exists, _ := q.client.Exists(ctx, q.queueName+":processing:"+deadMsg.ID).Result(); exists != 0 {
        t.Fatalf("processing key should be removed for %s", deadMsg.ID)
    }
}
