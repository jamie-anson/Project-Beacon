package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	// OpenTelemetry
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// cmdErr is the minimal command interface we use (Err accessor)
type cmdErr interface{ Err() error }

// simpleAdapter is a tiny seam used for testing Fail() without a real Redis.
// It intentionally only includes methods used by Fail().
type simpleAdapter interface {
	LPush(ctx context.Context, key string, values ...interface{}) cmdErr
	ZAdd(ctx context.Context, key string, members ...*redis.Z) cmdErr
	ZRem(ctx context.Context, key string, members ...interface{}) cmdErr
	Del(ctx context.Context, keys ...string) cmdErr
}

// RedisQueue provides Redis-based job queuing with retry logic
type RedisQueue struct {
	client        *redis.Client
	circuitClient *RedisCircuitBreaker
	testAdapter   simpleAdapter
	queueName     string
	retryQueue    string
	deadQueue     string
	maxRetries    int
	retryDelay    time.Duration
	visibilityTimeout time.Duration
}

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

// NewRedisQueue creates a new Redis queue instance
func NewRedisQueue(redisURL string, queueName string) (*RedisQueue, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opt)
	
	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Create circuit breaker wrapper
	circuitClient := NewRedisCircuitBreaker(client, "redis-queue-"+queueName)

	return &RedisQueue{
		client:            client,
		circuitClient:     circuitClient,
		queueName:         queueName,
		retryQueue:        queueName + ":retry",
		deadQueue:         queueName + ":dead",
		maxRetries:        3,
		retryDelay:        time.Minute,
		visibilityTimeout: 10 * time.Minute,
	}, nil
}

// WithTestAdapter sets a testing adapter seam for unit tests.
func (q *RedisQueue) WithTestAdapter(ad simpleAdapter) *RedisQueue {
	q.testAdapter = ad
	return q
}

// helpers to call either adapter or circuit breaker client
func (q *RedisQueue) lpush(ctx context.Context, key string, values ...interface{}) cmdErr {
	if q.testAdapter != nil {
		return q.testAdapter.LPush(ctx, key, values...)
	}
	return q.circuitClient.LPush(ctx, key, values...)
}

func (q *RedisQueue) zadd(ctx context.Context, key string, members ...*redis.Z) cmdErr {
	if q.testAdapter != nil {
		return q.testAdapter.ZAdd(ctx, key, members...)
	}
	return q.circuitClient.ZAdd(ctx, key, members...)
}

func (q *RedisQueue) del(ctx context.Context, keys ...string) cmdErr {
	if q.testAdapter != nil {
		return q.testAdapter.Del(ctx, keys...)
	}
	return q.circuitClient.Del(ctx, keys...)
}

func (q *RedisQueue) zrem(ctx context.Context, key string, members ...interface{}) cmdErr {
	if q.testAdapter != nil {
		return q.testAdapter.ZRem(ctx, key, members...)
	}
	return q.circuitClient.ZRem(ctx, key, members...)
}

// Enqueue adds a job to the queue
func (q *RedisQueue) Enqueue(ctx context.Context, jobSpecID string, action string, payload map[string]interface{}) error {
	tracer := otel.Tracer("runner/queue/redis")
	ctx, span := tracer.Start(ctx, "RedisQueue.Enqueue", trace.WithAttributes(
		attribute.String("queue.name", q.queueName),
		attribute.String("queue.retry", q.retryQueue),
		attribute.String("queue.dead", q.deadQueue),
		attribute.String("jobspec.id", jobSpecID),
		attribute.String("action", action),
	))
	defer span.End()
	message := &JobMessage{
		ID:         fmt.Sprintf("%s:%d", jobSpecID, time.Now().UnixNano()),
		JobSpecID:  jobSpecID,
		Action:     action,
		Payload:    payload,
		Attempts:   0,
		MaxRetries: q.maxRetries,
		EnqueuedAt: time.Now(),
	}

	messageJSON, err := json.Marshal(message)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to marshal job message: %w", err)
	}

	// Add to main queue
	if err := q.client.LPush(ctx, q.queueName, messageJSON).Err(); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	log.Printf("Enqueued job %s for JobSpec %s", message.ID, jobSpecID)
	span.SetAttributes(
		attribute.String("job.id", message.ID),
	)
	return nil
}

