package queue

import (
    "context"
    "errors"
    "testing"
    "time"

    miniredis "github.com/alicebob/miniredis/v2"
    "github.com/redis/go-redis/v9"
)

func TestClient_Ping_Close_NilSafe(t *testing.T) {
    var c *Client
    if err := c.Ping(context.Background()); err != nil {
        t.Fatalf("nil client Ping should be nil err, got %v", err)
    }
    if err := c.Close(); err != nil {
        t.Fatalf("nil client Close should be nil err, got %v", err)
    }

    c = &Client{}
    if err := c.Ping(context.Background()); err != nil {
        t.Fatalf("nil redis Ping should be nil err, got %v", err)
    }
    if err := c.Close(); err != nil {
        t.Fatalf("nil redis Close should be nil err, got %v", err)
    }
}

func TestStartWorker_FallbackToSimpleAndProcessOne(t *testing.T) {
    // Force advanced queue creation to fail
    orig := newAdvancedQueue
    newAdvancedQueue = func(redisURL, queueName string) (advQueue, error) {
        return nil, assertErr("forced fail")
    }
    defer func() { newAdvancedQueue = orig }()

    mr := miniredis.RunT(t)

    // Build client pointing at miniredis
    rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
    c := &Client{redis: rdb}

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    processed := make(chan []byte, 1)
    go c.StartWorker(ctx, "test-queue", func(b []byte) error {
        processed <- b
        // stop after first
        cancel()
        return nil
    })

    // enqueue one message
    if err := rdb.LPush(context.Background(), "test-queue", []byte(`{"hello":"world"}`)).Err(); err != nil {
        t.Fatalf("LPush failed: %v", err)
    }

    select {
    case <-processed:
        // ok
    case <-time.After(2 * time.Second):
        t.Fatalf("timeout waiting for message to be processed")
    }
}

type assertErr string

func (e assertErr) Error() string { return string(e) }

func TestStartWorker_SimpleMode_HandlerError(t *testing.T) {
    // Force advanced queue creation to fail
    orig := newAdvancedQueue
    newAdvancedQueue = func(redisURL, queueName string) (advQueue, error) {
        return nil, assertErr("forced fail")
    }
    defer func() { newAdvancedQueue = orig }()

    mr := miniredis.RunT(t)

    rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
    c := &Client{redis: rdb}

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    called := make(chan struct{}, 1)
    go c.StartWorker(ctx, "test-queue-err", func(b []byte) error {
        // signal we were called and return an error
        select { case called <- struct{}{}: default: }
        // cancel soon after to stop loop
        cancel()
        return errors.New("handler failed")
    })

    if err := rdb.LPush(context.Background(), "test-queue-err", []byte(`{"x":1}`)).Err(); err != nil {
        t.Fatalf("LPush failed: %v", err)
    }

    select {
    case <-called:
        // ok, handler error path executed
    case <-time.After(2 * time.Second):
        t.Fatalf("timeout waiting for handler to be invoked")
    }
}

func TestStartWorker_SimpleMode_TransientBRPOPError(t *testing.T) {
    // Force advanced queue creation to fail
    orig := newAdvancedQueue
    newAdvancedQueue = func(redisURL, queueName string) (advQueue, error) {
        return nil, assertErr("forced fail")
    }
    defer func() { newAdvancedQueue = orig }()

    mr := miniredis.RunT(t)

    rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
    c := &Client{redis: rdb}

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    done := make(chan struct{})
    go func() {
        // handler shouldn't be called; just block until context cancel
        c.StartWorker(ctx, "test-queue-transient", func(b []byte) error { return nil })
        close(done)
    }()

    // Stop miniredis to induce a BRPOP connection error quickly
    mr.Close()

    // Give the worker a brief moment to encounter the error, then cancel
    time.Sleep(150 * time.Millisecond)
    cancel()

    select {
    case <-done:
        // exited cleanly after cancel
    case <-time.After(2 * time.Second):
        t.Fatalf("worker did not exit after cancel on transient error")
    }
}
