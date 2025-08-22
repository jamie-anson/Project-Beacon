package queue

import (
    "context"
    "errors"
    "sync/atomic"
    "testing"
    "time"
)

type fakeAdvQueue struct{
    msg *JobMessage
    deqErr error
    failCount int32
    completeCount int32
    closeCount int32
    recoverCount int32
    dequeues int32
}

func (f *fakeAdvQueue) Dequeue(ctx context.Context) (*JobMessage, error) {
    atomic.AddInt32(&f.dequeues, 1)
    return f.msg, f.deqErr
}
func (f *fakeAdvQueue) Fail(ctx context.Context, message *JobMessage, jobError error) error {
    atomic.AddInt32(&f.failCount, 1)
    return nil
}
func (f *fakeAdvQueue) Complete(ctx context.Context, message *JobMessage) error {
    atomic.AddInt32(&f.completeCount, 1)
    return nil
}
func (f *fakeAdvQueue) RecoverStaleJobs(ctx context.Context) error {
    atomic.AddInt32(&f.recoverCount, 1)
    return nil
}
func (f *fakeAdvQueue) Close() error {
    atomic.AddInt32(&f.closeCount, 1)
    return nil
}

// simple handler toggles based on payload content
func handlerReturning(err error) func([]byte) error {
    return func([]byte) error { return err }
}

func TestStartWorker_HandlerSuccess_CallsComplete(t *testing.T) {
    // Override seam
    orig := newAdvancedQueue
    defer func(){ newAdvancedQueue = orig }()

    fake := &fakeAdvQueue{ msg: &JobMessage{ID:"m1", JobSpecID:"job-1"} }
    newAdvancedQueue = func(_, _ string) (advQueue, error) { return fake, nil }

    c := &Client{}
    ctx, cancel := context.WithCancel(context.Background())
    // cancel after first loop iteration
    go func(){ time.Sleep(50*time.Millisecond); cancel() }()

    c.StartWorker(ctx, "jobs", handlerReturning(nil))

    if got := atomic.LoadInt32(&fake.completeCount); got < 1 {
        t.Fatalf("expected Complete to be called, got %d", got)
    }
}

func TestStartWorker_HandlerError_CallsFail(t *testing.T) {
    orig := newAdvancedQueue
    defer func(){ newAdvancedQueue = orig }()

    fake := &fakeAdvQueue{ msg: &JobMessage{ID:"m2", JobSpecID:"job-2"} }
    newAdvancedQueue = func(_, _ string) (advQueue, error) { return fake, nil }

    c := &Client{}
    ctx, cancel := context.WithCancel(context.Background())
    go func(){ time.Sleep(50*time.Millisecond); cancel() }()

    c.StartWorker(ctx, "jobs", handlerReturning(errors.New("boom")))

    if got := atomic.LoadInt32(&fake.failCount); got < 1 {
        t.Fatalf("expected Fail to be called, got %d", got)
    }
}

func TestStartWorker_DequeueError_ContinuesLoop(t *testing.T) {
    orig := newAdvancedQueue
    defer func(){ newAdvancedQueue = orig }()

    fake := &fakeAdvQueue{ deqErr: errors.New("deq err") }
    newAdvancedQueue = func(_, _ string) (advQueue, error) { return fake, nil }

    c := &Client{}
    ctx, cancel := context.WithCancel(context.Background())
    // allow enough time for multiple dequeue attempts (500ms backoff)
    go func(){ time.Sleep(1100*time.Millisecond); cancel() }()

    c.StartWorker(ctx, "jobs", handlerReturning(nil))

    if got := atomic.LoadInt32(&fake.dequeues); got < 2 {
        t.Fatalf("expected >=2 dequeue attempts, got %d", got)
    }
}