// Dequeue retrieves and processes a job from the queue
func (q *RedisQueue) Dequeue(ctx context.Context) (*JobMessage, error) {
	tracer := otel.Tracer("runner/queue/redis")
	ctx, span := tracer.Start(ctx, "RedisQueue.Dequeue", trace.WithAttributes(
		attribute.String("queue.name", q.queueName),
	))
	defer span.End()
	// Try main queue first using circuit breaker
	result, err := q.circuitClient.BRPop(ctx, 1*time.Second, q.queueName).Result()
	if err != nil {
		if err == redis.Nil {
			// Try retry queue
			span.AddEvent("main_queue_empty_try_retry")
			return q.dequeueRetry(ctx)
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to dequeue from main queue: %w", err)
	}

	if len(result) < 2 {
		span.RecordError(fmt.Errorf("invalid queue result"))
		return nil, fmt.Errorf("invalid queue result")
	}

	var message JobMessage
	if err := json.Unmarshal([]byte(result[1]), &message); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to unmarshal job message: %w", err)
	}

	// Mark as processing by moving to processing set with expiration
	processingKey := fmt.Sprintf("%s:processing:%s", q.queueName, message.ID)
	messageJSON, _ := json.Marshal(message)
	
	if err := q.client.SetEX(ctx, processingKey, messageJSON, q.visibilityTimeout).Err(); err != nil {
		log.Printf("Warning: failed to mark job %s as processing: %v", message.ID, err)
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

// dequeueRetry attempts to dequeue from the retry queue
func (q *RedisQueue) dequeueRetry(ctx context.Context) (*JobMessage, error) {
	tracer := otel.Tracer("runner/queue/redis")
	ctx, span := tracer.Start(ctx, "RedisQueue.dequeueRetry", trace.WithAttributes(
		attribute.String("queue.retry", q.retryQueue),
	))
	defer span.End()
	// Check for jobs ready to retry
	now := time.Now()
	
	// Get jobs from retry queue that are ready to be retried
	results, err := q.circuitClient.ZRangeByScore(ctx, q.retryQueue, &redis.ZRangeBy{
		Min: "0",
		Max: fmt.Sprintf("%d", now.Unix()),
		Count: 1,
	}).Result()

	if err != nil || len(results) == 0 {
		if err != nil { span.RecordError(err) }
		return nil, nil // No jobs ready for retry
	}

	// Remove from retry queue
	if err := q.zrem(ctx, q.retryQueue, results[0]).Err(); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to remove job from retry queue: %w", err)
	}

	var message JobMessage
	if err := json.Unmarshal([]byte(results[0]), &message); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to unmarshal retry job message: %w", err)
	}

	// Mark as processing
	processingKey := fmt.Sprintf("%s:processing:%s", q.queueName, message.ID)
	messageJSON, _ := json.Marshal(message)
	
	if err := q.client.SetEX(ctx, processingKey, messageJSON, q.visibilityTimeout).Err(); err != nil {
		log.Printf("Warning: failed to mark retry job %s as processing: %v", message.ID, err)
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

// Complete marks a job as successfully completed
func (q *RedisQueue) Complete(ctx context.Context, message *JobMessage) error {
	tracer := otel.Tracer("runner/queue/redis")
	ctx, span := tracer.Start(ctx, "RedisQueue.Complete", trace.WithAttributes(
		attribute.String("queue.name", q.queueName),
		attribute.String("job.id", message.ID),
		attribute.String("jobspec.id", message.JobSpecID),
		attribute.Int("job.attempts", message.Attempts),
	))
	defer span.End()
	processingKey := fmt.Sprintf("%s:processing:%s", q.queueName, message.ID)
	
	// Remove from processing set
	if err := q.client.Del(ctx, processingKey).Err(); err != nil {
		log.Printf("Warning: failed to remove completed job %s from processing: %v", message.ID, err)
	}

	log.Printf("Completed job %s for JobSpec %s", message.ID, message.JobSpecID)
	return nil
}

// Fail handles job failure with retry logic
func (q *RedisQueue) Fail(ctx context.Context, message *JobMessage, jobError error) error {
	tracer := otel.Tracer("runner/queue/redis")
	ctx, span := tracer.Start(ctx, "RedisQueue.Fail", trace.WithAttributes(
		attribute.String("queue.name", q.queueName),
		attribute.String("queue.retry", q.retryQueue),
		attribute.String("queue.dead", q.deadQueue),
		attribute.String("job.id", message.ID),
		attribute.String("jobspec.id", message.JobSpecID),
		attribute.Int("job.attempts", message.Attempts),
		attribute.Int("job.max_retries", message.MaxRetries),
	))
	defer span.End()
	processingKey := fmt.Sprintf("%s:processing:%s", q.queueName, message.ID)
	
	// Remove from processing set
	if err := q.del(ctx, processingKey).Err(); err != nil {
		log.Printf("Warning: failed to remove failed job %s from processing: %v", message.ID, err)
	}

	message.Error = jobError.Error()
	span.RecordError(jobError)

	// Check if we should retry
	if message.Attempts < message.MaxRetries {
		// Calculate retry delay with exponential backoff
		retryDelay := q.retryDelay * time.Duration(message.Attempts)
		retryTime := time.Now().Add(retryDelay)

		messageJSON, err := json.Marshal(message)
		if err != nil {
			span.RecordError(err)
			return fmt.Errorf("failed to marshal retry message: %w", err)
		}

		// Add to retry queue with score as retry timestamp
		if err := q.zadd(ctx, q.retryQueue, &redis.Z{
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
	messageJSON, err := json.Marshal(message)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to marshal dead message: %w", err)
	}

	if err := q.lpush(ctx, q.deadQueue, messageJSON).Err(); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to add job to dead queue: %w", err)
	}

	log.Printf("Job %s moved to dead queue after %d attempts: %v", 
		message.ID, message.Attempts, jobError)
	span.SetAttributes(attribute.Bool("job.dead_letter", true))
	return nil
}

// GetQueueStats returns statistics about the queue
func (q *RedisQueue) GetQueueStats(ctx context.Context) (map[string]int64, error) {
	tracer := otel.Tracer("runner/queue/redis")
	ctx, span := tracer.Start(ctx, "RedisQueue.GetQueueStats", trace.WithAttributes(
		attribute.String("queue.name", q.queueName),
		attribute.String("queue.retry", q.retryQueue),
		attribute.String("queue.dead", q.deadQueue),
	))
	defer span.End()
	pipe := q.client.Pipeline()
	
	mainLen := pipe.LLen(ctx, q.queueName)
	retryLen := pipe.ZCard(ctx, q.retryQueue)
	deadLen := pipe.LLen(ctx, q.deadQueue)
	processingLen := pipe.Keys(ctx, fmt.Sprintf("%s:processing:*", q.queueName))

	_, err := pipe.Exec(ctx)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get queue stats: %w", err)
	}

	processingCount := int64(len(processingLen.Val()))

	span.SetAttributes(
		attribute.Int64("stats.main", mainLen.Val()),
		attribute.Int64("stats.retry", retryLen.Val()),
		attribute.Int64("stats.dead", deadLen.Val()),
		attribute.Int64("stats.processing", processingCount),
	)
	return map[string]int64{
		"main":       mainLen.Val(),
		"retry":      retryLen.Val(),
		"dead":       deadLen.Val(),
		"processing": processingCount,
	}, nil
}

// Close closes the Redis connection
func (q *RedisQueue) Close() error {
	return q.client.Close()
}

// RecoverStaleJobs recovers jobs that have been processing too long
func (q *RedisQueue) RecoverStaleJobs(ctx context.Context) error {
	tracer := otel.Tracer("runner/queue/redis")
	ctx, span := tracer.Start(ctx, "RedisQueue.RecoverStaleJobs", trace.WithAttributes(
		attribute.String("queue.name", q.queueName),
	))
	defer span.End()
	pattern := fmt.Sprintf("%s:processing:*", q.queueName)
	keys, err := q.client.Keys(ctx, pattern).Result()
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to get processing keys: %w", err)
	}

	recovered := 0
	for _, key := range keys {
		// Check if key has expired (TTL <= 0)
		ttl, err := q.client.TTL(ctx, key).Result()
		if err != nil {
			continue
		}

		if ttl <= 0 {
			// Job has expired, recover it
			messageJSON, err := q.client.Get(ctx, key).Result()
			if err != nil {
				continue
			}

			var message JobMessage
			if err := json.Unmarshal([]byte(messageJSON), &message); err != nil {
				continue
			}

			// Remove from processing
			q.client.Del(ctx, key)

			// Re-queue for retry
			if err := q.Fail(ctx, &message, fmt.Errorf("job processing timeout")); err != nil {
				log.Printf("Failed to recover stale job %s: %v", message.ID, err)
				continue
			}

			recovered++
			span.AddEvent("recovered_stale_job", trace.WithAttributes(
				attribute.String("job.id", message.ID),
				attribute.String("jobspec.id", message.JobSpecID),
			))
		}
	}

	if recovered > 0 {
		log.Printf("Recovered %d stale jobs", recovered)
	}

	span.SetAttributes(attribute.Int("recovered.count", recovered))
	return nil
}

// GetCircuitBreakerStats returns circuit breaker statistics
func (q *RedisQueue) GetCircuitBreakerStats() string {
	if q.circuitClient == nil {
		return "Circuit breaker not initialized"
	}
	stats := q.circuitClient.Stats()
	return stats.String()
}

// LogCircuitBreakerStats logs current circuit breaker statistics
func (q *RedisQueue) LogCircuitBreakerStats() {
	if q.circuitClient != nil {
		q.circuitClient.LogStats()
	}
}
