package queue

import (
	"context"
	"time"
)

// JobMessage represents a job in the queue
type JobMessage struct {
	ID          string                 `json:"id"`
	JobSpecID   string                 `json:"jobspec_id"`
	Action      string                 `json:"action"`
	Payload     map[string]interface{} `json:"payload"`
	Attempts    int                    `json:"attempts"`
	MaxRetries  int                    `json:"max_retries"`
	EnqueuedAt  time.Time              `json:"enqueued_at"`
	LastAttempt time.Time              `json:"last_attempt,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// Queue defines the interface for queue operations
type Queue interface {
	Enqueue(ctx context.Context, jobSpecID string, action string, payload map[string]interface{}) error
	Dequeue(ctx context.Context) (*JobMessage, error)
	Complete(ctx context.Context, message *JobMessage) error
	Fail(ctx context.Context, message *JobMessage, jobError error) error
	GetQueueStats(ctx context.Context) (map[string]int64, error)
	RecoverStaleJobs(ctx context.Context) error
	Close() error
}

// CircuitBreakerStats defines the interface for circuit breaker statistics
type CircuitBreakerStats interface {
	GetCircuitBreakerStats() string
	LogCircuitBreakerStats()
}

// cmdErr is the minimal command interface we use (Err accessor)
type cmdErr interface{ Err() error }

// simpleAdapter is a tiny seam used for testing Fail() without a real Redis.
// It intentionally only includes methods used by Fail().
type simpleAdapter interface {
	LPush(ctx context.Context, key string, values ...interface{}) cmdErr
	ZAdd(ctx context.Context, key string, members ...interface{}) cmdErr
	ZRem(ctx context.Context, key string, members ...interface{}) cmdErr
	Del(ctx context.Context, keys ...string) cmdErr
}
