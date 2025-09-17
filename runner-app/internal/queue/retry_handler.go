package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// RetryHandler manages job retry logic and dead letter queue operations
type RetryHandler struct {
	queueName     string
	retryQueue    string
	deadQueue     string
	maxRetries    int
	retryDelay    time.Duration
	client        *redis.Client
	circuitClient *RedisCircuitBreaker
	testAdapter   simpleAdapter
}

// NewRetryHandler creates a new RetryHandler
func NewRetryHandler(queueName string, client *redis.Client, circuitClient *RedisCircuitBreaker) *RetryHandler {
	return &RetryHandler{
		queueName:     queueName,
		retryQueue:    queueName + ":retry",
		deadQueue:     queueName + ":dead",
		maxRetries:    3,
		retryDelay:    time.Minute,
		client:        client,
		circuitClient: circuitClient,
	}
}

// WithTestAdapter sets a testing adapter seam for unit tests
func (r *RetryHandler) WithTestAdapter(adapter simpleAdapter) *RetryHandler {
	r.testAdapter = adapter
	return r
}

// helpers to call either adapter or circuit breaker client
func (r *RetryHandler) lpush(ctx context.Context, key string, values ...interface{}) cmdErr {
	if r.testAdapter != nil {
		return r.testAdapter.LPush(ctx, key, values...)
	}
	return r.circuitClient.LPush(ctx, key, values...)
}

func (r *RetryHandler) zadd(ctx context.Context, key string, members ...*redis.Z) cmdErr {
	if r.testAdapter != nil {
		// Convert *redis.Z to interface{} for test adapter
		interfaceMembers := make([]interface{}, len(members))
		for i, member := range members {
			interfaceMembers[i] = member
		}
		return r.testAdapter.ZAdd(ctx, key, interfaceMembers...)
	}
	return r.circuitClient.ZAdd(ctx, key, members...)
}

func (r *RetryHandler) zrem(ctx context.Context, key string, members ...interface{}) cmdErr {
	if r.testAdapter != nil {
		return r.testAdapter.ZRem(ctx, key, members...)
	}
	return r.circuitClient.ZRem(ctx, key, members...)
}


// DequeueRetry attempts to dequeue from the retry queue
func (r *RetryHandler) DequeueRetry(ctx context.Context) (*JobMessage, error) {
	tracer := otel.Tracer("runner/queue/retry")
	ctx, span := tracer.Start(ctx, "RetryHandler.DequeueRetry", trace.WithAttributes(
		attribute.String("queue.retry", r.retryQueue),
	))
	defer span.End()

	// Check for jobs ready to retry
	now := time.Now()
	
	// Get jobs from retry queue that are ready to be retried
	results, err := r.circuitClient.ZRangeByScore(ctx, r.retryQueue, &redis.ZRangeBy{
		Min:   "0",
		Max:   fmt.Sprintf("%d", now.Unix()),
		Count: 1,
	}).Result()

	if err != nil || len(results) == 0 {
		if err != nil {
			span.RecordError(err)
		}
		return nil, nil // No jobs ready for retry
	}

	// Remove from retry queue
	if err := r.zrem(ctx, r.retryQueue, results[0]).Err(); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to remove job from retry queue: %w", err)
	}

	var message JobMessage
	if err := json.Unmarshal([]byte(results[0]), &message); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to unmarshal retry job message: %w", err)
	}

	message.Attempts++
	message.LastAttempt = time.Now()

	span.SetAttributes(
		attribute.String("job.id", message.ID),
		attribute.String("jobspec.id", message.JobSpecID),
		attribute.Int("job.attempts", message.Attempts),
	)
	return &message, nil
}

// HandleFailure handles job failure with retry logic
func (r *RetryHandler) HandleFailure(ctx context.Context, message *JobMessage, jobError error) error {
	tracer := otel.Tracer("runner/queue/retry")
	ctx, span := tracer.Start(ctx, "RetryHandler.HandleFailure", trace.WithAttributes(
		attribute.String("queue.name", r.queueName),
		attribute.String("queue.retry", r.retryQueue),
		attribute.String("queue.dead", r.deadQueue),
		attribute.String("job.id", message.ID),
		attribute.String("jobspec.id", message.JobSpecID),
		attribute.Int("job.attempts", message.Attempts),
		attribute.Int("job.max_retries", message.MaxRetries),
	))
	defer span.End()

	message.Error = jobError.Error()
	span.RecordError(jobError)

	// Check if we should retry
	if message.Attempts < message.MaxRetries {
		// Calculate retry delay with exponential backoff
		retryDelay := r.retryDelay * time.Duration(message.Attempts)
		retryTime := time.Now().Add(retryDelay)

		messageJSON, err := json.Marshal(message)
		if err != nil {
			span.RecordError(err)
			return fmt.Errorf("failed to marshal retry message: %w", err)
		}

		// Add to retry queue with score as retry timestamp
		if err := r.zadd(ctx, r.retryQueue, &redis.Z{
			Score:  float64(retryTime.Unix()),
			Member: messageJSON,
		}).Err(); err != nil {
			span.RecordError(err)
			return fmt.Errorf("failed to add job to retry queue: %w", err)
		}

		log.Printf("Job %s failed (attempt %d/%d), scheduled for retry at %v: %v",
			message.ID, message.Attempts, message.MaxRetries, retryTime, jobError)
		span.SetAttributes(
			attribute.Bool("job.retry_scheduled", true),
			attribute.Int64("job.retry_at_unix", retryTime.Unix()),
			attribute.Int64("job.retry_backoff_ms", retryDelay.Milliseconds()),
		)
		return nil
	}

	// Max retries exceeded, move to dead letter queue
	return r.moveToDeadQueue(ctx, message, span)
}

// moveToDeadQueue moves a job to the dead letter queue
func (r *RetryHandler) moveToDeadQueue(ctx context.Context, message *JobMessage, span trace.Span) error {
	messageJSON, err := json.Marshal(message)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to marshal dead message: %w", err)
	}

	if err := r.lpush(ctx, r.deadQueue, messageJSON).Err(); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to add job to dead queue: %w", err)
	}

	log.Printf("Job %s moved to dead queue after %d attempts: %s",
		message.ID, message.Attempts, message.Error)
	span.SetAttributes(attribute.Bool("job.dead_letter", true))
	return nil
}
