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

func newMiniredisQueue(t *testing.T, queueName string) (*RedisQueue, *miniredis.Miniredis) {
    t.Helper()
    mr := miniredis.RunT(t)
    client := r8.NewClient(&r8.Options{Addr: mr.Addr()})
    q := &RedisQueue{
        client:            client,
        queueName:         queueName,
        retryQueue:        queueName + ":retry",
        deadQueue:         queueName + ":dead",
        maxRetries:        3,
        retryDelay:        1,
        visibilityTimeout: 0,
    }
    return q, mr
}

func TestDequeue_Retry_ZRemError(t *testing.T) {
    q, mr := newMiniredisQueue(t, "jobs")
    defer q.Close()
    defer mr.Close()

    ctx := context.Background()
    // Put a ready retry entry
    msg := JobMessage{ID: "rz1", JobSpecID: "s9", Attempts: 0, MaxRetries: 3}
    b, _ := json.Marshal(msg)
    if err := q.client.ZAdd(ctx, q.retryQueue, &r8.Z{Score: float64(time.Now().Unix()), Member: string(b)}).Err(); err != nil { t.Fatal(err) }

    // Inject adapter that fails ZRem
    fa := &fakeAdapter{zremErr: errors.New("zrem fail")}
    q.WithTestAdapter(fa)

    got, err := q.Dequeue(ctx)
    if err == nil { t.Fatalf("expected error due to ZRem failure, got message=%v", got) }
}

func TestEnqueue_PushesToMainQueue(t *testing.T) {
    q, mr := newMiniredisQueue(t, "jobs")
    defer q.Close()
    defer mr.Close()

    ctx := context.Background()
    if err := q.Enqueue(ctx, "spec-123", "run", map[string]interface{}{"k":"v"}); err != nil {
        t.Fatalf("enqueue err: %v", err)
    }
    // main queue should have 1 item
    if n, _ := q.client.LLen(ctx, q.queueName).Result(); n != 1 {
        t.Fatalf("want main len=1 got %d", n)
    }
    // Pop and verify JSON minimally parses and contains jobspec_id
    res, err := q.client.BRPop(ctx, time.Second, q.queueName).Result()
    if err != nil { t.Fatalf("brpop err: %v", err) }
    var msg JobMessage
    if err := json.Unmarshal([]byte(res[1]), &msg); err != nil { t.Fatalf("unmarshal: %v", err) }
    if msg.JobSpecID != "spec-123" { t.Fatalf("want jobspec_id=spec-123 got %s", msg.JobSpecID) }
}

func TestComplete_RemovesProcessingKey(t *testing.T) {
    q, mr := newMiniredisQueue(t, "jobs")
    defer q.Close()
    defer mr.Close()

    ctx := context.Background()
    // Seed a processing key and then call Complete
    m := &JobMessage{ID: "done1", JobSpecID: "s1", Attempts: 1, MaxRetries: 3}
    if err := q.client.Set(ctx, "jobs:processing:"+m.ID, "payload", 0).Err(); err != nil { t.Fatal(err) }
    if err := q.Complete(ctx, m); err != nil { t.Fatalf("complete err: %v", err) }
    if exists, _ := q.client.Exists(ctx, "jobs:processing:"+m.ID).Result(); exists != 0 {
        t.Fatalf("processing key should be removed, still exists=%d", exists)
    }
}
 

func TestDequeue_EmptyQueues_ReturnsNil(t *testing.T) {
    q, mr := newMiniredisQueue(t, "jobs")
    defer q.Close()
    defer mr.Close()

    ctx := context.Background()
    // Ensure both main list and retry zset are empty
    if n, _ := q.client.LLen(ctx, q.queueName).Result(); n != 0 { t.Fatalf("expected empty main queue") }
    if n, _ := q.client.ZCard(ctx, q.retryQueue).Result(); n != 0 { t.Fatalf("expected empty retry queue") }

    got, err := q.Dequeue(ctx)
    if err != nil { t.Fatalf("unexpected err: %v", err) }
    if got != nil { t.Fatalf("expected nil message when queues empty") }
}
 

func TestGetQueueStats_WithItems(t *testing.T) {
    q, mr := newMiniredisQueue(t, "jobs")
    defer q.Close()
    defer mr.Close()

    ctx := context.Background()

    // Populate queues
    // main list has 2 items
    if err := q.client.LPush(ctx, q.queueName, "a", "b").Err(); err != nil { t.Fatal(err) }
    // retry zset has 1 item
    if err := q.client.ZAdd(ctx, q.retryQueue, &r8.Z{Score: 1, Member: "x"}).Err(); err != nil { t.Fatal(err) }
    // dead list has 3 items
    if err := q.client.LPush(ctx, q.deadQueue, "d1", "d2", "d3").Err(); err != nil { t.Fatal(err) }
    // processing keys (2 keys)
    if err := q.client.Set(ctx, "jobs:processing:p1", "v1", 0).Err(); err != nil { t.Fatal(err) }
    if err := q.client.Set(ctx, "jobs:processing:p2", "v2", 0).Err(); err != nil { t.Fatal(err) }

    stats, err := q.GetQueueStats(ctx)
    if err != nil { t.Fatalf("unexpected err: %v", err) }

    if stats["main"] != 2 { t.Fatalf("want main=2 got %d", stats["main"]) }
    if stats["retry"] != 1 { t.Fatalf("want retry=1 got %d", stats["retry"]) }
    if stats["dead"] != 3 { t.Fatalf("want dead=3 got %d", stats["dead"]) }
    if stats["processing"] != 2 { t.Fatalf("want processing=2 got %d", stats["processing"]) }
}

