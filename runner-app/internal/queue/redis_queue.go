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
    if q.retryHandler == nil {
        // Initialize a minimal retry handler for tests that construct RedisQueue via literal
        rh := &RetryHandler{
            queueName:  q.queueName,
            retryQueue: q.retryQueue,
            deadQueue:  q.deadQueue,
            maxRetries: q.maxRetries,
            retryDelay: q.retryDelay,
            client:     q.client,
            // circuitClient can be nil in tests; testAdapter path bypasses it
        }
        q.retryHandler = rh
    }
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
	// Try main queue first using circuit breaker when available; fallback to raw client in tests
	var result []string
	var err error
	if q.circuitClient != nil {
		result, err = q.circuitClient.BRPop(ctx, 1*time.Second, q.queueName).Result()
	} else {
		result, err = q.client.BRPop(ctx, 1*time.Second, q.queueName).Result()
	}
	if err != nil {
		if err == redis.Nil {
			// Try retry queue
			span.AddEvent("main_queue_empty_try_retry")
			if q.retryHandler == nil {
				if q.circuitClient == nil {
					q.circuitClient = NewRedisCircuitBreaker(q.client, "redis-queue-"+q.queueName)
				}
				q.retryHandler = NewRetryHandler(q.queueName, q.client, q.circuitClient)
			}
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

	// Mark as processing using processing tracker (nil-safe)
	if q.processingTracker == nil {
		q.processingTracker = NewProcessingTracker(q.queueName, q.client, q.retryHandler)
	}
	if q.processingTracker != nil {
		if err := q.processingTracker.MarkAsProcessing(ctx, &message); err != nil {
			log.Printf("Warning: failed to mark job %s as processing: %v", message.ID, err)
		}
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
	
	// Remove from processing set using processing tracker (if available)
	if q.processingTracker != nil {
		return q.processingTracker.MarkAsCompleted(ctx, message)
	}
	// Fallback: best-effort direct DEL of processing key
	if q.client != nil {
		pkey := q.queueName + ":processing:" + message.ID
		_ = q.client.Del(ctx, pkey).Err()
	}
	return nil
}

// Fail handles job failure with retry logic
func (q *RedisQueue) Fail(ctx context.Context, message *JobMessage, jobError error) error {
    // Remove from processing set first
    if q.processingTracker != nil {
        if err := q.processingTracker.MarkAsFailed(ctx, message); err != nil {
            log.Printf("Warning: failed to remove failed job %s from processing: %v", message.ID, err)
        }
    }

    // Handle retry logic using retry handler
    if q.retryHandler == nil {
        // Ensure circuit breaker exists for retry operations
        if q.circuitClient == nil {
            q.circuitClient = NewRedisCircuitBreaker(q.client, "redis-queue-"+q.queueName)
        }
        q.retryHandler = NewRetryHandler(q.queueName, q.client, q.circuitClient)
        // Use queue's retry delay and max retries if set
        if q.retryDelay > 0 {
            q.retryHandler.retryDelay = q.retryDelay
        }
        if q.maxRetries > 0 {
            q.retryHandler.maxRetries = q.maxRetries
        }
    }
    
    // Schedule retry or move to dead-letter queue
    if err := q.retryHandler.HandleFailure(ctx, message, jobError); err != nil {
        return err
    }
    
    // Delete processing key after scheduling retry/dead-letter (matches Complete() behavior)
    processingKey := fmt.Sprintf("%s:processing:%s", q.queueName, message.ID)
    if delErr := q.retryHandler.del(ctx, processingKey).Err(); delErr != nil {
        log.Printf("Warning: failed to delete processing key %s: %v", processingKey, delErr)
    }
    
    return nil
}

// GetQueueStats returns statistics about the queue
func (q *RedisQueue) GetQueueStats(ctx context.Context) (map[string]int64, error) {
    if q.statsCollector == nil {
        q.statsCollector = NewStatsCollector(q.queueName, q.client, q.processingTracker)
    }
    // Ensure processing tracker is available for stats even if queue was constructed without it
    if q.statsCollector != nil && q.statsCollector.processingTracker == nil {
        q.statsCollector.processingTracker = NewProcessingTracker(q.queueName, q.client, q.retryHandler)
    }
    return q.statsCollector.GetQueueStats(ctx)
}

// Close closes the Redis connection
func (q *RedisQueue) Close() error {
	return q.client.Close()
}

// RecoverStaleJobs recovers jobs that have been processing too long
func (q *RedisQueue) RecoverStaleJobs(ctx context.Context) error {
	if q.processingTracker == nil {
		q.processingTracker = NewProcessingTracker(q.queueName, q.client, q.retryHandler)
	}
	if q.processingTracker == nil {
		return nil
	}
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
