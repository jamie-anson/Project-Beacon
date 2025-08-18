package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisQueue provides Redis-based job queuing with retry logic
type RedisQueue struct {
	client        *redis.Client
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

	return &RedisQueue{
		client:            client,
		queueName:         queueName,
		retryQueue:        queueName + ":retry",
		deadQueue:         queueName + ":dead",
		maxRetries:        3,
		retryDelay:        time.Minute,
		visibilityTimeout: 10 * time.Minute,
	}, nil
}

// Enqueue adds a job to the queue
func (q *RedisQueue) Enqueue(ctx context.Context, jobSpecID string, action string, payload map[string]interface{}) error {
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
		return fmt.Errorf("failed to marshal job message: %w", err)
	}

	// Add to main queue
	if err := q.client.LPush(ctx, q.queueName, messageJSON).Err(); err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	log.Printf("Enqueued job %s for JobSpec %s", message.ID, jobSpecID)
	return nil
}

// Dequeue retrieves and processes a job from the queue
func (q *RedisQueue) Dequeue(ctx context.Context) (*JobMessage, error) {
	// Try main queue first
	result, err := q.client.BRPop(ctx, 1*time.Second, q.queueName).Result()
	if err != nil {
		if err == redis.Nil {
			// Try retry queue
			return q.dequeueRetry(ctx)
		}
		return nil, fmt.Errorf("failed to dequeue from main queue: %w", err)
	}

	if len(result) < 2 {
		return nil, fmt.Errorf("invalid queue result")
	}

	var message JobMessage
	if err := json.Unmarshal([]byte(result[1]), &message); err != nil {
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

	return &message, nil
}

// dequeueRetry attempts to dequeue from the retry queue
func (q *RedisQueue) dequeueRetry(ctx context.Context) (*JobMessage, error) {
	// Check for jobs ready to retry
	now := time.Now()
	
	// Get jobs from retry queue that are ready to be retried
	results, err := q.client.ZRangeByScore(ctx, q.retryQueue, &redis.ZRangeBy{
		Min: "0",
		Max: fmt.Sprintf("%d", now.Unix()),
		Count: 1,
	}).Result()

	if err != nil || len(results) == 0 {
		return nil, nil // No jobs ready for retry
	}

	// Remove from retry queue
	if err := q.client.ZRem(ctx, q.retryQueue, results[0]).Err(); err != nil {
		return nil, fmt.Errorf("failed to remove job from retry queue: %w", err)
	}

	var message JobMessage
	if err := json.Unmarshal([]byte(results[0]), &message); err != nil {
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

	return &message, nil
}

// Complete marks a job as successfully completed
func (q *RedisQueue) Complete(ctx context.Context, message *JobMessage) error {
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
	processingKey := fmt.Sprintf("%s:processing:%s", q.queueName, message.ID)
	
	// Remove from processing set
	if err := q.client.Del(ctx, processingKey).Err(); err != nil {
		log.Printf("Warning: failed to remove failed job %s from processing: %v", message.ID, err)
	}

	message.Error = jobError.Error()

	// Check if we should retry
	if message.Attempts < message.MaxRetries {
		// Calculate retry delay with exponential backoff
		retryDelay := q.retryDelay * time.Duration(message.Attempts)
		retryTime := time.Now().Add(retryDelay)

		messageJSON, err := json.Marshal(message)
		if err != nil {
			return fmt.Errorf("failed to marshal retry message: %w", err)
		}

		// Add to retry queue with score as retry timestamp
		if err := q.client.ZAdd(ctx, q.retryQueue, &redis.Z{
			Score:  float64(retryTime.Unix()),
			Member: messageJSON,
		}).Err(); err != nil {
			return fmt.Errorf("failed to add job to retry queue: %w", err)
		}

		log.Printf("Job %s failed (attempt %d/%d), scheduled for retry at %v: %v", 
			message.ID, message.Attempts, message.MaxRetries, retryTime, jobError)
		return nil
	}

	// Max retries exceeded, move to dead letter queue
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal dead message: %w", err)
	}

	if err := q.client.LPush(ctx, q.deadQueue, messageJSON).Err(); err != nil {
		return fmt.Errorf("failed to add job to dead queue: %w", err)
	}

	log.Printf("Job %s moved to dead queue after %d attempts: %v", 
		message.ID, message.Attempts, jobError)
	return nil
}

// GetQueueStats returns statistics about the queue
func (q *RedisQueue) GetQueueStats(ctx context.Context) (map[string]int64, error) {
	pipe := q.client.Pipeline()
	
	mainLen := pipe.LLen(ctx, q.queueName)
	retryLen := pipe.ZCard(ctx, q.retryQueue)
	deadLen := pipe.LLen(ctx, q.deadQueue)
	processingLen := pipe.Keys(ctx, fmt.Sprintf("%s:processing:*", q.queueName))

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue stats: %w", err)
	}

	processingCount := int64(len(processingLen.Val()))

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
	pattern := fmt.Sprintf("%s:processing:*", q.queueName)
	keys, err := q.client.Keys(ctx, pattern).Result()
	if err != nil {
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
		}
	}

	if recovered > 0 {
		log.Printf("Recovered %d stale jobs", recovered)
	}

	return nil
}
