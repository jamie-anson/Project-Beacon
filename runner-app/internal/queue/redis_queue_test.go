package queue

import (
    "context"
    "encoding/json"
    "errors"
    "testing"
    "time"

    r8 "github.com/go-redis/redis/v8"
)

// fakeAdapter implements simpleAdapter for testing Fail() without a real Redis
type fakeAdapter struct {
    lpushCalls int
    zaddCalls  int
    delCalls   int
    zremCalls  int
    // injectable errors for specific commands
    lpushErr   error
    zaddErr    error
    delErr     error
    zremErr    error
    lastZ      *r8.Z
    lastDelKey string
}

func (f *fakeAdapter) LPush(ctx context.Context, key string, values ...interface{}) cmdErr {
    f.lpushCalls++
    if f.lpushErr != nil {
        return r8.NewStatusResult("", f.lpushErr)
    }
    return r8.NewStatusResult("OK", nil)
}

func (f *fakeAdapter) ZAdd(ctx context.Context, key string, members ...*r8.Z) cmdErr {
    f.zaddCalls++
    if len(members) > 0 {
        f.lastZ = members[0]
    }
    if f.zaddErr != nil {
        return r8.NewIntResult(0, f.zaddErr)
    }
    return r8.NewIntResult(1, nil)
}

func (f *fakeAdapter) Del(ctx context.Context, keys ...string) cmdErr {
    f.delCalls++
    if len(keys) > 0 {
        f.lastDelKey = keys[0]
    }
    if f.delErr != nil {
        return r8.NewIntResult(0, f.delErr)
    }
    return r8.NewIntResult(1, nil)
}

func (f *fakeAdapter) ZRem(ctx context.Context, key string, members ...interface{}) cmdErr {
    f.zremCalls++
    if f.zremErr != nil {
        return r8.NewIntResult(0, f.zremErr)
    }
    return r8.NewIntResult(1, nil)
}

func TestRedisQueue_Fail_SchedulesRetry(t *testing.T) {
    q := &RedisQueue{
        // client is not used because we route through testAdapter
        queueName:  "jobs",
        retryQueue: "jobs:retry",
        deadQueue:  "jobs:dead",
        maxRetries: 3,
        retryDelay: 2 * time.Second,
    }
    fa := &fakeAdapter{}
    q.WithTestAdapter(fa)

    msg := &JobMessage{ID: "id-1", JobSpecID: "js-1", Attempts: 1, MaxRetries: 3}

    err := q.Fail(context.Background(), msg, errors.New("boom"))
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if fa.delCalls == 0 {
        t.Fatalf("expected processing key deletion via Del")
    }
    if fa.zaddCalls != 1 {
        t.Fatalf("expected one ZAdd to retry queue, got %d", fa.zaddCalls)
    }
    if fa.lpushCalls != 0 {
        t.Fatalf("did not expect push to dead queue on retry path")
    }
    if fa.lastZ == nil {
        t.Fatalf("expected retry Z payload")
    }

    // Verify member is valid JSON JobMessage regardless of underlying type
    var out JobMessage
    switch v := fa.lastZ.Member.(type) {
    case string:
        if err := json.Unmarshal([]byte(v), &out); err != nil {
            t.Fatalf("invalid retry member json (string): %v", err)
        }
    case []byte:
        if err := json.Unmarshal(v, &out); err != nil {
            t.Fatalf("invalid retry member json (bytes): %v", err)
        }
    default:
        t.Fatalf("unexpected Z.Member type %T", v)
    }
    if out.ID != msg.ID {
        t.Fatalf("expected same job id in retry member")
    }
}

func TestRedisQueue_Fail_DeadLettersAfterMaxRetries(t *testing.T) {
    q := &RedisQueue{
        queueName:  "jobs",
        retryQueue: "jobs:retry",
        deadQueue:  "jobs:dead",
        maxRetries: 2,
        retryDelay: 1 * time.Second,
    }
    fa := &fakeAdapter{}
    q.WithTestAdapter(fa)

    // Attempts already at MaxRetries => should go to dead
    msg := &JobMessage{ID: "id-2", JobSpecID: "js-2", Attempts: 2, MaxRetries: 2}

    err := q.Fail(context.Background(), msg, errors.New("boom"))
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if fa.delCalls == 0 {
        t.Fatalf("expected processing key deletion via Del")
    }
    if fa.zaddCalls != 0 {
        t.Fatalf("did not expect ZAdd on dead-letter path")
    }
    if fa.lpushCalls != 1 {
        t.Fatalf("expected one LPush to dead queue, got %d", fa.lpushCalls)
    }
}