func TestRecoverStaleJobs_RetryAndDeadletter(t *testing.T) {
    q, mr := newMiniredisQueue(t, "jobs")
    defer q.Close()
    defer mr.Close()

    ctx := context.Background()

    // Prepare two processing keys with no TTL (TTL=-1), which code treats as stale (<=0)
    retryMsg := JobMessage{ID: "j1", JobSpecID: "s1", Attempts: 0, MaxRetries: 1}
    deadMsg := JobMessage{ID: "j2", JobSpecID: "s2", Attempts: 2, MaxRetries: 2}

    rb, _ := json.Marshal(retryMsg)
    db, _ := json.Marshal(deadMsg)

    if err := q.client.Set(ctx, "jobs:processing:"+retryMsg.ID, rb, 0).Err(); err != nil { t.Fatal(err) }
    if err := q.client.Set(ctx, "jobs:processing:"+deadMsg.ID, db, 0).Err(); err != nil { t.Fatal(err) }

    // Attach fake adapter to observe Fail() behavior (ZAdd for retry, LPush for dead)
    fa := &fakeAdapter{}
    q.WithTestAdapter(fa)

    if err := q.RecoverStaleJobs(ctx); err != nil {
        t.Fatalf("unexpected err: %v", err)
    }

    if fa.zaddCalls < 1 { t.Fatalf("expected at least one retry ZAdd, got %d", fa.zaddCalls) }
    if fa.lpushCalls < 1 { t.Fatalf("expected at least one dead LPush, got %d", fa.lpushCalls) }
}

func TestDequeue_FromMainQueue_HappyPath(t *testing.T) {
    q, mr := newMiniredisQueue(t, "jobs")
    // ensure non-zero visibility timeout so processing key TTL is set
    q.visibilityTimeout = 60_000_000_000 // 60s in ns
    defer q.Close()
    defer mr.Close()

    ctx := context.Background()
    msg := JobMessage{ID: "m1", JobSpecID: "s1", Attempts: 0, MaxRetries: 3}
    b, _ := json.Marshal(msg)
    if err := q.client.LPush(ctx, q.queueName, string(b)).Err(); err != nil { t.Fatal(err) }

    got, err := q.Dequeue(ctx)
    if err != nil { t.Fatalf("unexpected err: %v", err) }
    if got.ID != msg.ID { t.Fatalf("want id %s got %s", msg.ID, got.ID) }
    if got.Attempts != 1 { t.Fatalf("want attempts=1 got %d", got.Attempts) }

    // processing key should exist with TTL > 0
    pkey := "jobs:processing:" + got.ID
    ttl, err := q.client.TTL(ctx, pkey).Result()
    if err != nil { t.Fatalf("ttl err: %v", err) }
    if ttl <= 0 { t.Fatalf("expected positive TTL, got %v", ttl) }
}

func TestDequeue_FallsBackToRetryQueue(t *testing.T) {
    q, mr := newMiniredisQueue(t, "jobs")
    q.visibilityTimeout = 30_000_000_000 // 30s
    defer q.Close()
    defer mr.Close()

    ctx := context.Background()
    msg := JobMessage{ID: "r1", JobSpecID: "s2", Attempts: 0, MaxRetries: 3}
    b, _ := json.Marshal(msg)
    // put into retry set with score <= now
    if err := q.client.ZAdd(ctx, q.retryQueue, &r8.Z{Score: float64(time.Now().Unix()), Member: string(b)}).Err(); err != nil { t.Fatal(err) }

    got, err := q.Dequeue(ctx)
    if err != nil { t.Fatalf("unexpected err: %v", err) }
    if got.ID != msg.ID { t.Fatalf("want id %s got %s", msg.ID, got.ID) }
    if got.Attempts != 1 { t.Fatalf("want attempts=1 got %d", got.Attempts) }

    // ensure it was removed from retry set
    if n, _ := q.client.ZCard(ctx, q.retryQueue).Result(); n != 0 {
        t.Fatalf("expected retry set empty, got %d", n)
    }
}

func TestDequeue_InvalidJSONInMain(t *testing.T) {
    q, mr := newMiniredisQueue(t, "jobs")
    defer q.Close()
    defer mr.Close()

    ctx := context.Background()
    if err := q.client.LPush(ctx, q.queueName, "{not-json}").Err(); err != nil { t.Fatal(err) }
    if _, err := q.Dequeue(ctx); err == nil {
        t.Fatalf("expected error for invalid JSON in main queue")
    }
}

func TestDequeue_InvalidJSONInRetry(t *testing.T) {
    q, mr := newMiniredisQueue(t, "jobs")
    defer q.Close()
    defer mr.Close()

    ctx := context.Background()
    // ensure main is empty and retry has invalid JSON ready now
    if err := q.client.ZAdd(ctx, q.retryQueue, &r8.Z{Score: float64(time.Now().Unix()), Member: "{bad-json}"}).Err(); err != nil { t.Fatal(err) }
    if _, err := q.Dequeue(ctx); err == nil {
        t.Fatalf("expected error for invalid JSON in retry queue")
    }
}
