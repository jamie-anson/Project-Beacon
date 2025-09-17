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

// RedisQueue provides Redis-based job queuing with retry logic
type RedisQueue struct {
	client            *redis.Client
	circuitClient     *RedisCircuitBreaker
	queueName         string
	retryQueue        string
	deadQueue         string
	maxRetries        int
	retryDelay        time.Duration
	visibilityTimeout time.Duration
	
	// Composed components
	retryHandler      *RetryHandler
	processingTracker *ProcessingTracker
	statsCollector    *StatsCollector
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

	// Create composed components
	retryHandler := NewRetryHandler(queueName, client, circuitClient)
	processingTracker := NewProcessingTracker(queueName, client, retryHandler)
	statsCollector := NewStatsCollector(queueName, client, processingTracker)

	return &RedisQueue{
		client:            client,
		circuitClient:     circuitClient,
		queueName:         queueName,
		retryQueue:        queueName + ":retry",
		deadQueue:         queueName + ":dead",
		maxRetries:        3,
		retryDelay:        time.Minute,
		visibilityTimeout: 10 * time.Minute,
		retryHandler:      retryHandler,
		processingTracker: processingTracker,
		statsCollector:    statsCollector,
	}, nil
}

// WithTestAdapter sets a testing adapter seam for unit tests.
func (q *RedisQueue) WithTestAdapter(ad simpleAdapter) *RedisQueue {
	q.retryHandler.WithTestAdapter(ad)
	return q
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
			return q.retryHandler.DequeueRetry(ctx)
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
		// Fallback: outbox-published raw envelope {"id","enqueued_at","attempt",...}
		var env map[string]interface{}
		if jerr := json.Unmarshal([]byte(result[1]), &env); jerr == nil {
			if idVal, ok := env["id"].(string); ok && idVal != "" {
				// Attempt to extract fields from envelope
				attempts := 0
				if a, ok := env["attempt"].(float64); ok {
					attempts = int(a)
				}
				enq := time.Now()
				if s, ok := env["enqueued_at"].(string); ok && s != "" {
					if t, perr := time.Parse(time.RFC3339Nano, s); perr == nil {
						enq = t
					} else if t2, perr2 := time.Parse(time.RFC3339, s); perr2 == nil {
						enq = t2
					}
				}
				// Construct minimal JobMessage (Action/Payload unused by JobRunner path)
				message = JobMessage{
					ID:         fmt.Sprintf("env:%s:%d", idVal, time.Now().UnixNano()),
					JobSpecID:  idVal,
					Attempts:   attempts,
					MaxRetries: 3,
					EnqueuedAt: enq,
				}
			} else {
				span.RecordError(err)
				return nil, fmt.Errorf("failed to unmarshal job message: %w", err)
			}
		} else {
			span.RecordError(err)
			return nil, fmt.Errorf("failed to unmarshal job message: %w", err)
		}
	}

	message.Attempts++
	message.LastAttempt = time.Now()

	// Mark as processing using processing tracker
	if err := q.processingTracker.MarkAsProcessing(ctx, &message); err != nil {
		log.Printf("Warning: failed to mark job %s as processing: %v", message.ID, err)
	}

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
	
	// Remove from processing set using processing tracker
	return q.processingTracker.MarkAsCompleted(ctx, message)
}

// Fail handles job failure with retry logic
func (q *RedisQueue) Fail(ctx context.Context, message *JobMessage, jobError error) error {
	// Remove from processing set first
	if err := q.processingTracker.MarkAsFailed(ctx, message); err != nil {
		log.Printf("Warning: failed to remove failed job %s from processing: %v", message.ID, err)
	}

	// Handle retry logic using retry handler
	return q.retryHandler.HandleFailure(ctx, message, jobError)
}

// GetQueueStats returns statistics about the queue
func (q *RedisQueue) GetQueueStats(ctx context.Context) (map[string]int64, error) {
	return q.statsCollector.GetQueueStats(ctx)
}

// Close closes the Redis connection
func (q *RedisQueue) Close() error {
	return q.client.Close()
}

// RecoverStaleJobs recovers jobs that have been processing too long
func (q *RedisQueue) RecoverStaleJobs(ctx context.Context) error {
	return q.processingTracker.RecoverStaleJobs(ctx)
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
