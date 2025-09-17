package queue

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client wraps a Redis client for simple queue operations
type Client struct {
	redis *redis.Client
	advancedQueue advQueue
}

// tryDequeueSimpleOnce attempts a single BRPOP with a short timeout and returns a raw payload if available.
func (c *Client) tryDequeueSimpleOnce(ctx context.Context, queue string, timeout time.Duration) ([]byte, error) {
    if c == nil || c.redis == nil {
        return nil, nil
    }
    res, err := c.redis.BRPop(ctx, timeout, queue).Result()
    if err != nil {
        if err == redis.Nil {
            return nil, nil
        }
        return nil, err
    }
    if len(res) != 2 {
        return nil, nil
    }
    return []byte(res[1]), nil
}

// GetRedisClient returns the underlying Redis client for security features
func (c *Client) GetRedisClient() *redis.Client {
	if c == nil {
		return nil
	}
	return c.redis
}

// advQueue is the minimal interface used by StartWorker for the advanced queue
type advQueue interface {
    Dequeue(ctx context.Context) (*JobMessage, error)
    Fail(ctx context.Context, message *JobMessage, jobError error) error
    Complete(ctx context.Context, message *JobMessage) error
    RecoverStaleJobs(ctx context.Context) error
    Close() error
}

// newAdvancedQueue is an overridable factory for tests
var newAdvancedQueue = func(redisURL, queueName string) (advQueue, error) {
    return NewRedisQueue(redisURL, queueName)
}

// Ping checks connectivity to Redis
func (c *Client) Ping(ctx context.Context) error {
    if c == nil || c.redis == nil {
        return nil
    }
    return c.redis.Ping(ctx).Err()
}

// Close closes the underlying Redis client
func (c *Client) Close() error {
    if c == nil || c.redis == nil {
        return nil
    }
    return c.redis.Close()
}

// GetCircuitBreakerStats returns circuit breaker statistics if available
func (c *Client) GetCircuitBreakerStats() string {
	if c == nil || c.advancedQueue == nil {
		return "Circuit breaker not available"
	}
	
	if rq, ok := c.advancedQueue.(*RedisQueue); ok {
		return rq.GetCircuitBreakerStats()
	}
	
	return "Circuit breaker not available for this queue type"
}

// NewFromEnv initializes a Redis client using REDIS_URL env var or defaults
func NewFromEnv() (*Client, error) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		url = "redis://localhost:6379"
	}
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	cl := redis.NewClient(opt)
	return &Client{redis: cl}, nil
}

// MustNewFromEnv panics on error; suited for worker startup
func MustNewFromEnv() *Client {
	c, err := NewFromEnv()
	if err != nil {
		log.Fatalf("failed to init redis: %v", err)
	}
	return c
}

// Enqueue pushes a payload onto a list (acts as a FIFO queue)
func (c *Client) Enqueue(ctx context.Context, queue string, payload []byte) error {
	return c.redis.RPush(ctx, queue, payload).Err()
}

// StartWorker starts a blocking consumer loop on a queue using BRPOP with retry support
func (c *Client) StartWorker(ctx context.Context, queueName string, handler func([]byte) error) {
	// Initialize advanced queue for retry/dead-letter support
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}
	
	advancedQueue, err := newAdvancedQueue(redisURL, queueName)
	if err != nil {
		log.Printf("Failed to create advanced queue, falling back to simple mode: %v", err)
		c.startSimpleWorker(ctx, queueName, handler)
		return
	}
	defer advancedQueue.Close()
	
	// Store the advanced queue for circuit breaker stats access
	c.advancedQueue = advancedQueue

	log.Printf("queue worker started for '%s' with retry support", queueName)
	
	// Start stale job recovery routine
	if rq, ok := advancedQueue.(*RedisQueue); ok {
		go c.startStaleJobRecovery(ctx, rq)
	}
	
	for {
		select {
		case <-ctx.Done():
			log.Printf("queue worker stopping for '%s'", queueName)
			return
		default:
		}

		// Dequeue with retry support
		message, err := advancedQueue.Dequeue(ctx)
		if err != nil {
			log.Printf("queue dequeue error: %v", err)
			// Fallback: try to read a raw envelope once from the simple queue
			if payload, ferr := c.tryDequeueSimpleOnce(ctx, queueName, 500*time.Millisecond); ferr == nil && payload != nil {
				if hErr := handler(payload); hErr != nil {
					log.Printf("queue handler error (fallback simple): %v", hErr)
				}
				continue
			}
			time.Sleep(500 * time.Millisecond)
			continue
		}
		
		if message == nil {
			// Also attempt a non-blocking simple dequeue to pick up raw envelopes
			if payload, _ := c.tryDequeueSimpleOnce(ctx, queueName, 200*time.Millisecond); payload != nil {
				if hErr := handler(payload); hErr != nil {
					log.Printf("queue handler error (fallback simple no-msg): %v", hErr)
				}
			}
			continue // No advanced message available
		}

		// Process the job
		var payload []byte
		if message.Payload != nil {
			// For new-style messages with structured payload
			payloadJSON, _ := json.Marshal(message.Payload)
			payload = payloadJSON
		} else {
			// For simple envelope messages (current format). Some producers publish only {id,enqueued_at,attempt}.
			// Prefer JobSpecID, but fall back to message.ID when JobSpecID is empty (raw outbox envelope case).
			jobID := message.JobSpecID
			if jobID == "" {
				jobID = message.ID
			}
			envelope := map[string]interface{}{
				"id":          jobID,
				"enqueued_at": message.EnqueuedAt,
				"attempt":     message.Attempts,
			}
			payload, _ = json.Marshal(envelope)
		}

		if err := handler(payload); err != nil {
			log.Printf("queue handler error for job %s: %v", message.ID, err)
			if failErr := advancedQueue.Fail(ctx, message, err); failErr != nil {
				log.Printf("failed to handle job failure: %v", failErr)
			}
		} else {
			if completeErr := advancedQueue.Complete(ctx, message); completeErr != nil {
				log.Printf("failed to mark job as complete: %v", completeErr)
			}
		}
	}
}

// startSimpleWorker provides fallback to simple BRPOP behavior
func (c *Client) startSimpleWorker(ctx context.Context, queue string, handler func([]byte) error) {
	log.Printf("queue worker started for '%s' (simple mode)", queue)
	for {
		select {
		case <-ctx.Done():
			log.Printf("queue worker stopping for '%s'", queue)
			return
		default:
		}

		// Block for up to 5 seconds waiting for an item
		res, err := c.redis.BRPop(ctx, 5*time.Second, queue).Result()
		if err == redis.Nil {
			continue // timeout, loop again
		}
		if err != nil {
			// transient error
			log.Printf("queue BRPOP error: %v", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		if len(res) != 2 {
			continue
		}
		payload := []byte(res[1])
		if err := handler(payload); err != nil {
			log.Printf("queue handler error: %v", err)
		}
	}
}

// startStaleJobRecovery runs periodic recovery of stale processing jobs
func (c *Client) startStaleJobRecovery(ctx context.Context, queue *RedisQueue) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := queue.RecoverStaleJobs(ctx); err != nil {
				log.Printf("stale job recovery error: %v", err)
			}
		}
	}
}
